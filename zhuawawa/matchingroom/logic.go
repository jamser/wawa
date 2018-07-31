//Package matchingroom 游戏匹配 房间模块(table 定义)
package matchingroom

import (
	"fmt"
	"time"
	deviceMsg "zhuawawa/msg"

	"github.com/liangdas/mqant/log"
)

var (
	//InitPeriod 初始化期 状态
	InitPeriod = FSMState("初始化期")
	//IdlePeriod 空闲期 状态
	IdlePeriod = FSMState("空闲期")
	//OperationPeriod 操作期 状态
	OperationPeriod = FSMState("操作期")
	//CaughtPeriod 下抓期 状态
	CaughtPeriod = FSMState("下抓期")
	//SettlementPeriod 结算期 状态
	SettlementPeriod = FSMState("结算期")
	//InitPeriodEvent 进入初始化期 事件
	InitPeriodEvent = FSMEvent("进入初始化期")
	//IdlePeriodEvent 进入空闲期 事件
	IdlePeriodEvent = FSMEvent("进入空闲期")
	//OperationPeriodEvent 进入操作期 事件
	OperationPeriodEvent = FSMEvent("进入操作期")
	//CaughtPeriodEvent 进入下抓期  事件
	CaughtPeriodEvent = FSMEvent("进入下抓期")
	//SettlementPeriodEvent 进入结算期 事件
	SettlementPeriodEvent = FSMEvent("进入结算期")
)

//InitFsm 初始化状态机
func (table *Table) InitFsm() {
	table.fsm = *NewFSM(InitPeriod)
	table.InitPeriodHandler = FSMHandler(func() FSMState {
		log.Info("table (%d)已进入初始化期", table.TableId)
		return InitPeriod
	})
	table.IdlePeriodHandler = FSMHandler(func() FSMState {
		log.Info("table (%d)已进入空闲期", table.TableId)
		//table.CurCountDown = 0
		//table.updateTime(table.IdleCountDown)
		//table.IdleTime = time.Now()
		//广播通知 （投币玩家已确定，隐藏投币按钮）
		table.NotifyIdle()
		return IdlePeriod
	})

	table.OperationPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入操作期")
		table.CurCountDown = 0
		table.updateTime(table.OperationCountDown)
		table.OperationTime = time.Now()
		//广播通知
		buf, _ := table.CurPlayer.Serializable()
		table.NotifyOperation(buf)

		return OperationPeriod
	})

	table.CaughtPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入下抓期")

		table.NotifyCaught()
		/*
			userSession := table.CurPlayer.Session()
			//通知当前玩家
			if userSession != nil {
				userSession.SendNR("game/Caught", []byte(""))
			}
		*/
		if table.Device.State == 1 {
			//到时间，主动下抓
			action := new(deviceMsg.SCDeviceAction)
			action.Action = 5
			msg, _ := action.MarshalJSON()
			table.DeviceSession.SendNR("game/action", msg) //向设备发送移动指令
		}
		return CaughtPeriod
	})

	table.SettlementPeriodHandler = FSMHandler(func() FSMState {
		fmt.Println("已进入结算期")
		//table.CurCountDown = 0
		//table.updateTime(table.SettlementCountDown)
		//table.SettlementTime = time.Now()

		//结算时说明游戏结束，玩家站起，需要继续投币
		table.CurPlayer.OnSitUp()

		//广播通知
		//table.NotifySettlement([]byte(""))

		return SettlementPeriod
	})

	table.fsm.AddHandler(InitPeriod, IdlePeriodEvent, table.IdlePeriodHandler)
	table.fsm.AddHandler(IdlePeriod, OperationPeriodEvent, table.OperationPeriodHandler)
	table.fsm.AddHandler(OperationPeriod, CaughtPeriodEvent, table.CaughtPeriodHandler)
	table.fsm.AddHandler(CaughtPeriod, SettlementPeriodEvent, table.SettlementPeriodHandler)
	table.fsm.AddHandler(SettlementPeriod, InitPeriodEvent, table.InitPeriodHandler)
}

func (table *Table) updateTime(timeLeft time.Duration) {
	timeMsg := new(deviceMsg.SCTimeUpdate)
	timeMsg.TimeCountDown = int64(timeLeft)
	buf, _ := timeMsg.MarshalJSON()
	table.NotifyTime(buf)
}

//StateSwitch 状态切换
func (table *Table) StateSwitch() {

	switch table.fsm.getState() {
	case InitPeriod:
		//log.Info("table(%d)在初始化状态")
	case IdlePeriod:
		table.fsm.Call(OperationPeriodEvent)
	case OperationPeriod:
		timepassed := time.Since(table.OperationTime)

		if timepassed > (table.OperationCountDown * time.Second) {
			//log.Info("StateSwitch to SettlementPeriodEvent")
			table.fsm.Call(CaughtPeriodEvent)
		} else {
			if timepassed > (table.CurCountDown+1)*time.Second {
				//更新时间
				table.CurCountDown++
				table.updateTime(table.OperationCountDown - table.CurCountDown)
			}
		}
	case CaughtPeriod:

	case SettlementPeriod:
		//timepassed := time.Since(table.SettlementTime)
		//if timepassed > (table.SettlementCountDown * time.Second) {
		//table.fsm.Call(InitPeriodEvent)
		table.Stop()
		//}
		/*else if timepassed > (table.CurCountDown+1)*time.Second {
			//更新时间
			table.CurCountDown++
			table.updateTime(table.SettlementCountDown - table.CurCountDown)
		}*/
	}
}
