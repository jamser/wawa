//Package matchingroom 游戏匹配 房间模块(通知 定义)
package matchingroom

import (
	"strconv"
	"zhuawawa/matchingroom/objects"

	"github.com/liangdas/mqant/log"

	RetMsg "zhuawawa/msg"
)

//NotifyJoin 通知所有玩家有新玩家加入
func (t *Table) NotifyJoin(player *objects.Player) {
	buf, _ := player.Serializable()
	t.NotifyCallBackMsg("game/UserJoin", buf)
}

//NotifyIdle 通知所有玩家进入准备期 （不可以投币）
func (t *Table) NotifyIdle() {
	t.NotifyCallBackMsg("game/EnterIdle", []byte(""))
}

//NotifyCaught 通知所有玩家进入下抓期 （不可以投币）
func (t *Table) NotifyCaught() {
	t.NotifyCallBackMsg("game/Caught", []byte(""))
}

//NotifyDeviceResult 通知所有玩家中奖结果 （可以投币）
func (t *Table) NotifyDeviceResult(state int) {

	deviceResult := new(RetMsg.SCDeviceRegRet)
	deviceResult.Destription = strconv.FormatInt(t.CurPlayer.UserInfo.ID, 10)
	deviceResult.UserID = t.CurPlayer.UserInfo.ID
	if state == 4 {
		log.Info("玩家(%s)中奖!", t.CurPlayer.UserInfo.NickName)
		deviceResult.Success = true

	} else {
		log.Info("玩家(%s)未中奖!", t.CurPlayer.UserInfo.NickName)
		deviceResult.Success = false
	}
	t.Device.State = 0

	buf, _ := deviceResult.MarshalJSON()
	//log.Info("玩家中奖信息!")
	t.NotifyCallBackMsg("game/result", buf)
}

//NotifyOperation 通知所有玩家进入操作期 ， 不可再投币
func (t *Table) NotifyOperation(opUser []byte) {
	//log.Info("通知所有人投币成功玩家信息")
	t.NotifyCallBackMsg("game/opUser", opUser)
}

/*
//NotifySettlement 通知所有玩家进入结算期
func (t *Table) NotifySettlement(settlement []byte) {
	t.NotifyCallBackMsg("game/Settlement", settlement)
}
*/

//NotifyTime 通知所有玩家时间更新
func (t *Table) NotifyTime(timeMsg []byte) {
	t.NotifyCallBackMsg("game/TimeUpdate", timeMsg)
}

//NotifyStop 游戏结束通知
/*
func (t *Table) NotifyStop() {
	t.NotifyCallBackMsg("game/GameEnd", []byte(""))
}
*/

//NotifyChat 广播聊天信息
func (t *Table) NotifyChat(chatMsg []byte) {
	t.NotifyCallBackMsg("game/RoomChat", chatMsg)
}
