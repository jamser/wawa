/*Package lobby 大厅模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package lobby

import (
	"strconv"
	"time"
	MatchingRoom "zhuawawa/matchingroom"
	LobbyMsg "zhuawawa/msg"

	"github.com/astaxie/beego/utils"
	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"
)

//UserRetainTime 用户断线后用户信息保存时间
const UserRetainTime = 30 * time.Second

//Module 模块实例
var Module = func() module.Module {
	lobby := new(Lobby)
	return lobby
}

//Lobby 大厅结构体
type Lobby struct {
	basemodule.BaseModule
	//UserMap     *utils.BeeMap
	//User2DelMap *utils.BeeMap
	RoomsMap map[int]*utils.BeeMap
}

/*
type usersInfo struct {
	session  gate.Session //用户session，用于发送数据
	ID       int64        //用户ID
	NickName string       //用户昵称
	HeadURL  string       //用户头像URL
	Gender   bool         //性别 （true 男  false 女）
	Gold     int64        //金币数量
}
*/

//GetType 模块类型
func (lobbyl *Lobby) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Lobby"
}

//Version 模块版本
func (lobbyl *Lobby) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

//OnInit 模块初始化
func (lobbyl *Lobby) OnInit(app module.App, settings *conf.ModuleSettings) {
	lobbyl.BaseModule.OnInit(lobbyl, app, settings)

	//lobbyl.UserMap = utils.NewBeeMap()
	//lobbyl.User2DelMap = utils.NewBeeMap()
	lobbyl.RoomsMap = make(map[int]*utils.BeeMap, 3)

	lobbyl.GetServer().RegisterGO("HD_LoginLobby", lobbyl.loginLobby)      //登陆大厅，获取用户自己信息以及大厅信息
	lobbyl.GetServer().RegisterGO("HD_GetRoomsInfo", lobbyl.roomInfo)      //获取相应房间的table信息（即娃娃机信息）
	lobbyl.GetServer().RegisterGO("deviceupdate", lobbyl.updateDevice)     //更新房间的table信息（即娃娃机信息）
	lobbyl.GetServer().RegisterGO("userDisConnect", lobbyl.userDisconnect) //用户断开连接，开启定时器，一段时间后清理信息

	lobbyl.GetServer().RegisterGO("getUserData", lobbyl.getUserData)//信息的更新
	//lobbyl.GetServer().RegisterGO("updateUserGold", lobbyl.updateUserGold)
	lobbyl.GetServer().RegisterGO("HD_GetSuccessRank", lobbyl.getSuccessRank) //客户端获取成功排行榜
	lobbyl.GetServer().RegisterGO("HD_GetPayRank", lobbyl.getPayRank)         //客户端获取充值排行榜
	lobbyl.GetServer().RegisterGO("HD_GetCheckInfo", lobbyl.getCheckInfo)     //客户端获取签到信息
	lobbyl.GetServer().RegisterGO("HD_CheckIn", lobbyl.checkIn)               //客户端签到
	lobbyl.GetServer().RegisterGO("HD_GetAddressInfo", lobbyl.getAddressInfo) //客户端获取发货地址信息
	lobbyl.GetServer().RegisterGO("HD_AddAddress", lobbyl.addAddress)         //客户端新增发货地址

	//邀请码相关
	lobbyl.GetServer().RegisterGO("HD_GetInviteInfo", lobbyl.getInviteInfo) //获取邀请信息（自己邀请码，邀请人个数等）
	lobbyl.GetServer().RegisterGO("HD_BindUser", lobbyl.bindUser)           //绑定邀请码

	//活动相关
	lobbyl.GetServer().RegisterGO("HD_GetActivesInfo", lobbyl.getActivesInfo) //获取活动信息

	//抓中娃娃相关
	lobbyl.GetServer().RegisterGO("HD_GetWaWaInfo", lobbyl.getWawaInfo)       //获取所有娃娃信息
	lobbyl.GetServer().RegisterGO("HD_ExchangeWawa", lobbyl.exchangeWawa)     //娃娃兑换娃娃币
	lobbyl.GetServer().RegisterGO("HD_AskForDelivery", lobbyl.askForDelivery) //娃娃兑换娃娃币

	//分享奖励
	lobbyl.GetServer().RegisterGO("HD_ShareDone", lobbyl.shareDone) //用户分享成功，奖励金币

	//联系信息
	lobbyl.GetServer().RegisterGO("HD_Contact", lobbyl.getContact) //获取联系信息
}

//Run 模块运行
func (lobbyl *Lobby) Run(closeSig chan bool) {
	/*
	   	t := time.NewTicker(time.Second)
	   	defer t.Stop()

	   ForEnd:
	   	for {
	   		select {
	   		case <-closeSig:
	   			break ForEnd
	   		case <-t.C:
	   			//查看待删除列表中是否有到期用户需要删除
	   			for userID, userTime := range lobbyl.User2DelMap.Items() {
	   				log.Info("玩家%s在清空列表内！", userID)
	   				if time.Since(userTime.(time.Time)) > UserRetainTime {
	   					lobbyl.UserMap.Delete(userID)
	   					lobbyl.User2DelMap.Delete(userID)
	   					log.Info("清空玩家%s信息", userID)
	   				}
	   			}

	   		}
	   	}
	*/
}

//OnDestroy 模块清理
func (lobbyl *Lobby) OnDestroy() {
	//一定别忘了关闭RPC
	lobbyl.GetServer().OnDestroy()
}

/*****************以下设备相关函数*******************************/

func (lobbyl *Lobby) roomInfo(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	queryRoommsg := new(LobbyMsg.CSQueryRoomInfo)
	jsonErr := queryRoommsg.UnmarshalJSON(msg)
	retMsg := new(LobbyMsg.SCRoomInfoRet)

	if jsonErr != nil {
		retMsg.Success = false
		retData, _ := retMsg.MarshalJSON()
		return lobbyl.App.ProtocolMarshal(string(retData), "获取房间信息失败！")
	}

	retMsg.Success = true

	if queryRoommsg.GameID == 0 { //所有类型
		retMsg.Devices = make([]LobbyMsg.SCTableInfoRet, 0)
		//idx := 0
		for _, roomMap := range lobbyl.RoomsMap {
			tables := roomMap.Items()
			for _, v := range tables {
				retMsg.Devices = append(retMsg.Devices, v.(LobbyMsg.SCTableInfoRet))
				//retMsg.Devices[idx] = v.(LobbyMsg.SCTableInfoRet)
				//idx++
			}
		}
	} else {
		tablesMap, bOk := lobbyl.RoomsMap[queryRoommsg.GameID]

		if bOk {
			tables := tablesMap.Items()
			retMsg.Devices = make([]LobbyMsg.SCTableInfoRet, len(tables))
			idx := 0
			for _, v := range tables {
				retMsg.Devices[idx] = v.(LobbyMsg.SCTableInfoRet)
				idx++
			}
		} else {
			retMsg.Devices = make([]LobbyMsg.SCTableInfoRet, 0)
		}
	}

	retData, _ := retMsg.MarshalJSON()
	return lobbyl.App.ProtocolMarshal(string(retData), "")
}

func (lobbyl *Lobby) updateDevice(msg []byte) (result string, err string) {
	tableMsg := new(LobbyMsg.SCTableInfoRet)
	errJSON := tableMsg.UnmarshalJSON(msg)
	if errJSON != nil {
		return "", "解析SCTableInfoRet 数据出错"
	}

	gameID, tableID, _, errID := MatchingRoom.ParseBigRoomID(tableMsg.BigRoomID)

	if errID != nil {
		log.Warning("更新设备（%s）状态出错！", tableMsg.Device.DeviceID)
	}

	if tableMsg.Device.State == -1 {
		if dMap, bOK := lobbyl.RoomsMap[gameID]; bOK {
			//log.Info("娃娃机下线 %d", tableID)
			dMap.Delete(tableID)
		} else {
			log.Warning("设备map出错!!!!")
		}
		return
	}

	if dMap, bOK := lobbyl.RoomsMap[gameID]; bOK {
		dMap.Set(tableID, *tableMsg)
	} else {
		lobbyl.RoomsMap[gameID] = utils.NewBeeMap()
		lobbyl.RoomsMap[gameID].Set(tableID, *tableMsg)
	}
	return
}

/*****************以上设备相关函数*******************************/

/*****************以下用户相关函数*******************************/
func (lobbyl *Lobby) loginLobby(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	
		//是否在列表内
		if lobbyl.UserMap.Check(session.GetUserid()) {

			//从待删除列表内移除该用户
			if lobbyl.User2DelMap.Get(session.GetUserid()) != nil {
				lobbyl.User2DelMap.Delete(session.GetUserid())
			}


				//存在，则更新session信息
				user := (lobbyl.UserMap.Get(session.GetUserid())).(usersInfo)
				user.session = session

				infoRet := new(LobbyMsg.SCUserInfoRet)
				infoRet.ID = user.ID
				infoRet.HeadURL = user.HeadURL
				infoRet.NickName = user.NickName
				infoRet.Gender = user.Gender
				infoRet.Gold = user.Gold

				buf, _ := infoRet.MarshalJSON()
				return lobbyl.App.ProtocolMarshal(string(buf), err)

		}
	
	r, err := lobbyl.RpcInvoke("RedisDB", "getUserInfo", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}

	ret := r.(map[string]interface{})
	//lobbyl.UserMap.Set(session.GetUserid(), usersInfo{ID: int64(ret["id"].(float64)), NickName: ret["nick"].(string), HeadURL: ret["head"].(string), Gender: ret["sex"].(bool), Gold: int64(ret["gold"].(float64)), session: session})
	infoRet := new(LobbyMsg.SCUserInfoRet)
	infoRet.ID = int64(ret["id"].(float64))
	infoRet.HeadURL = ret["head"].(string)
	infoRet.NickName = ret["nick"].(string)
	infoRet.Gender = ret["sex"].(bool)
	infoRet.Gold = int64(ret["gold"].(float64))

	buf, _ := infoRet.MarshalJSON()
	return lobbyl.App.ProtocolMarshal(string(buf), err)
}

func (lobbyl *Lobby) userDisconnect(session gate.Session) (result string, err string) {
	/*
		//玩家在大厅内才处理
		if lobbyl.UserMap.Get(userID) != nil {
			if r := lobbyl.User2DelMap.Get(userID); r == nil {
				lobbyl.User2DelMap.Set(userID, time.Now())
				log.Info("用户%s加入大厅待删除用户列表！", userID)
			} else {
				log.Warning("用户%s已在大厅待删除用户列表！", userID)
			}
		} else {
			log.Warning("用户%s不在大厅内，无法从大厅离开", userID)
		}
	*/
	return
}


func (lobbyl *Lobby) getUserData(userID string) (result map[string]interface{}, err string) {

	 ret := lobbyl.UserMap.Get(userID)
	if ret != nil {
		user := ret.(usersInfo)
		result = make(map[string]interface{})
		result["id"] = user.ID
		result["head"] = user.HeadURL
		result["sex"] = user.Gender
		result["nick"] = user.NickName
		result["gold"] = user.Gold
		return
	}
r, err := lobbyl.RpcInvoke("RedisDB", "getUserData")
	log.Warning("获取用户（%s）信息失败", userID)
	err = "获取用户信息失败"
	return
}
/*
func (lobbyl *Lobby) updateUserGold(userID string, newGold int64) (result bool, err string) {
	ret := lobbyl.UserMap.Get(userID)
	if ret != nil {
		user := ret.(usersInfo)
		user.Gold = newGold
		lobbyl.UserMap.Set(userID, user)
		return
	}
	return false, "更新用户不存在！"
}
*/
func (lobbyl *Lobby) getSuccessRank(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getSuccessRank", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) getPayRank(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getPayRank", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) getCheckInfo(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getCheckInfo", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) checkIn(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "checkIn", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) getAddressInfo(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getAddressInfo", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) addAddress(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "addAddress", session.GetUserid(), msg)
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) getInviteInfo(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getInviteInfo", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) bindUser(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "bindUser", session.GetUserid(), msg)
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) getActivesInfo(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getActives")
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) getWawaInfo(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "getWaWaInfo", session.GetUserid())
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) exchangeWawa(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "exchangeWawa", session.GetUserid(), msg)
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) askForDelivery(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	retList, err := lobbyl.RpcInvoke("RedisDB", "askForDelivery", session.GetUserid(), msg)
	if err != "" {
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retList.(string), "")
}

func (lobbyl *Lobby) shareDone(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	nowGold, err := lobbyl.RpcInvoke("RedisDB", "shareDone", session.GetUserid(), msg)
	if err != "" {
		log.Warning("分享得奖励失败：" + err)
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(strconv.Itoa(int(nowGold.(int64))), err)
}

func (lobbyl *Lobby) getContact(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	retInfo, err := lobbyl.RpcInvoke("RedisDB", "getBGContackInfo")
	if err != "" {
		log.Warning("获取联系信息失败：" + err)
		return lobbyl.App.ProtocolMarshal("", err)
	}
	return lobbyl.App.ProtocolMarshal(retInfo.(string), err)
}

/*****************以上用户相关函数*******************************/
