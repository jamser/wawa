/*Package shop 商店模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package shop

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/guidao/gopay"

	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"

	"github.com/guidao/gopay/common"
	"github.com/guidao/gopay/util"

	shopMsg "zhuawawa/msg"

	"github.com/guidao/gopay/client"
	"github.com/guidao/gopay/constant"
)

//Module 模块实例
var Module = func() module.Module {
	shop := new(Shop)
	return shop
}

//Shop 商店结构体
type Shop struct {
	basemodule.BaseModule
	shopInfos          *shopMsg.SCAllShopItems //商品信息
	IOSAppStoreVersion bool                    //是否AppStore审核版本
}

//GetType 模块类型
func (shop *Shop) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Shop"
}

//Version 模块版本
func (shop *Shop) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

//OnInit 模块初始化
func (shop *Shop) OnInit(app module.App, settings *conf.ModuleSettings) {
	shop.BaseModule.OnInit(shop, app, settings)

	shop.GetServer().RegisterGO("HD_GetShopList", shop.getShopList) //获取商店信息
	shop.GetServer().RegisterGO("HD_BuyItem", shop.buyItem)         //开始购买商品
	shop.GetServer().RegisterGO("HD_BuyDone", shop.buyDone)         //购买商品成功
	shop.GetServer().RegisterGO("HD_BuyDoneIOS", shop.buyDoneIOS)   //IAP内购购买商品成功（只在IOSAppStoreVersion版本中起作用）

	client.InitWechatH5Client(&client.WechatH5Client{
		AppID:       "wxa69e5412ab2eb702",                             // 公众账号ID
		MchID:       "1505305251",                                     // 商户号ID
		Key:         "0947714885BB9B8C13A14074E32894B7",               //秘钥
		CallbackURL: "http://BrainManPay.wingjoy.cn:3500/payNotify",   // 回调地址
		PayURL:      "https://api.mch.weixin.qq.com/pay/unifiedorder", // 统一下单地址
		QueryURL:    "https://api.mch.weixin.qq.com/pay/orderquery",   // 订单查询地址
	})

	shop.IOSAppStoreVersion = shop.GetModuleSettings().Settings["appStore"].(bool)
}

//Run 模块运行
func (shop *Shop) Run(closeSig chan bool) {
}

//OnDestroy 模块清理
func (shop *Shop) OnDestroy() {
	//一定别忘了关闭RPC
	shop.GetServer().OnDestroy()
}

func (shop *Shop) getShopList(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	if shop.shopInfos == nil {
		retListString, err := shop.RpcInvoke("RedisDB", "getShopList")
		if err != "" {
			log.Warning("%s", err)
			return shop.App.ProtocolMarshal("", err)
		}
		shop.shopInfos = new(shopMsg.SCAllShopItems)
		shop.shopInfos.UnmarshalJSON([]byte(retListString.(string)))
	}
	retState, err := shop.RpcInvoke("RedisDB", "getPlayerCardStates", session.GetUserid())

	retMap := retState.(map[string]interface{})
	retData := new(shopMsg.SCShopInfos)
	retData.CardState = int(retMap["card"].(float64))
	retData.GetState = int(retMap["state"].(float64))
	retData.AllItems = *shop.shopInfos
	buf, _ := retData.MarshalJSON()
	//log.Info("%v", retData)
	return shop.App.ProtocolMarshal(string(buf), "")
}

func (shop *Shop) buyItem(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	buyInfo := new(shopMsg.CSBuyItem)
	buyInfo.UnmarshalJSON(msg)

	shopID, _ := strconv.ParseInt(buyInfo.ShopID, 10, 64)
	shopID--
	charge := new(common.Charge)
	charge.PayMethod = constant.WECHATH5
	charge.MoneyFee = int64(1) //int64(shop.shopInfos.AllItems[shopID].Cost)
	charge.Describe = shop.shopInfos.AllItems[shopID].ShopDes
	charge.NonceStr = util.RandomStr()
	charge.TradeNum = fmt.Sprintf("%d%d", time.Now().Unix(), shopID)
	ipAndPort := strings.Split(session.GetIP(), ":")
	clientIP := ipAndPort[0]
	for i := 1; i < len(ipAndPort)-1; i++ {
		clientIP += ipAndPort[i]
	}
	charge.OpenID = clientIP //利用该字段保存客户端IP
	charge.CallbackURL = client.DefaultWechatH5Client().CallbackURL

	//log.Info("开始购买物品：%d -- %s----%d", shopID, charge.OpenID, shop.shopInfos.AllItems[shopID].Type)
	ret, err := shop.RpcInvoke("RedisDB", "startBuy", charge.TradeNum, session.GetUserid(), int64(shop.shopInfos.AllItems[shopID].Cost), int64(shop.shopInfos.AllItems[shopID].Gold), charge.NonceStr, shop.shopInfos.AllItems[shopID].ShopDes, shopID, int64(shop.shopInfos.AllItems[shopID].Type))
	for ret.(bool) == false {
		charge.TradeNum = fmt.Sprintf("%d%d", time.Now().Unix(), shopID)
		ret, err = shop.RpcInvoke("RedisDB", "startBuy", charge.TradeNum, session.GetUserid(), int64(shop.shopInfos.AllItems[shopID].Cost), int64(shop.shopInfos.AllItems[shopID].Gold), charge.NonceStr, shop.shopInfos.AllItems[shopID].ShopDes, shopID, int64(shop.shopInfos.AllItems[shopID].Type))
	}

	wechatH5Url, errWechatH5 := gopay.Pay(charge)

	//log.Info("url is： %s", wechatH5Url)
	if errWechatH5 != nil {
		log.Info("获取prePayID失败：%s", errWechatH5.Error())
		return shop.App.ProtocolMarshal("获取prePayID失败:"+errWechatH5.Error(), "")
	}

	ret, err = shop.RpcInvoke("RedisDB", "updateBuy", charge.TradeNum, int64(1))
	if ret.(bool) == false {
		log.Warning("更新购买状态失败！购买出错！")
		return shop.App.ProtocolMarshal("111", "")
	}

	retInfo := new(shopMsg.SCPrePayID)
	retInfo.PrePayID = wechatH5Url //fmt.Sprintf("%s&redirect_url=%s", wechatH5Url, url.QueryEscape("http://BrainManPay.wingjoy.cn:3500/payRet"))
	//log.Info("new url is : %s", retInfo.PrePayID)
	retInfo.TradeNumber = charge.TradeNum
	//log.Info("tradeNum is : %s", retInfo.TradeNumber)
	buf, _ := retInfo.MarshalJSON()
	return shop.App.ProtocolMarshal(string(buf), err)
}

func (shop *Shop) buyDone(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	tradeInfo := new(shopMsg.CSBuyDone)
	tradeInfo.UnmarshalJSON(msg)

	ret, err := shop.RpcInvoke("RedisDB", "buyDone", tradeInfo.TradeNumber, false, int64(0))
	if err != "" {
		log.Error("支付出错：%s", err)
		return
	}
	buyRet := new(shopMsg.SCBuyResult)
	buyRet.UnmarshalJSON([]byte(ret.(string)))
	if buyRet.State == 0 { //发起订单查询
		charge := new(common.Charge)
		charge.PayMethod = constant.WECHATH5
		charge.NonceStr = buyRet.Description //使用该变量暂时存储随机字符串

		for {
			//log.Info("开始查询订单 ：%s 随机字符串：%s", tradeInfo.TradeNumber, charge.NonceStr)
			retJSON, errH5Pay := gopay.OrderQuery(charge, tradeInfo.TradeNumber)

			if errH5Pay != nil {
				if errH5Pay.Error() == "SYSTEMERROR" {
					log.Info("%s : 系统错误，重新发起查询！", errH5Pay)
					continue
				}
			}
			//log.Info("查询订单结束 ：%s", tradeInfo.TradeNumber)
			retWechat, _ := retJSON.(*common.WeChatQueryResult)
			//log.Info("查询订单结果： %v", retWechat)
			if retWechat.TradeState == "SUCCESS" {
				ret, err = shop.RpcInvoke("RedisDB", "buyDone", tradeInfo.TradeNumber, true, int64(2)) //查询结果：支付成功
			} else {
				ret, err = shop.RpcInvoke("RedisDB", "buyDone", tradeInfo.TradeNumber, true, int64(3)) //查询结果：失败或取消支付
			}
			return shop.App.ProtocolMarshal(ret.(string), err)
		}
	} else {
		return shop.App.ProtocolMarshal(ret.(string), err)
	}
}

func (shop *Shop) buyDoneIOS(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	if shop.IOSAppStoreVersion == false {
		return shop.App.ProtocolMarshal("", "非IOS IAP环境该函数不应该被调用！")
	}

	tradeInfo := new(shopMsg.CSBuyDone)
	tradeInfo.UnmarshalJSON(msg)
	//ios iap内购传过来的值从1开始， 对应id为3的商品， 所以id++才是该商品信息
	shopID, _ := strconv.ParseInt(tradeInfo.TradeNumber, 10, 64)
	shopID++
	ret, err := shop.RpcInvoke("RedisDB", "buyDoneIOS", session.GetUserid(), int64(shop.shopInfos.AllItems[shopID].Cost), int64(shop.shopInfos.AllItems[shopID].Gold), shop.shopInfos.AllItems[shopID].ShopDes, shopID)
	if err != "" {
		log.Error("支付出错：%s", err)
		return
	}

	return shop.App.ProtocolMarshal(ret.(string), err)
}
