//Package matchingroom 游戏匹配 房间模块(table 定义)
package matchingroom

import (
	"container/list"
	"math/rand"
	"time"
	"zhuawawa/matchingroom/objects"
	tableMsg "zhuawawa/msg"

	"github.com/liangdas/mqant-modules/room"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

//RandInt64 随机数 区间
func RandInt64(min, max int64) int64 {
	if min >= max {
		return max
	}
	return rand.Int63n(max-min) + min
}

//MaxSeats 每桌最多游戏人数
const MaxSeats = 100

//游戏逻辑相关的代码
//table 创建后， 默认为初始化状态， 包括结算后房间也将重置为初始化状态， 房间可能被pool管理，循环使用
//第一阶段 空闲期，倒计时，系统准备题库 （3S）
//第二阶段 答题期  玩家答题 （10s）
//第三阶段 开题期  公布答案， 玩家更新得分等UI并等待下一题（答满题目则进入第四阶段）
//第四阶段 结算期  结算

//Table 桌子定义结构体
type Table struct {
	fsm                          FSM //游戏状态机
	room.BaseTableImp                //table 基础功能
	room.QueueTable                  //table 事件队列
	room.UnifiedSendMessageTable     //table 统一发送函数接口
	room.TimeOutTable                //table 断线和超时处理接口
	module                       module.RPCModule
	seats                        []*objects.Player //游戏玩家
	viewer                       *list.List        //观众
	seatMax                      int               //房间最大座位数
	stoped                       bool

	InitPeriodHandler       FSMHandler //初始化状态 处理器
	IdlePeriodHandler       FSMHandler //空闲期状态 处理器
	OperationPeriodHandler  FSMHandler //操作期状态 处理器
	CaughtPeriodHandler     FSMHandler //下抓期状态 处理器
	SettlementPeriodHandler FSMHandler //结算期状态 处理器
	//IdleCountDown           time.Duration //空闲期倒计时 时长
	OperationCountDown time.Duration //操作期倒计时 时长
	//CaughtCountDown         time.Duration //下抓期倒计时 时长
	//SettlementCountDown     time.Duration //结算期倒计时 时长
	CurCountDown time.Duration //倒计时 时长

	//IdleTime       time.Time //空闲期进入时间
	OperationTime time.Time //操作期进入时间
	//CaughtTime time.Time //下抓期进入时间
	//SettlementTime time.Time //结算期进入时间

	Device          *tableMsg.SCDeviceInfo //设备信息
	DeviceSession   gate.Session           //设备session，主动给设备发送投币，控制等信息
	CurPlayer       objects.Player         //当前操作玩家
	lastTwoPlaysers []objects.Player       //前两位操作玩家
}

//SetTableDeviceAndSession 创建桌子后设置设备信息
func (table *Table) SetTableDeviceAndSession(device *tableMsg.SCDeviceInfo, session gate.Session) {
	table.Device = device
	table.DeviceSession = session
}

//GetModule 获取talbe的RPC 模块
func (table *Table) GetModule() module.RPCModule {
	return table.module
}

//NewTable 创建新桌子
func NewTable(module module.RPCModule, tableID int) *Table {
	table := &Table{
		module:  module,
		stoped:  true,
		seatMax: MaxSeats,
		//IdleCountDown:       0,
		OperationCountDown: 15,
		//CaughtCountDown:     15,
		//SettlementCountDown: 15,
		CurCountDown:    0,
		lastTwoPlaysers: make([]objects.Player, 2),
	}

	table.BaseTableImpInit(tableID, table)
	table.QueueInit()
	table.UnifiedSendMessageTableInit(table)
	//lin: 暂时没必要做超时检测
	//table.TimeOutTableInit(table, table, 60)
	//游戏逻辑状态机
	table.InitFsm()
	table.seats = make([]*objects.Player, table.seatMax)
	table.viewer = list.New()

	table.Register("SitDown", table.SitDown)
	//	table.Register("UserDone", table.UserDone)

	//for indexSeat, _ := range table.seats {
	for indexSeat := range table.seats {
		table.seats[indexSeat] = objects.NewPlayer()
	}

	return table
}

//GetSeats 获取游戏玩家列表
func (table *Table) GetSeats() []room.BasePlayer {
	m := make([]room.BasePlayer, len(table.seats))
	for i, seat := range table.seats {
		m[i] = seat
	}
	return m
}

//GetViewer 获取观众列表
func (table *Table) GetViewer() *list.List {
	return table.viewer
}

//OnNetBroken 玩家断线
func (table *Table) OnNetBroken(player room.BasePlayer) {
	player.OnNetBroken()
}

//VerifyAccessAuthority 访问权限校验
func (table *Table) VerifyAccessAuthority(userID string, bigRoomID string) bool {
	_, tableID, transactionID, err := ParseBigRoomID(bigRoomID)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if (tableID != table.TableId()) || (transactionID != table.TransactionId()) {
		log.Error("transactionId!=this.TransactionId()", transactionID, table.TransactionId())
		return false
	}
	return true
}

//AllowJoin 是否可以加入
func (table *Table) AllowJoin() bool {
	ready := true
	for _, seat := range table.GetSeats() {
		if seat.Bind() == false {
			//还没有准备好
			ready = false
			break
		}
	}
	return !ready
}

//OnCreate 第一次被创建的时候调用
func (table *Table) OnCreate() {
	table.BaseTableImp.OnCreate()
	table.ResetTimeOut()
	log.Debug("Table-%s", "OnCreate")
	if table.stoped {
		table.stoped = false
		go func() {
			//这里设置为500ms
			tick := time.NewTicker(100 * time.Millisecond)
			defer func() {
				tick.Stop()
			}()
			for !table.stoped {
				select {
				case <-tick.C:
					table.Update(nil)
				}
			}
		}()
	}
}

//OnStart 一次游戏开始时调用
func (table *Table) OnStart() {
	table.BaseTableImp.OnStart()
	log.Debug("Table-%s", "OnStart")
	//将游戏状态设置到空闲期
	table.fsm.Call(IdlePeriodEvent)
}

//OnResume 取得控制权，可接受用户输入
func (table *Table) OnResume() {
	table.BaseTableImp.OnResume()
	log.Debug("Table-%s", "OnResume")
	//table.NotifyResume()
}

//OnPause table内暂停，可接收用户消息,此方法主要用在游戏过程中的游戏时钟暂停,不销毁本次游戏的数据
func (table *Table) OnPause() {
	table.BaseTableImp.OnPause()
	log.Debug("Table-%s", "OnPause")
	//table.NotifyPause()
}

//OnStop 当本次游戏完成时调用,这里需要销毁游戏数据，对游戏数据做本地化处理，比如游戏结算等
func (table *Table) OnStop() {
	table.BaseTableImp.OnStop()
	log.Debug("Table-%s", "OnStop")
	//将游戏状态设置到初始化期
	table.fsm.Call(InitPeriodEvent)
	//table.NotifyStop()
	table.ExecuteCallBackMsg() //统一发送数据到客户端
	table.CurCountDown = 0
	/*
		for _, player := range table.seats {
			player.OnGameOver()
		}

		var nv *list.Element
		for e := table.viewer.Front(); e != nil; e = nv {
			nv = e.Next()
			table.viewer.Remove(e)
		}
	*/
}

//OnDestroy 在table销毁时调用,将无法再接收用户消息
func (table *Table) OnDestroy() {
	table.BaseTableImp.OnDestroy()
	log.Debug("Table-%s", "OnDestroy")
	table.stoped = true
}

func (table *Table) onGameOver() {
	table.CurCountDown = 0
	table.Finish()
}

//Update 牌桌主循环
func (table *Table) Update(arge interface{}) {
	table.ExecuteEvent(arge) //执行这一帧客户端发送过来的消息
	if table.State() == room.Active {
		table.StateSwitch()
	} else if table.State() == room.Initialized {
		/*
			ready := true
			for _, seat := range table.GetSeats() {
				if seat.SitDown() == false {
					//还没有准备好
					log.Info("当前还没有准备好")
					ready = false
					break
				}
			}

			if ready {
				log.Info("当前所有人都准备了")
				table.Start() //开始游戏了
			}
		*/
	}

	table.ExecuteCallBackMsg() //统一发送数据到客户端
	//lin: 暂时没必要做超时检测
	//table.CheckTimeOut()
}

//Exit 玩家离开牌桌
func (table *Table) Exit(session gate.Session) error {
	player := table.GetBindPlayer(session)
	if player != nil {
		playerImp := player.(*objects.Player)
		playerImp.OnUnBind()
		return nil
	}
	return nil
}
