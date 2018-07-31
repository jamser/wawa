//Package matchingroom 游戏匹配 游戏定义
package matchingroom

import (
	"errors"
	"zhuawawa/matchingroom/objects"
	userMsg "zhuawawa/msg"

	"github.com/liangdas/mqant/gate"
)

//SitDown 玩家坐下
func (t *Table) SitDown(session gate.Session) error {
	playerImp := t.GetBindPlayer(session)
	if playerImp != nil {
		player := playerImp.(*objects.Player)
		player.OnRequest(session)
		player.OnSitDown()
		t.CurPlayer = *player
		t.CurPlayer.UserID = session.GetUserid()
		//林： 玩家坐下后自动开始
		t.Start()
	}
	return nil
}

//UserDone 玩家主动抓物
/*
func (t *Table) UserDone(session gate.Session) error {
	playerImp := t.GetBindPlayer(session)
	if playerImp != nil {
		player := playerImp.(*objects.Player)
		player.OnRequest(session)
	}

	//主动进入结算期
	t.fsm.Call(SettlementPeriodEvent)
	return nil
}
*/

//PauseGame 暂停游戏
func (t *Table) PauseGame(session gate.Session) error {
	playerImp := t.GetBindPlayer(session)
	if playerImp != nil {
		player := playerImp.(*objects.Player)
		player.OnRequest(session)
		player.OnSitDown()
		t.Pause()
		return nil
	}
	return nil
}

//Join 玩家加入table 玩家加入场景
func (t *Table) Join(session gate.Session, userInfo userMsg.SCUserInfoRet) (string, string) {
	player := t.GetBindPlayer(session)
	if player != nil {
		playerImp := player.(*objects.Player)
		playerImp.OnRequest(session)
		playerImp.UserInfo = userInfo
		t.NotifyJoin(playerImp) //广播给所有玩家 加入消息
	} else {
		indexSeat := -1
		for i, player := range t.seats {
			if !player.Bind() {
				indexSeat = i
				player.OnBind(session)
				player.UserInfo = userInfo
				t.NotifyJoin(player) //广播给所有玩家 加入消息
				break
			}
		}
		if indexSeat == -1 {
			return "", "房间已满！"
		}
	}
	player = t.GetBindPlayer(session)
	playerImp := player.(*objects.Player)
	t.lastTwoPlaysers[1] = t.lastTwoPlaysers[0]
	t.lastTwoPlaysers[0] = *playerImp

	retInfo := new(userMsg.SCRoomLast2Players)
	idx := 0
	for i := 0; i < 2; i++ {
		if (t.lastTwoPlaysers[i].UserID != "") && (t.lastTwoPlaysers[i].UserID != session.GetUserid()) {
			retInfo.Users[idx] = t.lastTwoPlaysers[i].UserID
			retInfo.Heads[idx] = t.lastTwoPlaysers[i].UserInfo.HeadURL
			idx++
		}
	}
	retInfo.Playing = t.Device.State >= 1
	buf, _ := retInfo.MarshalJSON()
	return string(buf), ""
}

//CoinUserUpdate 投币用户更新信息
func (t *Table) CoinUserUpdate(session gate.Session, userInfo userMsg.SCUserInfoRet) error {
	player := t.GetBindPlayer(session)
	if player != nil {
		playerImp := player.(*objects.Player)
		playerImp.UserInfo = userInfo
		return nil
	}

	return errors.New("投币用户不在当前房间！")
}
