/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"
)

//Module 模块实例
var Module = func() module.Module {
	redisdb := new(redisDB)
	return redisdb
}

var pool *redis.Pool

const userIDMin int64 = 100000 //用户ID最小值

type redisDB struct {
	basemodule.BaseModule
}

func (m *redisDB) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "RedisDB"
}
func (m *redisDB) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (m *redisDB) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)

	pool = &redis.Pool{
		MaxIdle:     16,
		MaxActive:   1024,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}

	for cNum := 0; cNum < pool.MaxIdle; cNum++ {
		go func() {
			c := pool.Get()
			defer c.Close()
		}()
	}

	c := pool.Get()
	defer c.Close()

	initScript := redis.NewScript(0, `
		if not redis.call("Get", "PubNotice") then
			redis.call("Set", "PubNotice", "快乐娃娃机欢迎各位玩家，祝游戏愉快")
		
			--注册管理用户ID agPwd 123456的md5值
			redis.call("HMSET", "100000", "agPwd", "e10adc3949ba59abbe56e057f20f883e", "nick","总代理")  
			--签到奖励默认设置	
			redis.call("HMSet", "SignReward", "Day1", 10, "Day2", 10, "Day3", 10, "Day4", 10,"Day5", 10, "Day6", 10,"Day7", 40)
			
			--分享奖励默认设置 ， 每天 好友和朋友圈各有一次分享奖励机会， 奖励1娃娃币
			redis.call("hmset", "ShareConfig", "Friend", 1, "Circle", 1)
			
			--商品默认设置
			redis.call("hmset", "shopInfos", "1", '{"shopID": "1", "title": "周卡", "des": "花费48元购买周卡", "extra": "周卡", "type": 1, "cost": 48, "ext": 50, "gold": 480,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"2", '{"shopID": "2", "title": "月卡", "des": "花费100元购买月卡", "extra": "月卡", "type": 2, "cost": 100, "ext": 34, "gold": 1000,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"3", '{"shopID": "3", "title": "购买娃娃币", "des": "花费6元购买60娃娃币", "extra": "", "type": 0, "cost": 6, "ext": 0, "gold": 60,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"4", '{"shopID": "4", "title": "购买娃娃币", "des": "花费18元购买280娃娃币", "extra": "", "type": 0, "cost": 18, "ext": 0, "gold": 280,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"5", '{"shopID": "5", "title": "购买娃娃币", "des": "花费50元购买800娃娃币", "extra": "", "type": 0, "cost": 50, "ext": 0, "gold": 800,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"6", '{"shopID": "6", "title": "购买娃娃币", "des": "花费98元购买1600娃娃币", "extra": "", "type": 0, "cost": 98, "ext": 0, "gold": 1600,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"7", '{"shopID": "7", "title": "购买娃娃币", "des": "花费298元购买4900娃娃币", "extra": "", "type": 0, "cost": 298, "ext": 0, "gold": 4900,"url": "http://www.wingjoy.cn/images/cooker-img.png"}',
				"8", '{"shopID": "8", "title": "购买娃娃币", "des": "花费518元购买8700娃娃币", "extra": "", "type": 0, "cost": 518, "ext": 0, "gold": 8700,"url": "http://www.wingjoy.cn/images/cooker-img.png"}'
			)

			--绑定相关默认设置
			redis.call("hmset", "bindInfos", "userGain", 60, "binderGain", 30, "goldGainMaxUser", 10, "oneMaxMoney", 100, "totalMoney", 10000, "moneyRate", 0.04)

			--活动默认图片设置
			redis.call("hmset", "actives", "1", "http://www.wingjoy.cn/images/cooker-img.png", "2", "http://www.wingjoy.cn/images/cooker-img.png", "3", "http://www.wingjoy.cn/images/cooker-img.png", "4", "http://www.wingjoy.cn/images/cooker-img.png")

			--默认联系方式
			redis.call("hmset", "contact", "qq", "741464061", "wechat", "sjw741464061", "kefu", "待设置")
		end
	`)

	if _, errRedis := initScript.Do(c); errRedis != nil {
		log.Error("初始化出错:" + errRedis.Error())
	}

	log.Info("Redis Connection Pool start ready!!")

	//注册登陆相关实现
	m.GetServer().RegisterGO("checklogin", m.checklogin)   //模块间的rpc调用,登陆检测
	m.GetServer().RegisterGO("register", m.register)       //模块间的rpc调用，注册
	m.GetServer().RegisterGO("saveSession", m.saveSession) //模块间的rpc调用， 保存session，用来断线重连
	m.GetServer().RegisterGO("loadSession", m.loadSession) //模块间的rpc调用， 获取session，用来断线重连

	//大厅相关实现
	m.GetServer().RegisterGO("getUserData", m.getUserData)//信息的更新
	m.GetServer().RegisterGO("getUserInfo", m.getUserInfo)       //模块间的rpc调用， 获取用户信息
	m.GetServer().RegisterGO("getSuccessRank", m.getSuccessRank) //模块间的rpc调用， 获取成功抓取列表
	m.GetServer().RegisterGO("getPayRank", m.getPayRank)         //模块间的rpc调用， 获取充值排行榜列表
	m.GetServer().RegisterGO("getCheckInfo", m.getCheckInfo)     //模块间的rpc调用， 用户获取签到信息
	m.GetServer().RegisterGO("checkIn", m.checkIn)               //模块间的rpc调用， 用户签到
	m.GetServer().RegisterGO("getAddressInfo", m.getAddressInfo) //模块间的rpc调用， 用户获取签到信息
	m.GetServer().RegisterGO("addAddress", m.addAddress)         //模块间的rpc调用， 用户新增地址
	m.GetServer().RegisterGO("getWaWaInfo", m.getAllWaWa)        //模块间的rpc调用， 用户获取所有抓到的娃娃信息
	m.GetServer().RegisterGO("exchangeWawa", m.exchangeWawa)     //模块间的rpc调用， 用户将娃娃兑换为娃娃币
	m.GetServer().RegisterGO("askForDelivery", m.askForDelivery) //模块间的rpc调用， 用户申请发货

	//邮件相关rpc实现
	m.GetServer().RegisterGO("pushUserMail", m.pushUserMail)     //模块间的rpc调用， 存储用户邮件列表
	m.GetServer().RegisterGO("pushSystemMail", m.pushSystemMail) //模块间的rpc调用， 存储系统邮件列表
	m.GetServer().RegisterGO("readUserMail", m.readUserMail)     //模块间的rpc调用， 用户阅读邮件
	m.GetServer().RegisterGO("readSystemMail", m.readSystemMail) //模块间的rpc调用， 用户阅读系统邮件
	m.GetServer().RegisterGO("getMailLists", m.getMailLists)     //模块间的rpc调用， 获取用户邮件列表

	//娃娃机房间相关rpc实现
	m.GetServer().RegisterGO("updateUserGold", m.updateUserGold)         //模块间的rpc调用， 更新用户金币信息
	m.GetServer().RegisterGO("devicegolive", m.devicegolive)             //模块间的rpc调用， 更新或注册设备信息
	m.GetServer().RegisterGO("getDevicesInfo", m.getDevicesInfo)         //模块间的rpc调用， 获取所有设备信息
	m.GetServer().RegisterGO("changeDeviceConfig", m.changeDeviceConfig) //模块间的rpc调用， 修改设备配置信息
	m.GetServer().RegisterGO("recordSuccess", m.recordSuccess)           //模块间的rpc调用， 记录成功抓取
	m.GetServer().RegisterGO("getSuccessList", m.getSuccessList)         //模块间的rpc调用， 获取成功抓取列表
	m.GetServer().RegisterGO("recordPlay", m.recordPlay)                 //模块间的rpc调用， 记录游戏失败次数

	//后台相关rpc调用
	m.GetServer().RegisterGO("loginBG", m.loginBG)                       //模块间的rpc调用,后台登陆检测
	m.GetServer().RegisterGO("bgGetDevices", m.bgGetDevices)             //模块间的rpc调用,后台获取设备信息
	m.GetServer().RegisterGO("pubBG", m.pubBG)                           //模块间的rpc调用,后台获取公告信息
	m.GetServer().RegisterGO("changePub", m.changePub)                   //模块间的rpc调用,后台更改公告信息
	m.GetServer().RegisterGO("getActives", m.getActives)                 //模块间的rpc调用,后台获取活动信息
	m.GetServer().RegisterGO("changeActive", m.changeActive)             //模块间的rpc调用,后台更改活动信息
	m.GetServer().RegisterGO("getInviteConfig", m.inviteConfig)          //模块间的rpc调用,后台获取邀请奖励信息
	m.GetServer().RegisterGO("changeInviteConfig", m.changeInviteConfig) //模块间的rpc调用,后台更改邀请奖励信息
	m.GetServer().RegisterGO("getBGUserInfo", m.getBGUserInfo)           //模块间的rpc调用,后台获取注册用户信息
	m.GetServer().RegisterGO("getBGTradesInfo", m.getBGTradesInfo)       //模块间的rpc调用,后台获取订单信息
	m.GetServer().RegisterGO("getBGUnsendList", m.getBGUnsendList)       //模块间的rpc调用,后台获取中奖信息
	m.GetServer().RegisterGO("getBGAsksendList", m.getBGAsksendList)     //模块间的rpc调用,后台获取申请发货信息
	m.GetServer().RegisterGO("getBGSendedList", m.getBGSendedList)       //模块间的rpc调用,后台获取已发货信息
	m.GetServer().RegisterGO("snedPrize", m.snedPrize)                   //模块间的rpc调用， 后台发货
	m.GetServer().RegisterGO("getBGContackInfo", m.getBGContackInfo)     //模块间的rpc调用， 获取联系信息
	m.GetServer().RegisterGO("bgUpWechat", m.bgUpWechat)                 //模块间的rpc调用， 设置微信联系信息
	m.GetServer().RegisterGO("bgUpQQ", m.bgUpQQ)                         //模块间的rpc调用， 设置QQ联系信息
	m.GetServer().RegisterGO("bgUpKeFu", m.bgUpKeFu)                     //模块间的rpc调用， 设置公众号联系信息

	//商店相关rpc调用
	m.GetServer().RegisterGO("getShopList", m.getShopList)                 //模块间的rpc调用,获取商品信息
	m.GetServer().RegisterGO("changeShopConfig", m.changeShopConfig)       //模块间的rpc调用,获取商品信息
	m.GetServer().RegisterGO("getPlayerCardStates", m.getPlayerCardStates) //模块间的rpc调用,获取用户周卡月卡状态
	m.GetServer().RegisterGO("startBuy", m.startBuy)                       //模块间的rpc调用,开始购买商品
	m.GetServer().RegisterGO("updateBuy", m.updateBuy)                     //模块间的rpc调用,更新购买流程状态
	//m.GetServer().RegisterGO("buyItem", m.buyItem)         //模块间的rpc调用,获取商品信息
	m.GetServer().RegisterGO("buyDone", m.buyDone)       //模块间的rpc调用,检测是否购买成功
	m.GetServer().RegisterGO("buyDoneIOS", m.buyDoneIOS) //模块间的rpc调用,检测是否购买成功

	//邀请码相关rpc调用
	m.GetServer().RegisterGO("getInviteInfo", m.getInviteInfo) //模块间的rpc调用,获取邀请码信息
	m.GetServer().RegisterGO("bindUser", m.BindUser)           //模块间的rpc调用,获取邀请码信息

	//分享相关rpc
	m.GetServer().RegisterGO("shareDone", m.shareDone) //模块间的rpc调用， 分享完成获得奖励
}

func (m *redisDB) Run(closeSig chan bool) {
}

func (m *redisDB) OnDestroy() {
	//一定别忘了关闭RPC
	m.GetServer().OnDestroy()
	defer pool.Close()
}

//方便统计获取Redis Conn 获取时间
func (m *redisDB) getRedisCon() (c redis.Conn) {
	//before := time.Now()
	c = pool.Get()
	//log.Info("get redis connection time is %s", time.Since(before).String())
	return
}

//方便统计执行时间
func (m *redisDB) doCommand(c redis.Conn, command string, args ...interface{}) (reply interface{}, err error) {
	//before := time.Now()
	reply, err = c.Do(command, args...)
	//log.Info("redis command exec time is %s", time.Since(before).String())
	return
}
