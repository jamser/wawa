//Package matchingroom 游戏匹配 房间模块
/**
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package matchingroom

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	matchMsg "zhuawawa/msg"

	"github.com/liangdas/mqant-modules/room"
	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"
	"github.com/liangdas/mqant/utils"
)

//Module 模块实例
var Module = func() module.Module {
	m := new(MatchRoom)
	return m
}

//MatchRoom 匹配房间 结构体
type MatchRoom struct {
	basemodule.BaseModule
	roomsMap *utils.BeeMap
	//room        *room.Room
	userInfoMap    *utils.BeeMap //key为 玩家ID， value 为用户信息cache
	deviceTableMap map[string]*Table
}

//GetType 模块类型
func (m *MatchRoom) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "MatchRoom"
}

//Version 模块版本
func (m *MatchRoom) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (m *MatchRoom) usableTable(table room.BaseTable) bool {
	return table.AllowJoin()
}
func (m *MatchRoom) newTable(module module.RPCModule, tableID int) (room.BaseTable, error) {
	table := NewTable(module, tableID)
	return table, nil
}

//OnInit 模块初始化
func (m *MatchRoom) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	m.roomsMap = utils.NewBeeMap()
	roomNum := int(settings.Settings["RoomNum"].(float64))
	for i := 0; i < roomNum; i++ {
		m.roomsMap.Set(i, room.NewRoom(m, i, m.newTable, m.usableTable))
	}
	//for k, room := range m.roomsMap.Items() {
	//	log.Info("room: %v ---%v ", k, room)
	//}

	//log.Info("0 room id is %d", m.roomsMap.Get(0).(*room.Room).RoomId())
	m.deviceTableMap = make(map[string]*Table)
	//m.room = room.NewRoom(m, m.gameID, m.newTable, m.usableTable)

	//用户信息cache，在房间内时不需要次次从lobby获取，又或者断线后重连也直接从这里获取头像昵称等信息，只有从房间退出时才清除该信息，匹配成功时获取信息后缓存
	m.userInfoMap = utils.NewBeeMap()

	m.GetServer().RegisterGO("HD_Join", m.join)       //加入table
	m.GetServer().RegisterGO("HD_Exit", m.exit)       //退出table
	m.GetServer().RegisterGO("HD_SitDown", m.sitdown) //坐下table

	m.GetServer().Register("HD_UserPay", m.coin)      //投币table
	m.GetServer().RegisterGO("HD_UserAction", m.move) //控制移动table

	m.GetServer().RegisterGO("HD_DeviceUpdate", m.updateDevice) //更新设备信息
	m.GetServer().RegisterGO("changeConfig", m.changeConfig)    //更新设备配置
	//m.GetServer().RegisterGO("HD_DeviceTimeUpdate", m.updateDeviceTime) //更新游戏时间信息
	m.GetServer().RegisterGO("HD_RoomChat", m.roomChat) //房间聊天信息

	m.GetServer().RegisterGO("HD_RoomSuccessList", m.getSuccessList) //当前设备成功抓取记录列表

	m.GetServer().RegisterGO("userDisConnect", m.userDisConnect) //断线处理

}

//Run 模块运行
func (m *MatchRoom) Run(closeSig chan bool) {
}

//OnDestroy 模块清理
func (m *MatchRoom) OnDestroy() {
	//一定别忘了关闭RPC
	m.GetServer().OnDestroy()
}

/*********************处理函数*****************************/
//林：使用自己的解析方式

//BuildBigRoomID 生成房间ID
func BuildBigRoomID(gameID, tableID, transactionID int) string {
	return fmt.Sprintf("BR:%d:%d:%d", gameID, tableID, transactionID)
}

//ParseBigRoomID 解析房间ID
func ParseBigRoomID(bigroomID string) (int, int, int, error) {
	s := strings.Split(bigroomID, ":")
	if len(s) != 4 {
		return -1, -1, 0, fmt.Errorf("The bigroomId data structure is incorrect")
	}
	gameID, error := strconv.Atoi(s[1])
	if error != nil {
		return -1, -1, 0, error
	}
	tableID, error := strconv.Atoi(s[2])
	if error != nil {
		return -1, -1, 0, error
	}
	transactionID, error := strconv.Atoi(s[3])
	if error != nil {
		return -1, -1, 0, error
	}
	return gameID, tableID, transactionID, nil
}

func (m *MatchRoom) roomChat(session gate.Session, msg []byte) (result bool, err string) {

	chatInfo := new(matchMsg.CSRoomChat)

	if errJSON := chatInfo.UnmarshalJSON(msg); errJSON != nil {
		log.Warning("聊天信息结构体有误： %s", errJSON.Error())
		return false, errJSON.Error()
	}

	deviceTable, bOK := m.deviceTableMap[chatInfo.DeviceID]

	if bOK {
		user := m.userInfoMap.Get(session.GetUserid())
		if user != nil {
			userInfo := user.(matchMsg.SCUserInfoRet)

			chatMsg := matchMsg.SCRoomChat{UserNick: userInfo.NickName, ChatInfo: chatInfo.ChatInfo}
			buf, _ := chatMsg.MarshalJSON()
			deviceTable.NotifyChat(buf)
			return true, ""
		}

		return false, "发言玩家信息获取失败！"
	}

	return false, "聊天出错！，用户不在该房间但向该房间玩家发出聊天信息！"
}

/*
func (m *MatchRoom) updateDeviceTime(session gate.Session, msg []byte) (result bool, err string) {
	deviceTimeInfo := new(matchMsg.SCDeviceTimeUpdate)
	errMsg := deviceTimeInfo.UnmarshalJSON(msg)

	if errMsg != nil {
		return false, "设备游戏时间更新失败！"
	}

	table, bOK := m.deviceTableMap[deviceTimeInfo.DeviceID]

	if bOK {
		if deviceTimeInfo.TimeCountDown == 255 {
			//开始中奖检测，通知当前游戏玩家，隐藏操作UI，显示检测UI，table进入结算状态
			table.fsm.Call(SettlementPeriodEvent)
		} else {
			table.NotifyTime(msg)
		}
	} else {
		log.Error("设备（%s）并未注册！", deviceTimeInfo.DeviceID)
	}

	return
}
*/

func (m *MatchRoom) updateDevice(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	deviceInfo := new(matchMsg.SCDeviceInfo)
	errMsg := deviceInfo.UnmarshalJSON(msg)

	retMsg := new(matchMsg.SCDeviceRegRet)

	if errMsg != nil {
		retMsg.Success = false
		retMsg.Destription = "设备信息结构体出错！"
		buf, _ := retMsg.MarshalJSON()
		return m.App.ProtocolMarshal(string(buf), "")
	}

	table, bOk := m.deviceTableMap[deviceInfo.DeviceID]

	if bOk {
		if table.Device.State != deviceInfo.State {
			table.Device.State = deviceInfo.State
		}
	} else {
		//创建新的table
		tableInfo, err := m.getUsableTable(deviceInfo.Group)
		if err != "" {
			log.Error("获取不到未使用的桌子：%s", err)
		}
		gameID, tableID, _, errID := ParseBigRoomID(tableInfo)
		if errID != nil {
			log.Error("匹配成功后获取不到正确的未使用的桌子：%s", errID.Error())
		}

		//注册或者更新设备信息
		newInfoStr, err := m.RpcInvoke("RedisDB", "devicegolive", msg)

		if err == "" {
			table = m.roomsMap.Get(gameID).(*room.Room).GetTable(tableID).(*Table)
			table.SetTableDeviceAndSession(deviceInfo, session)

			m.deviceTableMap[deviceInfo.DeviceID] = table
			session.Set("lastplace", m.GetType())
			session.Set("BigRoomID", tableInfo)
			session.Set("IsDevice", "true")

			session.SendNR("game/updateInfo", ([]byte)(newInfoStr.(string)))
			session.Push()
		} else {
			log.Warning("注册设备失败： %s！！！", err)
		}
	}

	//当状态为中奖结果4或者5时， 此处设备状态更新为可游戏状态（0）
	if deviceInfo.State == 2 {
		table.fsm.Call(CaughtPeriodEvent) //进入下抓状态
	} else if deviceInfo.State == 3 {
		//进入中奖判定状态
		//通知当前玩家
		if table.CurPlayer.Session() != nil {
			table.CurPlayer.Session().SendNR("game/Judge", []byte(""))
		}
	} else if deviceInfo.State >= 4 {
		//状态改变 ， 通知table内所有玩家
		table.NotifyDeviceResult(deviceInfo.State)

		if deviceInfo.State == 4 { //记录最近成功抓住记录
			successInfo := new(matchMsg.SCSuccessRecord)
			successInfo.UserID = table.CurPlayer.UserID
			successInfo.UserNick = table.CurPlayer.UserInfo.NickName
			successInfo.UserHead = table.CurPlayer.UserInfo.HeadURL
			successInfo.Video = "以后需要获取的录像地址！"
			successInfo.DeviceID = table.Device.DeviceID
			buf, _ := successInfo.MarshalJSON()
			m.RpcInvokeNR("RedisDB", "recordSuccess", deviceInfo.DeviceID, buf)
		} else if deviceInfo.State == 5 { //抓取失败
			m.RpcInvokeNR("RedisDB", "recordPlay", deviceInfo.DeviceID)
		}

		deviceInfo.State = 0
		//进入结算期
		//log.Info("fsm state: OperationPeriod called")
		table.fsm.Call(SettlementPeriodEvent)
	}

	//设备更新发送到大厅
	tableMsg := new(matchMsg.SCTableInfoRet)
	tableMsg.Device = *deviceInfo
	tableMsg.TableCount = table.GetViewer().Len() + 1
	tableMsg.BigRoomID = BuildBigRoomID(deviceInfo.Group, table.TableId(), table.TransactionId())
	buf, _ := tableMsg.MarshalJSON()
	m.RpcInvokeNR("Lobby", "deviceupdate", buf)

	table.Device = deviceInfo //更新table中设备信息
	retMsg.Success = true
	retMsg.Destription = "设备更新成功！"
	buf, _ = retMsg.MarshalJSON()
	return m.App.ProtocolMarshal(string(buf), "")
}

func (m *MatchRoom) getUsableTable(gameID int) (tableRoomID, err string) {
	//table, errTable := m.room.GetUsableTable()
	table, errTable := m.roomsMap.Get(gameID).(*room.Room).GetUsableTable()
	if errTable == nil {
		table.Create()
		return BuildBigRoomID(gameID, table.TableId(), table.TransactionId()), ""
	}

	return "", "There is no available table"
}

//进入table
func (m *MatchRoom) join(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	//lin:客户端发送数据前会在最签名添加空格
	bigRoomID := string(msg[1:])

	gameID, tableID, _, errID := ParseBigRoomID(bigRoomID)
	if errID != nil {
		return m.App.ProtocolMarshal("", errID.Error())
	}

	table := m.roomsMap.Get(gameID).(*room.Room).GetTable(tableID)
	if table != nil {
		tableimp := table.(*Table)
		if table.VerifyAccessAuthority(session.GetUserid(), bigRoomID) == false {
			return m.App.ProtocolMarshal("", "加入房间失败：权限验证未通过")
		}

		//userData, errRPC := m.RpcInvoke("Lobby", "getUserData", session.GetUserid())
		userData, errRPC := m.RpcInvoke("RedisDB", "getUserInfo", session.GetUserid())
		/*进行json解析时，若以interface{}接收数据，则会按照下列规则进行解析：

		bool, for JSON booleans
		float64, for JSON numbers
		string, for JSON strings
		[]interface{}, for JSON arrays
		map[string]interface{}, for JSON objects
		nil for JSON null
		*/
		if errRPC != "" {
			log.Warning("获取用户（%s）信息失败！", session.GetUserid())
			return m.App.ProtocolMarshal("", errRPC)
		}

		user := userData.(map[string]interface{})
		userInfo := new(matchMsg.SCUserInfoRet)
		userInfo.ID = int64(user["id"].(float64))
		userInfo.Gender = user["sex"].(bool)
		userInfo.NickName = user["nick"].(string)
		userInfo.HeadURL = user["head"].(string)
		userInfo.Gold = int64(user["gold"].(float64))
		m.userInfoMap.Set(session.GetUserid(), *userInfo) //匹配成功缓存信息

		retStr, err := tableimp.Join(session, *userInfo)
		if err == "" {
			session.Set("lastplace", m.GetType())
			session.Set("BigRoomID", bigRoomID)
			session.Push()
			return m.App.ProtocolMarshal(retStr, "")
		}
		return m.App.ProtocolMarshal(retStr, err)
	}

	log.Warning("服务器出错，匹配成功的玩家根据房间ID未找到对应table")
	return m.App.ProtocolMarshal("", "房间未找到！") //出现这个log时需要排查原因
}

//GetTableByBigRoomID 根据bigroomid获取对应talbe实例
func (m *MatchRoom) GetTableByBigRoomID(bigRoomID string) (*Table, error) {
	gameID, tableID, _, err := ParseBigRoomID(bigRoomID)
	if err != nil {
		return nil, err
	}
	table := m.roomsMap.Get(gameID).(*room.Room).GetTable(tableID)
	if table != nil {
		tableimp := table.(*Table)
		return tableimp, nil
	}
	return nil, errors.New("No table found")
}

//退出table
func (m *MatchRoom) exit(session gate.Session, msg []byte) (string, string) {

	bigRoomID := session.Get("BigRoomID")
	table, err := m.GetTableByBigRoomID(bigRoomID)
	m.userInfoMap.Delete(session.GetUserid())
	if err != nil {
		return "", err.Error()
	}
	err = table.Exit(session)

	//log.Info("lastplace 设置为 lobby")
	if err != nil {

		return bigRoomID, ""
	}
	delete(session.GetSettings(), "BigRoomID")
	session.Set("lastplace", "Lobby")
	//log.Info("lastplace 设置为 lobby")
	session.Push()
	return "", ""
}

//坐下
func (m *MatchRoom) sitdown(session gate.Session, msg []byte) (string, string) {
	bigRoomID := session.Get("BigRoomID")
	if bigRoomID == "" {
		return "", "session中未包含BigRoomID字段"
	}
	table, err := m.GetTableByBigRoomID(bigRoomID)

	if err != nil {
		return "", err.Error()
	}
	err = table.PutQueue("SitDown", session)
	if err != nil {
		return "", err.Error()
	}
	return "success", ""
}

/*
//用户主动抓娃娃，直接进入结算期
func (m *MatchRoom) userDown(session gate.Session) (string, string) {
	bigRoomID := session.Get("BigRoomID")
	if bigRoomID == "" {
		return "", "session中未包含BigRoomID字段"
	}
	table, err := m.GetTableByBigRoomID(bigRoomID)

	if err != nil {
		return "", err.Error()
	}
	err = table.PutQueue("UserDone", session)
	if err != nil {
		return "", err.Error()
	}
	return "success", ""
}
*/
func (m *MatchRoom) coin(session gate.Session, msg []byte) (result bool, err string) {

	deviceInfo := new(matchMsg.SCDeviceCoin)

	if errJSON := deviceInfo.UnmarshalJSON(msg); errJSON != nil {
		log.Warning("下注信息有误： %s", errJSON.Error())
		return false, errJSON.Error()
	}

	if session.GetSettings()["BigRoomID"] != deviceInfo.BigRoomID {
		log.Warning("下注信息有误： 下注房间与所在房间不一致！")
		return false, "下注信息有误： 下注房间与所在房间不一致！"
	}

	deviceTable, bOk := m.deviceTableMap[deviceInfo.DeviceID]

	ret := new(matchMsg.SCCoinRet)
	if bOk {
		if deviceTable.Device.State >= 1 {
			ret.Success = false
			ret.Destription = "当前机器已有玩家正在游戏！"
			buf, _ := ret.MarshalJSON()
			session.SendNR("game/coin", buf)
			return false, "当前机器已有玩家正在游戏，请稍后再试！"
		}

		userInfo := m.userInfoMap.Get(session.GetUserid())
		if userInfo != nil {
			user := userInfo.(matchMsg.SCUserInfoRet)
			if user.Gold >= int64(deviceTable.Device.Cost) {
				err = deviceTable.DeviceSession.SendNR("game/coin", []byte(""))
				if err != "" {
					ret.Success = false
					ret.Destription = "设备不在线，请稍后再试"
					buf, _ := ret.MarshalJSON()
					session.SendNR("game/coin", buf)
					return false, err
				}

				user.Gold -= int64(deviceTable.Device.Cost)
				ret.Success = true
				ret.Destription = "投币成功！开始游戏！"
				ret.Gold = user.Gold
				buf, _ := ret.MarshalJSON()
				session.SendNR("game/coin", buf)
				deviceTable.Device.State = 1
				_, err := m.sitdown(session, nil)
				if err != "" {
					return false, err
				}
				//更新用户信息
				errUpdate := deviceTable.CoinUserUpdate(session, user)
				if errUpdate != nil {
					log.Warning("投币用户信息更新有误：%s", errUpdate.Error())
					err = "投币用户信息更新有误：" + errUpdate.Error()
					return false, err
				}
				//设备更新发送到大厅
				tableMsg := new(matchMsg.SCTableInfoRet)
				tableMsg.Device = *deviceTable.Device
				tableMsg.TableCount = deviceTable.GetViewer().Len() + 1
				tableMsg.BigRoomID = deviceInfo.BigRoomID
				buf, _ = tableMsg.MarshalJSON()
				//大厅更新设备信息
				m.RpcInvokeNR("Lobby", "deviceupdate", buf)
				//更新redis内金币信息
				m.RpcInvokeNR("RedisDB", "updateUserGold", session.GetUserid(), user.Gold)
				//更新大厅缓存金币信息
				//m.RpcInvokeNR("Lobby", "updateUserGold", session.GetUserid(), user.Gold)
				m.userInfoMap.Set(session.GetUserid(), user)
				return true, ""
			}
		}
	}
	ret.Success = false
	ret.Destription = "投币设备不存在！"
	buf, _ := ret.MarshalJSON()
	session.SendNR("game/coin", buf)
	return false, "投币设备不存在！"
}

func (m *MatchRoom) move(session gate.Session, msg []byte) (result bool, err string) {

	action := new(matchMsg.SCDeviceAction)

	if errJSON := action.UnmarshalJSON(msg); errJSON != nil {
		log.Warning("操作信息有误： %s", errJSON.Error())
		return false, errJSON.Error()
	}

	deviceTable, bOK := m.deviceTableMap[action.DeviceID]

	if bOK {
		//if deviceTable.GetSeats()[0].Session().GetUserid() == session.GetUserid() {
		if deviceTable.CurPlayer.Session().GetUserid() == session.GetUserid() {
			err = deviceTable.DeviceSession.SendNR("game/action", msg) //向设备发送移动指令
			if err != "" {
				//session.SendNR("game/action", []byte(err))
				return false, err
			}
			/*
				if action.Action == 5 { //主动抓物，通知table进入结算状态
					_, err := m.userDown(session)
					if err != "" {
						return false, err
					}
				}
			*/
			//session.SendNR("game/action", []byte("OK"))
			return true, ""
		}

		//session.SendNR("game/action", []byte("不是游戏用户，操作无效！"))
		return false, err
	}
	session.SendNR("game/action", []byte("设备不存在，操作无效！"))

	return false, "操作有误！"
}

func (m *MatchRoom) getSuccessList(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	rMsg := new(matchMsg.CSSuccessList)
	rMsg.UnmarshalJSON(msg)

	retList, _ := m.RpcInvoke("RedisDB", "getSuccessList", rMsg.DeviceID)
	return m.App.ProtocolMarshal(retList.(string), "")
}

func (m *MatchRoom) userDisConnect(session gate.Session) (result string, err string) {

	bigRoomID := session.Get("BigRoomID")
	table, errGet := m.GetTableByBigRoomID(bigRoomID)
	if errGet != nil {
		return "", errGet.Error()
	}
	if session.Get("IsDevice") != "" {
		log.Info("设备下线！！")
		//停止服务！
		table.Stop()
		delete(m.deviceTableMap, table.Device.DeviceID)
		delete(session.GetSettings(), "IsDevice")
		tableMsg := new(matchMsg.SCTableInfoRet)
		tableMsg.Device = matchMsg.SCDeviceInfo{State: -1}
		tableMsg.TableCount = 0
		tableMsg.BigRoomID = bigRoomID
		buf, _ := tableMsg.MarshalJSON()
		m.RpcInvokeNR("Lobby", "deviceupdate", buf)
	} else {
		m.userInfoMap.Delete(session.GetUserid())
		errGet = table.Exit(session)
		if errGet == nil {
			return bigRoomID, ""
		}
	}

	delete(session.GetSettings(), "BigRoomID")
	session.Set("lastplace", "Lobby")
	session.Push()
	return
}

func (m *MatchRoom) changeConfig(deviceID, newConfig string) (result bool, err string) {
	log.Info("设置新的抓取基数！%s-", deviceID)
	if deviceTable, bOK := m.deviceTableMap[deviceID]; bOK {
		log.Info("发送新的抓取基数！%s", deviceID)
		deviceTable.DeviceSession.SendNR("game/updateInfo", ([]byte)(newConfig))
		return true, ""
	}
	return false, "设备不在线！"
}
