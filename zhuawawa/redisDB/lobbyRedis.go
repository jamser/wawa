/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"fmt"
	"strconv"
	"time"
	redisMsg "zhuawawa/msg"

	"github.com/garyburd/redigo/redis"
	"github.com/liangdas/mqant/log"
)

func (m *redisDB) getUserInfo(userName string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	r, errRedis := redis.Values(m.doCommand(c, "HMGET", []interface{}{userName, "id", "nick", "head", "sex", "gold"}...))
	if errRedis != nil {
		log.Warning("获取用户%s信息有误！ %s", userName, errRedis.Error())
		err = "获取用户详细信息出错！"
		return
	}

	result = make(map[string]interface{})
	result["id"], _ = redis.Int64(r[0], nil)
	result["nick"], _ = redis.String(r[1], nil)
	result["head"], _ = redis.String(r[2], nil)
	result["sex"], _ = redis.Bool(r[3], nil)
	result["gold"], _ = redis.Int64(r[4], nil)

	return
}

func (m *redisDB) getSuccessRank(uid string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retTable = {-1,0}
		
		retTable[2] = redis.call("zscore", "successRank", ARGV[1])

		if (not retTable[2]) then
			retTable[2] = 0
		else	
			retTable[2] = tonumber(retTable[2])		
			retTable[1] = tonumber(redis.call("Zrevrank", "successRank", ARGV[1]))
		end

		local retScoreTable = redis.call("ZREVRANGE", "successRank", 0, 9, "WITHSCORES")
		local idx = 2
		for i=1, #retScoreTable, 2 do
			retTable[idx+1], retTable[idx+2] = unpack(redis.call("hmget", retScoreTable[i], "nick", "head"))
			retTable[idx+3] = tonumber(retScoreTable[i+1])
			idx = idx + 3
		end

		return retTable
	`)

	//retInfo, errRedis := redis.Strings(m.doCommand(c, "ZREVRANGE", "successRank", 0, 9, "WITHSCORES"))
	retInfo, errRedis := redis.Values(luaScript.Do(c, uid))
	//log.Info("%v", retInfo)
	if errRedis != nil {
		err = "获取成功记录排行榜有误：" + errRedis.Error()
		return "", err
	}
	recordLen := (len(retInfo) - 2) / 3
	//log.Info("排行榜个数为：%d", recordLen)
	retList := new(redisMsg.SCSuccessRank)
	retList.SelfRank = int(retInfo[0].(int64)) + 1
	retList.SelfScore = int(retInfo[1].(int64))
	retList.SuccessCount = make([]string, recordLen)
	retList.UserNicks = make([]string, recordLen)
	retList.UserHead = make([]string, recordLen)

	for i := 0; i < recordLen; i++ {
		retList.UserNicks[i] = string(retInfo[(i+1)*3-1].([]uint8))
		retList.UserHead[i] = string(retInfo[(i+1)*3].([]uint8))
		retList.SuccessCount[i] = strconv.FormatInt(retInfo[(i+1)*3+1].(int64), 10)
	}

	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getPayRank(uid string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retTable = {-1,0}
		
		retTable[2] = redis.call("zscore", "payRank", ARGV[1])

		if (not retTable[2]) then
			retTable[2] = 0
		else	
			retTable[2] = tonumber(retTable[2])		
			retTable[1] = tonumber(redis.call("Zrevrank", "payRank", ARGV[1]))
		end

		local retScoreTable = redis.call("ZREVRANGE", "payRank", 0, 9, "WITHSCORES")
		local idx = 2
		for i=1, #retScoreTable, 2 do
			retTable[idx+1], retTable[idx+2] = unpack(redis.call("hmget", retScoreTable[i], "nick", "head"))
			retTable[idx+3] = tonumber(retScoreTable[i+1])
			idx = idx + 3
		end
		return retTable
	`)

	//retInfo, errRedis := redis.Strings(m.doCommand(c, "ZREVRANGE", "successRank", 0, 9, "WITHSCORES"))
	retInfo, errRedis := redis.Values(luaScript.Do(c, uid))
	//log.Info("%v", retInfo)
	if errRedis != nil {
		err = "获取充值记录排行榜有误：" + errRedis.Error()
		return "", err
	}
	recordLen := (len(retInfo) - 2) / 3
	//log.Info("排行榜个数为：%d", recordLen)
	retList := new(redisMsg.SCSuccessRank)
	retList.SelfRank = int(retInfo[0].(int64)) + 1
	retList.SelfScore = int(retInfo[1].(int64))
	retList.SuccessCount = make([]string, recordLen)
	retList.UserNicks = make([]string, recordLen)
	retList.UserHead = make([]string, recordLen)

	for i := 0; i < recordLen; i++ {
		retList.UserNicks[i] = string(retInfo[(i+1)*3-1].([]uint8))
		retList.UserHead[i] = string(retInfo[(i+1)*3].([]uint8))
		retList.SuccessCount[i] = strconv.FormatInt(retInfo[(i+1)*3+1].(int64), 10)
	}

	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func timeSub(t1, t2 time.Time) int {
	t1 = t1.UTC().Truncate(24 * time.Hour)
	t2 = t2.UTC().Truncate(24 * time.Hour)
	return int(t1.Sub(t2).Hours() / 24)
}

func (m *redisDB) getCheckInfo(userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	curTime := time.Now()
	curDate := curTime.Format("20060102")

	preTime := curTime.AddDate(0, 0, -7)
	//preDate := fmt.Sprintf("%d%d%d", preTime.Year(), int(preTime.Month()), preTime.Day())
	preDate := preTime.Format("20060102")
	luaScript := redis.NewScript(0, `
		local signCount, startDate, gold = unpack(redis.call("hmget", ARGV[1], "keepSign", "startSign", "gold"))
		
		local startDay, preDay, nowDay, sign = tonumber(startDate), tonumber(ARGV[2]), tonumber(ARGV[3]), tonumber(signCount)


		if (startDay < preDay) or ( nowDay - startDay < sign -1  or nowDay - startDay > sign ) then
			redis.call("hmset", ARGV[1], "keepSign" , 0, "startSign", 0)
			return {0, tonumber(gold), 0}
		end

		return {sign, tonumber(gold), startDay}
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	ret, errRedis := redis.Values(luaScript.Do(c, userAc, preDate, curDate))

	if errRedis != nil {
		return "", "获取签到信息有误：" + errRedis.Error()
	}

	chectRet := new(redisMsg.SCCheckInCount)
	chectRet.Count = int(ret[0].(int64))
	chectRet.Gold = int(ret[1].(int64))
	if chectRet.Count > 0 {
		startTime, _ := time.Parse("20060102", strconv.FormatInt(ret[2].(int64), 10))
		if (timeSub(curTime, startTime) + 1) == chectRet.Count {
			chectRet.Today = true
		} else {
			chectRet.Today = false
		}
	} else {
		chectRet.Today = false
	}

	buf, _ := chectRet.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) checkIn(userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	curTime := time.Now()
	curDate := curTime.Format("20060102")

	preTime := curTime.AddDate(0, 0, -7)
	preDate := preTime.Format("20060102")

	luaScript := redis.NewScript(0, `
		local signCount, startDate = unpack(redis.call("hmget", ARGV[1], "keepSign", "startSign"))
		local cursingCount = tonumber(signCount)
		if (tonumber(startDate) < tonumber(ARGV[3])) or (cursingCount == 7)  then
			redis.call("hmset", ARGV[1], "keepSign" , 1, "startSign", ARGV[2])
			local goldGot = redis.call("hget", "SignReward", "Day"..1)
			local goldNum = redis.call("HIncrby", ARGV[1], "gold", goldGot)
			return {1, goldNum}
		end

		local goldGot = redis.call("hget", "SignReward", "Day"..(1 + cursingCount))
		local goldNum = redis.call("HIncrby", ARGV[1], "gold", goldGot)
		redis.call("hset", ARGV[1], "keepSign" , 1 + cursingCount)
		return {1 + cursingCount, goldNum}
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	ret, errRedis := redis.Values(luaScript.Do(c, userAc, curDate, preDate))
	if errRedis != nil {
		return "", "签到有误：" + errRedis.Error()
	}
	chectRet := new(redisMsg.SCCheckInCount)
	chectRet.Count = int(ret[0].(int64))
	chectRet.Gold = int(ret[1].(int64))
	chectRet.Today = true
	buf, _ := chectRet.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getAddressInfo(userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local addrs = redis.call("hget", ARGV[1], "addrs")
		if not addrs then
			return ""	
		end

		return addrs
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	ret, errRedis := redis.String(luaScript.Do(c, userAc))
	if errRedis != nil {
		log.Warning("获取用户（%s）地址信息有误！ %s", userAc, errRedis.Error())
		err = "获取用户地址信息出错！"
	}

	//log.Info("%v", ret)
	retAddr := new(redisMsg.SCAddresss)
	if ret == "" {
		retAddr.Addresss = make([]redisMsg.SCAddressInfo, 0)
		buf, _ := retAddr.MarshalJSON()
		return string(buf), ""
	}

	return ret, ""
}

func (m *redisDB) addAddress(userAc string, msg []byte) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	_, errRedis := m.doCommand(c, "hset", userAc, "addrs", string(msg))
	if errRedis != nil {
		return "failed", "添加或更新地址信息有误：" + errRedis.Error()
	}

	return "ok", ""
}

func (m *redisDB) getAllWaWa(userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local IDs = redis.call("LRANGE", ARGV[1].."-unSendList", 0, -1)
		local retTable, idx, wawaInfo = {}, 1, ""
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] = tostring(1) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					wawaInfo = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = wawaInfo["thub"], wawaInfo["des"], tostring(wawaInfo["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[1].."-unSendList", 1, v)
					redis.call("LRem", "100000-unSendList", 1,  v)
				end			
			end
		end
		
		IDs = redis.call("LRANGE", ARGV[1].."-askSendList", 0, -1)
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] = tostring(2) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					wawaInfo = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = wawaInfo["thub"], wawaInfo["des"], tostring(wawaInfo["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[1].."-askSendList", 1, v)
					redis.call("del", v.."-Addr") -- 删除娃娃发货地址信息
					redis.call("LRem", "100000-askSendList", 1,  v)
				end			
			end
		end

		IDs = redis.call("LRANGE", ARGV[1].."-SendedList", 0, -1)
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] =tostring(3) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					wawaInfo = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = wawaInfo["thub"], wawaInfo["des"], tostring(wawaInfo["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[1].."-SendedList", 1, v)
					redis.call("del", v.."-Addr") -- 删除娃娃发货地址信息
					redis.call("LRem", "100000-SendedList", 1,  v)
				end			
			end
		end

		return retTable
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	ret, errRedis := redis.Strings(luaScript.Do(c, userAc))
	if errRedis != nil {
		log.Warning("获取用户（%s）娃娃信息有误！ %s", userAc, errRedis.Error())
		err = "获取用户娃娃信息出错！"
	}

	//log.Info("%s - %s - %s - %s", ret[0], ret[1], ret[2], ret[3])

	retData := new(redisMsg.SCWawaList)
	len := len(ret) / 5
	retData.Records = make([]redisMsg.SCSuccessRecord, len)
	retData.State = make([]int, len)
	retData.Thub = make([]string, len)
	retData.Name = make([]string, len)
	retData.Exchange = make([]int, len)
	for i := 0; i < len; i++ {
		retData.Records[i].UnmarshalJSON([]byte(ret[i*5]))
		retData.State[i], _ = strconv.Atoi(ret[i*5+1])
		retData.Thub[i] = ret[i*5+2]
		retData.Name[i] = ret[i*5+3]
		retData.Exchange[i], _ = strconv.Atoi(ret[i*5+4])
	}
	buf, _ := retData.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) exchangeWawa(userAc string, msg []byte) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	exchangeMsg := new(redisMsg.CSWaWaExchange)
	exchangeMsg.UnmarshalJSON(msg)

	luaScript := redis.NewScript(0, `
		local retTable, idx = {}, 1

		--得到娃娃索引对应的id
		local wawID = redis.call("LIndex", ARGV[1].."-unSendList", ARGV[2])
		--从玩家库存列表中删除该ID
		redis.call("LRem", ARGV[1].."-unSendList", 1,wawID)
		--从系统库存列表中删除该ID
		redis.call("LRem", "100000-unSendList", 1, wawID)
		
		--获得相关信息，兑换的金币数，并增加玩家当前金币数量
		local wawaInfo = redis.call("get", wawID)
		--删除该娃娃信息
		redis.call("del", wawID)
		wawaInfo = cjson.decode(wawaInfo)
		local deviceID = wawaInfo["deviceID"]
		--娃娃可兑换金币数量
		--deviceID = redis.call("hget", "dev"..deviceID, "exchange")
		deviceID = cjson.decode(redis.call("get", "dev"..deviceID))
		deviceID = deviceID["exchange"]
		--玩家更新之后金币数量
		retTable[1] = tostring(redis.call("hincrby", ARGV[1], "gold", deviceID))
		idx = 2
		local IDs = redis.call("LRANGE", ARGV[1].."-unSendList", 0, -1)
		
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] = tostring(1) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					deviceID = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = deviceID["thub"], deviceID["des"], tostring(deviceID["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[1].."-unSendList", 1, v)
					redis.call("LRem", "100000-unSendList", 1,  v)
				end			
			end
		end
		
		IDs = redis.call("LRANGE", ARGV[1].."-askSendList", 0, -1)
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] = tostring(2) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					deviceID = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = deviceID["thub"], deviceID["des"], tostring(deviceID["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[1].."-askSendList", 1, v)
					redis.call("del", v.."-Addr") -- 删除娃娃发货地址信息
					redis.call("LRem", "100000-askSendList", 1,  v)
				end			
			end
		end

		IDs = redis.call("LRANGE", ARGV[1].."-SendedList", 0, -1)
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] =tostring(3) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					deviceID = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = deviceID["thub"], deviceID["des"], tostring(deviceID["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[1].."-SendedList", 1, v)
					redis.call("del", v.."-Addr") -- 删除娃娃发货地址信息
					redis.call("LRem", "100000-SendedList", 1,  v)
				end			
			end
		end

		return retTable
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	ret, errRedis := redis.Strings(luaScript.Do(c, userAc, exchangeMsg.WaWaID))
	if errRedis != nil {
		log.Warning("用户（%s）娃娃兑换娃娃币有误！ %s", userAc, errRedis.Error())
		err = "娃娃兑换娃娃币出错！"
	}

	//log.Info("%s - %s - %s - %s", ret[0], ret[1], ret[2], ret[3])

	retData := new(redisMsg.SCWaWaExchangeRet)
	retData.Gold, _ = strconv.Atoi(ret[0])
	len := (len(ret) - 1) / 5
	retData.List.Records = make([]redisMsg.SCSuccessRecord, len)
	retData.List.State = make([]int, len)
	retData.List.Thub = make([]string, len)
	retData.List.Name = make([]string, len)
	retData.List.Exchange = make([]int, len)
	for i := 0; i < len; i++ {
		retData.List.Records[i].UnmarshalJSON([]byte(ret[i*5+1]))
		retData.List.State[i], _ = strconv.Atoi(ret[i*5+2])
		retData.List.Thub[i] = ret[i*5+3]
		retData.List.Name[i] = ret[i*5+4]
		retData.List.Exchange[i], _ = strconv.Atoi(ret[i*5+5])
	}
	buf, _ := retData.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) askForDelivery(userAc string, msg []byte) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	deliverMsg := new(redisMsg.CSAskDelivery)
	deliverMsg.UnmarshalJSON(msg)

	luaScript := redis.NewScript(0, `
		local retTable, idx = {}, 1

		local resultStrList = {}
		string.gsub(ARGV[1],'[^:]+',function ( w )
			table.insert(resultStrList,1 , w)
		end)

		if #resultStrList == 1 then
			--需要花费128娃娃币
			--扣除128娃娃币
			retTable[1] = tostring(redis.call("hincrby", ARGV[2], "gold", -128))
			
		else
			retTable[1] = tostring(redis.call("hget", ARGV[2], "gold"))
		end

		idx = 2
		
		for k,v in ipairs(resultStrList) do
			--得到娃娃索引对应的id
			local wawaID = redis.call("LIndex", ARGV[2].."-unSendList", v)
			--从玩家库存列表中删除该ID
			redis.call("LRem", ARGV[2].."-unSendList", 1,wawaID)
			redis.call("LPUSH", ARGV[2].."-askSendList", wawaID)
			--从系统库存列表中删除该ID
			redis.call("LRem", "100000-unSendList", 1, wawaID)
			redis.call("LPUSH", "100000-askSendList", wawaID)
			
			--设置该娃娃发货地址信息
			redis.call("hmset", wawaID.."-Addr", "name", ARGV[3], "tel", ARGV[4], "addr", ARGV[5])
		end
		
		
		local IDs = redis.call("LRANGE", ARGV[2].."-unSendList", 0, -1)
		
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] = tostring(1) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					data = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = data["thub"], data["des"], tostring(data["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[2].."-unSendList", 1, v)
					redis.call("LRem", "100000-unSendList", 1,  v)
				end			
			end
		end
		
		IDs = redis.call("LRANGE", ARGV[2].."-askSendList", 0, -1)
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] = tostring(2) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					data = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = data["thub"], data["des"], tostring(data["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[2].."-askSendList", 1, v)
					redis.call("del", v.."-Addr") -- 删除娃娃发货地址信息
					redis.call("LRem", "100000-askSendList", 1,  v)
				end			
			end
		end

		IDs = redis.call("LRANGE", ARGV[2].."-SendedList", 0, -1)
		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then
					retTable[idx+1] =tostring(3) --表示是库存状态
					local data = cjson.decode(retTable[idx])
					--retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", "dev"..data["deviceID"], "thub", "des", "exchange"))
					data = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+2],retTable[idx+3],retTable[idx+4] = data["thub"], data["des"], tostring(data["exchange"])
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", ARGV[2].."-SendedList", 1, v)
					redis.call("del", v.."-Addr") -- 删除娃娃发货地址信息
					redis.call("LRem", "100000-SendedList", 1,  v)
				end			
			end
		end

		return retTable
	`)

	//log.Info("%s", deliverMsg.DeliveryIDs)

	ret, errRedis := redis.Strings(luaScript.Do(c, deliverMsg.DeliveryIDs, userAc, deliverMsg.Address.Name, deliverMsg.Address.Tel,
		fmt.Sprintf("%s %s", deliverMsg.Address.Area, deliverMsg.Address.Addr)))
	if errRedis != nil {
		log.Warning("获取用户（%s）娃娃信息有误！ %s", userAc, errRedis.Error())
		err = "获取用户娃娃信息出错！"
	}

	//log.Info("%s - %s - %s - %s", ret[0], ret[1], ret[2], ret[3])
	//log.Info("%v", ret)

	retData := new(redisMsg.SCWaWaExchangeRet)
	retData.Gold, _ = strconv.Atoi(ret[0])
	len := (len(ret) - 1) / 5
	//log.Info("个数： %d  金币： %d", len, retData.Gold)
	retData.List.Records = make([]redisMsg.SCSuccessRecord, len)
	retData.List.State = make([]int, len)
	retData.List.Thub = make([]string, len)
	retData.List.Name = make([]string, len)
	retData.List.Exchange = make([]int, len)
	for i := 0; i < len; i++ {
		retData.List.Records[i].UnmarshalJSON([]byte(ret[i*5+1]))
		retData.List.State[i], _ = strconv.Atoi(ret[i*5+2])
		retData.List.Thub[i] = ret[i*5+3]
		retData.List.Name[i] = ret[i*5+4]
		retData.List.Exchange[i], _ = strconv.Atoi(ret[i*5+5])
	}
	buf, _ := retData.MarshalJSON()
	return string(buf), ""
}
