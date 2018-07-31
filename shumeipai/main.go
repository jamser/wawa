package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"time"

	deviceMsg "shumeipai/msg"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
)

//SendDataStruct 发送给娃娃机的数据结构
/*
***注1：2字节及以上参数，先发送最高字节，最后发送最低字节；接收数据相同，先接收最高字节，最后接收最低字节。
***注2：参数长度为0时，无参数。
 */
type SendDataStruct struct {
	Head  [2]byte //帧头， 2字节，  命令标识 0xD5F1
	Len   byte    //参数长度 1字节
	Cmd   byte    //功能（命令）1字节
	Data  []byte  //参数 Len字节
	Check byte    //验证位 Cmd、Data和0x55异或。 即	Check = Cmd xor Data[0] xor Data[1] xor … xor Data[n-1] xor 0x55
}

const (
	//ButtonRight 右按钮
	ButtonRight = 0x1
	//ButtonLeft 左按钮
	ButtonLeft = 0x2
	//ButtonDown 下按钮
	ButtonDown = 0x10
	//ButtonBack 后按钮
	ButtonBack = 0x40
	//ButtonFront 前按钮
	ButtonFront = 0x80
)

var (
	s *serial.Port
	//ButtonState 按钮状态
	ButtonState = 0
	mqtt        *MqttClient
	inputChan   = make(chan int) //1 代表 投币操作  2：代表按钮状态改变
	deviceInfo  (*deviceMsg.SCDeviceInfo)

	//MyLog log输出
	MyLog *log.Logger

	controlCount byte //控制基数！
)

func (sData *SendDataStruct) generateCheck() {
	sData.Check = sData.Cmd ^ 0x55
	if sData.Len > 0 {
		for i := 0; i < int(sData.Len); i++ {
			sData.Check ^= sData.Data[i]
		}
	}
}

/*
运行状态
0-3: 主板初始化
4-5: 测试接口
6-21: 故障或处理中
22:   设置操作
23-26: 天车初始化操作
27-32: 等待游戏开始
33-34: 天车运行中(客户操作中)
35-36：下抓中
37: 取物中
38-42：上提中
43: 释放礼品
44-47: 天车归位

2.	出物状态
B7	                   B6				B5…B4			B3…B0	备注
0：正在检测出物	  0：未中奖或检测中		待定 				  0	  等待
1：检测完成或未开始	 1：中奖							  	  1	  感应器故障
															2	正在检测
															3	中奖1
															4	中奖2
															5	其他

*/
//检测出物状态 0, 中奖 1  未中奖 2
func checkResult(running, outState byte) (state int) {

	if deviceInfo.State == 3 {
		if outState&0x80 != 0 && outState&0x40 != 0 {
			//中奖
			deviceInfo.State = 4
			MyLog.Println("恭喜中奖了！")
			updateDeviceInfo(nil)
			deviceInfo.State = 0
			return 1
		} else if outState&0x80 != 0 { //未中奖
			deviceInfo.State = 5
			MyLog.Println("没中奖！")
			updateDeviceInfo(nil)
			deviceInfo.State = 0
			return 2
		}
		return
	}

	if outState&0x80 == 0 {
		deviceInfo.State = 3 //设备中奖判定状态
		updateDeviceInfo(nil)
		MyLog.Println("开始检测中奖！")
		ButtonState = 0
		return
	}
	return
}

/*
a.	联机心跳
发送：Len=3，Cmd=0，Data=接口状态(状态,接口,其他)，3字节
返回：Len=4，Cmd=0，Data=返回状态，4字节
注：每1秒必须(至少)发送一次，主板在2秒内未收到心跳命令，自动按掉线处理。
发送数据：
状态: B7: 1=使用外部设备接口，即接口数据有效， 0=使用机器上摇杆按钮等；
B6: 1=未安装外部液晶屏， 0=使用液晶屏，即检查外部液晶屏有效；
其他=0：待定
接口: B0:右按钮
B1:左按钮
B4:下按钮
B6:后按钮
B7:前按钮
其他: 待扩充
返回数据：
Data[0]：运行状态；
Data[1]：出物状态；
Data[2]：游戏时间；
Data[3]：可玩次数。
*/

func headbeat() {
	defer func() {
		if err := recover(); err != nil {
			MyLog.Println("心跳包出错")
		}
	}()

	data := new(SendDataStruct)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Len = 3
	data.Cmd = 0
	data.Data = make([]byte, 3)
	data.Data[0] = 0xC0 //B6 B7 均为1
	data.Data[1] = byte(ButtonState)
	MyLog.Println("sending heart beat")

	buf := data.Serializable()

	buf = sendReceiveMsg(buf, 9)

	data.DeSerializable(buf)
	checkResult(data.Data[0], data.Data[1])

	MyLog.Printf("读取信息  运行状态 %d 出物状态： %d 可玩次数： %d  游戏时间： %d", data.Data[0], data.Data[1], data.Data[3], data.Data[2])
}

func move(dir int) {
	switch dir {
	case 1:
		ButtonState = ButtonRight
	case 2:
		ButtonState = ButtonLeft
	case 3:
		ButtonState = ButtonFront
	case 4:
		ButtonState = ButtonBack
	case 5:
		ButtonState = ButtonDown
	case 0:
		ButtonState = 0 //输入清零
	}
}

//Serializable 序列化数据
func (sData *SendDataStruct) Serializable() (buf []byte) {

	sData.generateCheck()

	buf = make([]byte, sData.Len+5)
	buf[0] = sData.Head[0]
	buf[1] = sData.Head[1]
	buf[2] = sData.Len
	buf[3] = sData.Cmd
	if sData.Len > 0 {
		for i := 0; i < int(sData.Len); i++ {
			buf[4+i] = sData.Data[i]
		}
	}
	buf[4+sData.Len] = sData.Check
	return
}

//DeSerializable 反序列化数据
func (sData *SendDataStruct) DeSerializable(buf []byte) {

	sData.Head[0] = buf[0]
	sData.Head[1] = buf[1]
	sData.Len = buf[2]
	sData.Cmd = buf[3]

	if sData.Len > 0 {
		sData.Data = make([]byte, sData.Len)
		for i := 0; i < int(sData.Len); i++ {
			sData.Data[i] = buf[4+i]
		}
	}
	sData.Check = buf[4+sData.Len]
}

func sendReceiveMsg(buf []byte, receiveLen int) (dataReceive []byte) {
	len2Send := len(buf)
	sendSize := 0
	for sendSize < len2Send {
		bufTemp := make([]byte, 8)
		for i := 0; i < len2Send-sendSize && i < 8; i++ {
			bufTemp[i] = buf[sendSize+i]
		}
		n, err := s.Write(bufTemp)
		if err != nil {
			MyLog.Println("sendReceiveMsg write err:", err)
		}
		sendSize += n
	}

	sendSize = 0
	dataReceive = make([]byte, receiveLen)
	for sendSize < receiveLen {
		bufTemp := make([]byte, 8)

		n, err := s.Read(bufTemp)
		if err != nil {
			MyLog.Println("sendReceiveMsg write err:", err)
		}
		for i := 0; i < receiveLen-sendSize && i < 8; i++ {
			dataReceive[sendSize+i] = bufTemp[i]
		}
		sendSize += n
	}
	return
}

func setPosition() {
	//重置出厂设置
	data := new(SendDataStruct)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Len = 0
	data.Cmd = 16

	buf := data.Serializable()
	MyLog.Println("send [postion] : ", buf)
	buf = sendReceiveMsg(buf, 5)
	MyLog.Println("received : ", buf)

	//获取当前设置
	data.Head = [2]byte{0xD5, 0xF1}
	data.Len = 0
	data.Cmd = 12

	buf = data.Serializable()
	buf = sendReceiveMsg(buf, 16)
	data.DeSerializable(buf)
	MyLog.Println("默认数据:", data.Data)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Cmd = 22

	data.Data[6] = 1
	data.Data[7] = 1
	data.Data[8] = 1 //1秒自动开始

	buf = data.Serializable()

	buf = sendReceiveMsg(buf, 16)
	//data.DeSerializable(buf)
	//MyLog.Println(buf)

	//默认抓力
	/*
		data.Head = [2]byte{0xD5, 0xF1}
		data.Len = 0
		data.Cmd = 14

		buf = data.Serializable()
		buf = sendReceiveMsg(buf, 8)
		data.DeSerializable(buf)
		MyLog.Println("默认抓力:", data.Data)
	*/
}

func payCoin() {
	data := new(SendDataStruct)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Len = 1
	data.Cmd = 41
	data.Data = make([]byte, 1)
	data.Data[0] = 1

	buf := data.Serializable()

	buf = sendReceiveMsg(buf, 6)
	data.DeSerializable(buf)
	if data.Data[0] == 1 {
		MyLog.Println("投币成功!")

		deviceInfo.State = 1
		updateDeviceInfo(nil)
	} else {

		MyLog.Println("投币失败!", data.Data[0])
	}
}

func coin(client MQTT.Client, msg MQTT.Message) {
	MyLog.Println("投币信息!")
	inputChan <- 1
}

func action(client MQTT.Client, msg MQTT.Message) {
	ac := new(deviceMsg.SCDeviceAction)
	err := ac.UnmarshalJSON(msg.Payload())
	if err != nil {
		MyLog.Println("action msg format error")
	}

	move(ac.Action)
	if ac.Action == 5 {
		//controlPower(1)
		deviceInfo.State = 2 //设备下抓
		updateDeviceInfo(nil)
	}
	inputChan <- 2
}

//ControlCount 设置娃娃机中奖几率
func updateInfo(client MQTT.Client, msg MQTT.Message) {
	MyLog.Println("ControlCount接收到:", msg.Payload())
	newDeviceInfo := new(deviceMsg.SCDeviceInfo)
	err := newDeviceInfo.UnmarshalJSON(msg.Payload())
	if err != nil {
		MyLog.Println("ControlCount msg format error")
	}
	controlCount = (byte)(newDeviceInfo.Force)
	deviceInfo = newDeviceInfo

	err = ioutil.WriteFile("config.json", msg.Payload(), 0644)
	if err != nil {
		MyLog.Println("写入最新配置信息出错！")
	}
	inputChan <- 3
}

func controlWawa() {
	//获取当前设置
	data := new(SendDataStruct)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Len = 0
	data.Cmd = 12

	buf := data.Serializable()
	buf = sendReceiveMsg(buf, 16)
	data.DeSerializable(buf)
	MyLog.Println("ControlCount默认数据:", data.Data)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Cmd = 22

	data.Data[2] = controlCount

	buf = data.Serializable()

	buf = sendReceiveMsg(buf, 16)
	MyLog.Println("ControlCount最新数据", buf)
}

/*
c.	外部支付强制控制抓力（后台干预）
发送 Len=1，Cmd=43，Data=种类，1字节
设备返回：Len=1，Cmd=43，Data=种类，1字节
仅在游戏开始后且下抓前有效，即当前状态=33-36
发送种类：
0：取消强制抓力
1：强抓力
2：弱抓力
其他：无效
返回种类：
0x80：成功取消强制抓力设置；
0x81：成功设置为强抓力；
0x82：成功设置为弱抓力；
0：设置无效，当前取消强制抓力设置；
1：设置无效，当前已经设置为强抓力；
2：设置无效，当前已经设置为弱抓力；
其他：无效
*/

func controlPower(power byte) {
	data := new(SendDataStruct)
	data.Head = [2]byte{0xD5, 0xF1}
	data.Len = 1
	data.Cmd = 43
	data.Data = make([]byte, 1)
	data.Data[0] = power

	buf := data.Serializable()

	buf = sendReceiveMsg(buf, 6)
	data.DeSerializable(buf)

	switch power {
	case 0:
		if data.Data[0] == 0x80 || data.Data[0] == 0 {
			MyLog.Println("成功取消强制抓力设置")
		}
	case 1:
		if data.Data[0] == 0x81 || data.Data[0] == 1 {
			MyLog.Println("成功设置为强抓力")
		}
	case 2:
		if data.Data[0] == 0x82 || data.Data[0] == 2 {
			MyLog.Println("成功设置为弱抓力")
		}
	default:
		MyLog.Println("无效设置")
	}
}

/*
{
	//设备状态  -1 设备下线 0： 设备可玩 1： 设备正在运行
	"state":0,
	//设备所属类型
	"group":0,
	//设备ID
	"id":"test001",
	//设备rtmp URL
	"rtmp":"www.baidu.com",
	//设备描述
	"des":"娃娃机1号"
}
*/

func parsePkgInfo(msg MQTT.Message) ([]byte, error) {
	retPkg := new(deviceMsg.SCPackageInfo)
	retPkg.UnmarshalJSON(msg.Payload())

	if retPkg.Error != "" {
		return []byte(""), errors.New("Request 返回数据结构出错！")
	}

	if retPkg.Error != "" {
		return []byte(""), errors.New(retPkg.Error)
	}
	return []byte(retPkg.Result), nil
}

func sendMsg(topic string, msg []byte) (MQTT.Message, error) {
	return mqtt.Request(topic, []byte(" "+string(msg)))
}

func updateDeviceInfo(callback func(MQTT.Message, error) error) error {
	buf, _ := deviceInfo.MarshalJSON()
	if callback != nil {
		return callback(sendMsg("MatchRoom/HD_DeviceUpdate", buf))
	}

	_, err := sendMsg("MatchRoom/HD_DeviceUpdate", buf)
	return err
}

func register(retMsg MQTT.Message, err error) error {
	if err == nil {
		buf, err := parsePkgInfo(retMsg)
		if err != nil {
			MyLog.Println("注册失败：", err.Error())
			return nil
		}

		registerRet := new(deviceMsg.SCDeviceRegRet)

		registerRet.UnmarshalJSON(buf)
		if registerRet.Success {
			MyLog.Println("设备注册成功")
		} else {
			MyLog.Println("注册失败：", registerRet.Destription)
			return nil
		}
	} else {
		MyLog.Println("注册失败！，程序退出！", err.Error())
		return nil
	}

	return err
}

func main() {

	// 定义一个文件
	fileName := "ll.log"
	logFile, err := os.Create(fileName)
	defer logFile.Close()
	if err != nil {
		log.Fatalln("open file error !")
	}
	// 创建一个日志对象
	MyLog = log.New(logFile, "[Debug]", log.LstdFlags)

	data, err := ioutil.ReadFile("config.json")

	if err != nil {
		MyLog.Println("配置文件丢失： config.json")
		return

	}
	deviceInfo = new(deviceMsg.SCDeviceInfo)
	err = deviceInfo.UnmarshalJSON(data)
	if err != nil {
		MyLog.Println("配置文件解析失败")
		return
	}
	mqtt = new(MqttClient)
	op := mqtt.GetDefaultOptions("tcp://47.96.100.91:3565")
	//op := mqtt.GetDefaultOptions("tcp://121.196.213.86:3563")

	bConnected := false
	op.SetConnectionLostHandler(func(client MQTT.Client, err error) {
		fmt.Println("连接丢失：", err.Error())
		bConnected = false
		go func() {
			t := time.NewTicker(3 * time.Second)
			defer t.Stop()

		Reconnect:
			for {
				select {
				case <-t.C:
					if bConnected {
						break Reconnect
					}
					fmt.Println("正在重连中。。。")
					mqtt.Connect(op)
				}
			}
		}()
	})
	op.SetOnConnectHandler(func(client MQTT.Client) {
		fmt.Println("成功连接到服务器！")
		bConnected = true
		updateDeviceInfo(register)
	})

	err = mqtt.Connect(op)
	if err != nil {
		MyLog.Println("连接服务器失败")
		return
	}

	mqtt.On("game/coin", coin)
	mqtt.On("game/action", action)
	mqtt.On("game/updateInfo", updateInfo)
	/*
		ttt := <-inputChan
		ttt++
	*/
	c := &serial.Config{Name: "/dev/ttyAMA0", Baud: 57600}
	MyLog.Println("开始连接串口设备！")
	s, err = serial.OpenPort(c)
	MyLog.Println("连接串口设备成功！")
	if err != nil {
		MyLog.Printf("串口打开失败:%s", err.Error())
		return
	}

	defer s.Close()

	setPosition()

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		select {
		case <-t.C:
			headbeat()
		}

		for {
			select {
			case inputCmd := <-inputChan:
				if inputCmd == 1 {
					payCoin()
				} else if inputCmd == 2 {
					headbeat()
				} else if inputCmd == 3 {
					controlWawa()
				}
			case <-t.C:
				headbeat()
			}
		}
	}()

	wait := make(chan os.Signal)
	signal.Notify(wait, os.Interrupt, os.Kill)
	ss := <-wait
	MyLog.Println("退出信号", ss)

}
