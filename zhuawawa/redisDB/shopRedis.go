/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"fmt"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"

	redisMsg "zhuawawa/msg"
)

func (m *redisDB) getShopList() (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	bRet, _ := redis.Values(m.doCommand(c, "hgetall", "shopInfos"))

	shopItemNum := len(bRet) / 2
	allItems := new(redisMsg.SCAllShopItems)
	allItems.AllItems = make([]redisMsg.SCShopItemInfo, shopItemNum)
	for i := 0; i < shopItemNum; i++ {
		allItems.AllItems[i].UnmarshalJSON(bRet[2*i+1].([]uint8))
	}

	allItemsRet := new(redisMsg.SCAllShopItems)
	allItemsRet.AllItems = make([]redisMsg.SCShopItemInfo, shopItemNum)

	for i := 0; i < shopItemNum; i++ {
		idx, _ := strconv.Atoi(allItems.AllItems[i].ID)
		allItemsRet.AllItems[idx-1] = allItems.AllItems[i]
	}

	//log.Info("商品信息：%v -- %d ----%v", bRet, len(bRet), allItemsRet)
	buf, _ := allItemsRet.MarshalJSON()
	return string(buf), err
}
func (m *redisDB) changeShopConfig(newConfig []byte, itemID string) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	_, errRedis := m.doCommand(c, "hset", "shopInfos", itemID, newConfig)
	if errRedis != nil {
		return false, errRedis.Error()
	}
	return
}
func (m *redisDB) startBuy(TradeNum, userID string, cost, gold int64, nonceStr, shopDes string, shopID, itemType int64) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		redis.call("select", 1) --单独数据库存储订单信息
		local retTrade = redis.call("LINDEX", "TradeInfos", 0)--获取第一个元素，如果相同，则重新生成订单号
		if retTrade and retTrade == ARGV[1] then
			redis.call("select", 0)
			return 0
		end

		redis.call("LPUSH", "TradeInfos", ARGV[1])
		redis.call("hmset", ARGV[1], "ac", ARGV[2], "cost", ARGV[3], "gold", ARGV[4], "nonce", ARGV[5], "itemID", ARGV[6],"type", ARGV[7], "state", 0, "des", ARGV[8], "date", ARGV[9])
		redis.call("select", 0)
		return 1
		`)

	curDate := time.Now().Format("2006/01/02 15:04:05")
	ret, errRedis := redis.Int(luaScript.Do(c, TradeNum, userID, cost, gold, nonceStr, shopID+1, itemType, shopDes, curDate))
	if errRedis != nil {
		return false, errRedis.Error()
	}

	if ret == 1 {
		return true, ""
	}

	return false, ""

}

func (m *redisDB) updateBuy(TradeNum string, newState int64) (result bool, err string) {

	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		redis.call("select", 1) 
		local retInt = redis.call("hexists",  ARGV[1], "state")
		if retInt == 0 then
			redis.call("select", 0) 
			return 0
		end

		redis.call("hset", ARGV[1], "state", ARGV[2])
		redis.call("select", 0) 
		return 1
		`)

	ret, errRedis := redis.Int(luaScript.Do(c, TradeNum, newState))
	if errRedis != nil {
		return false, errRedis.Error()
	}

	if ret == 1 {
		return true, ""
	}
	return false, ""
}

//bQueryOrSet 代表主动查询或者设置购买结果，false： 客户端查询时只返回当前支付状态（1：成功，无需查询订单情况， 0：待处理，服务器主动向微信查询支付结果， -1： 支付失败）
//true:微信通知或者我们服务器主动查询支付结果，将该结果更新到数据库内，当第一次成功时更新金币以及周卡月卡状态！
func (m *redisDB) buyDone(tradeNumber string, bQueryOrSet bool, newState int64) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	curTime := time.Now()
	curDate := curTime.Format("20060102")

	luaScript := redis.NewScript(0, `
		redis.call("select", 1) 
		local nowState, acount, itemType, goldGot, cost, nonce = unpack(redis.call("hmget", ARGV[1], "state", "ac", "type", "gold", "cost", "nonce"))

		if (ARGV[3]) then
			if tonumber(nowState) == 1 then --第一次设置购买结果，重入时不再处理
				redis.call("hset", ARGV[1], "state", ARGV[4])
				redis.call("select", 0)

				if (tonumber(ARGV[4]) == 2) then
					goldGot = redis.call("hincrby", acount, "gold", goldGot)

					redis.call("zincrby", "payRank", cost, acount)
					--当只有一个成员时，更新超时时间
					if (redis.call("zcard", "payRank") == 1) then
						redis.call("expire", "payRank", ARGV[5])
					end

					if tonumber(itemType) == 1 then -- 周卡
						redis.call("hmset", acount, "weekCard", ARGV[2], "weekGet", "")
					elseif tonumber(itemType) == 2 then --月卡
						redis.call("hmset", acount, "monthCard", ARGV[2], "monthGet", "")
					end
					return {1, goldGot}
				else
					return {-1, 0}
				end
			end

			redis.call("select", 0)
			goldGot = redis.call("hget", acount, "gold")
			return {1, goldGot}
		else
			redis.call("select", 0)
			if tonumber(nowState) == 2 then
				goldGot = redis.call("hget", acount, "gold")
				return {1, goldGot}
			elseif tonumber(nowState) == 1 then 			--需要自己查询支付结果
				return {0, nonce}
			else--支付失败或者取消
				return {-1}
			end
		end

		`)

	timeSub := 0
	if newState == 2 {
		nowTime := time.Now()
		weekDay := nowTime.Weekday()
		//星期天，只算24点到现在的时间差,其他时间计算到下周一的天数乘以86400秒
		if weekDay != time.Sunday {
			timeSub = int((time.Saturday - weekDay + 1) * 86400)
		}
		timeSub += (23-nowTime.Hour())*3600 + (59-nowTime.Minute())*60 + (60 - nowTime.Second())
	}

	retTable, errRedis := redis.Values(luaScript.Do(c, tradeNumber, curDate, bQueryOrSet, newState, timeSub))

	if errRedis != nil {
		return "", errRedis.Error()
	}
	buyResult := new(redisMsg.SCBuyResult)

	buyResult.State, _ = redis.Int(retTable[0], nil)
	if buyResult.State == 1 {
		buyResult.Description = "购买成功！"
		buyResult.NowGold, _ = redis.Int(retTable[1], nil)
	} else if buyResult.State == -1 {
		buyResult.Description = "支付失败或取消，请重新购买！"
	} else if buyResult.State == 0 {
		buyResult.Description, _ = redis.String(retTable[1], nil)
	}
	buf, _ := buyResult.MarshalJSON()
	//log.Info("更新购买结果： %v", buyResult)
	return string(buf), ""
}

func (m *redisDB) buyDoneIOS(userID string, cost, gold int64, shopDes string, shopID int64) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		redis.call("select", 1) --单独数据库存储订单信息
		local retTrade = redis.call("LINDEX", "TradeInfos", 0)--获取第一个元素，如果相同，则重新生成订单号
		local tradeNum = ARGV[1]
		if retTrade and tonumber(retTrade) == tonumber(tradeNum) then
			tradeNum = tonumber(tradeNum) + 1
		end

		redis.call("LPUSH", "TradeInfos", tradeNum)
		redis.call("hmset", tradeNum, "ac", ARGV[2], "cost", ARGV[3], "gold", ARGV[4], "nonce", "IOSIAP", "itemID", ARGV[5],"type", 0, "state", 2, "des", ARGV[6], "date", ARGV[7])
		redis.call("select", 0)
		return redis.call("hincrby", ARGV[2], "gold", ARGV[4])
		`)

	curDate := time.Now().Format("2006/01/02 15:04:05")
	ret, errRedis := redis.Int(luaScript.Do(c, fmt.Sprintf("%d%d", time.Now().Unix(), shopID), userID, cost, gold, shopID+1, shopDes, curDate))

	if errRedis != nil {
		return "", errRedis.Error()
	}

	buyResult := new(redisMsg.SCBuyResult)

	buyResult.State = 1
	buyResult.Description = "购买成功！"
	buyResult.NowGold = ret

	buf, _ := buyResult.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getPlayerCardStates(uAc string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		return  redis.call("hmget", ARGV[1], "weekCard", "weekGet", "monthCard", "monthGet")
		`)
	//log.Info("uac is %s", uAc)
	retTable, errRedis := redis.Strings(luaScript.Do(c, uAc))
	//log.Info("%v", retTable)
	result = make(map[string]interface{})
	if errRedis != nil {
		return result, errRedis.Error()
	}

	curTime := time.Now()
	curDate := curTime.Format("20060102")

	cardNum, cardGotNum := 0, 0
	if retTable[0] != "" {
		cardNum++
		if retTable[0] != curDate && retTable[1] != curDate {
			cardGotNum++
		}
	}
	if retTable[2] != "" {
		cardNum += 2
		if retTable[2] != curDate && retTable[3] != curDate {
			cardGotNum += 2
		}
	}
	result["card"] = cardNum
	result["state"] = cardGotNum
	//log.Info("%v", result)
	return
}
