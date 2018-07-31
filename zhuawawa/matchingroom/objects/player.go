//Package objects 玩家信息定义
package objects

import (
	playerMsg "zhuawawa/msg"

	"github.com/liangdas/mqant-modules/room"
)

//Player 玩家信息结构体
type Player struct {
	room.BasePlayerImp
	UserInfo playerMsg.SCUserInfoRet
	UserID   string //防止断线后成功抓到娃娃需要记录时拿不到UserID
}

//NewPlayer 创建游戏玩家
func NewPlayer() *Player {
	player := new(Player)
	return player
}

//OnGameOver 当玩家退出时调用， 清除本局相关信息
func (p *Player) OnGameOver() {
	p.OnUnBind()
}

//Serializable 序列化玩家信息
func (p *Player) Serializable() ([]byte, error) {
	return p.UserInfo.MarshalJSON()
}
