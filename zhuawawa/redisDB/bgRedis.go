/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"strconv"
	"strings"
	redisMsg "zhuawawa/msg"

	"github.com/garyburd/redigo/redis"
	"github.com/liangdas/mqant/log"
)

func (m *redisDB) loginBG(uID, pwd string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retTable = {"fail","",""}
	
		local pwd, nick = unpack(redis.call("hmget", ARGV[1], "agPwd", "nick"))
				
		if (pwd == ARGV[2]) 
		then
			retTable[1] = "ok"
			retTable[2] = nick
			retTable[3] = redis.call("get", "PubNotice")
			return retTable
		end

		return retTable
	`)
	log.Info("loginBG called")
	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	retTable, errRedis := redis.Strings(luaScript.Do(c, uID, pwd))

	if errRedis != nil {
		err = "后台登陆有误：" + errRedis.Error()
		return
	}

	result = make(map[string]interface{}, 3)
	result["state"] = retTable[0]
	result["nick"] = retTable[1]
	result["pub"] = retTable[2]
	return result, ""
}

func (m *redisDB) bgGetDevices() (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retTable = {}
		local deviceIdTable = redis.call("Smembers", "allDevices")
		
		for _, device in ipairs(deviceIdTable) do
			--retTable[#retTable + 1] = redis.call("hmget", device, "id", "group", "sub", "des", "thub", "cost", "state", "play", "success")
			retTable[#retTable + 1] = redis.call("get", device)
		end
		 
		return retTable
	`)

	log.Info("bgGetDevices called")
	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	retTable, errRedis := redis.Strings(luaScript.Do(c))

	result = make(map[string]interface{})
	//log.Info("%v", retTable)
	if errRedis != nil {
		err = "获取设备信息有误：" + errRedis.Error()
		return
	}
	for i := 0; i < len(retTable); i++ {
		result[strconv.Itoa(i)] = strings.Replace(retTable[i], "\\", "", -1)
	}
	/*
		for i := 0; i < len(retTable); i++ {
			oneDeviceMap := make(map[string]string)
			oneDevice := retTable[i].([]interface{})
			oneDeviceMap["id"] = string(oneDevice[0].([]uint8))
			oneDeviceMap["group"] = string(oneDevice[1].([]uint8))
			oneDeviceMap["sub"] = string(oneDevice[2].([]uint8))
			if oneDeviceMap["sub"] == "0" {
				oneDeviceMap["sub"] = "玩偶"
			} else if oneDeviceMap["sub"] == "1" {
				oneDeviceMap["sub"] = "酒水"
			}
			oneDeviceMap["des"] = string(oneDevice[3].([]uint8))
			oneDeviceMap["thub"] = string(oneDevice[4].([]uint8))
			oneDeviceMap["cost"] = string(oneDevice[5].([]uint8))
			oneDeviceMap["state"] = string(oneDevice[6].([]uint8))
			oneDeviceMap["play"] = string(oneDevice[7].([]uint8))
			oneDeviceMap["success"] = string(oneDevice[8].([]uint8))

			result[oneDeviceMap["id"]] = oneDeviceMap
		}
	*/
	return
}

func (m *redisDB) pubBG(uID string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		return {redis.call("hget", ARGV[1], "nick"),   redis.call("get", "PubNotice")}
	`)
	retTable, errRedis := redis.Strings(luaScript.Do(c, uID))

	if errRedis != nil {
		err = "后台获取公告信息有误：" + errRedis.Error()
		return
	}

	result = make(map[string]interface{}, 2)
	result["nick"] = retTable[0]
	result["pub"] = retTable[1]
	return result, ""
}

func (m *redisDB) changePub(uID, newPub string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		redis.call("set", "PubNotice", ARGV[2])
		return {redis.call("hget", ARGV[1], "nick"),   ARGV[2]}
	`)
	retTable, errRedis := redis.Strings(luaScript.Do(c, uID, newPub))

	if errRedis != nil {
		err = "后台修改公告信息有误：" + errRedis.Error()
		return
	}

	result = make(map[string]interface{}, 2)
	result["nick"] = retTable[0]
	result["pub"] = retTable[1]

	m.RpcInvokeNR("Gate", "BroadCast", "Update/pubChange", []byte(retTable[1]))
	return result, ""
}

func (m *redisDB) getActives() (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		return redis.call("hmget", "actives", "1", "2", "3", "4")
	`)

	retTable, errRedis := redis.Strings(luaScript.Do(c))

	if errRedis != nil {
		err = "后台获取活动信息有误：" + errRedis.Error()
		return
	}

	retData := new(redisMsg.SCActivesInfo)
	retData.Actives = make([]redisMsg.SCActiveInfo, 4)
	for i := 0; i < 4; i++ {
		retData.Actives[i].ImgURL = retTable[i]
		retData.Actives[i].ActiveID = i + 1
	}
	buf, _ := retData.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) changeActive(newActive string) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	retData := new(redisMsg.SCActiveInfo)
	retData.UnmarshalJSON([]byte(newActive))

	_, errRedis := m.doCommand(c, "hset", "actives", retData.ActiveID, retData.ImgURL)

	if errRedis != nil {
		err = "后台修改活动信息有误：" + errRedis.Error()
		return false, err
	}

	m.RpcInvokeNR("Gate", "BroadCast", "Update/activeChange", []byte(newActive))
	return true, ""
}

func (m *redisDB) inviteConfig() (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	retTable, errRedis := redis.Values(m.doCommand(c, "hmget", "bindInfos", "userGain", "binderGain", "goldGainMaxUser", "oneMaxMoney", "totalMoney", "moneyRate"))

	if errRedis != nil {
		err = "后台获取邀请奖励信息有误：" + errRedis.Error()
		return
	}

	result = make(map[string]interface{}, 6)
	result["userGain"], _ = redis.Int(retTable[0], nil)
	result["binderGain"], _ = redis.Int(retTable[1], nil)
	result["goldGainMaxUser"], _ = redis.Int(retTable[2], nil)
	result["oneMaxMoney"], _ = redis.Int(retTable[3], nil)
	result["totalMoney"], _ = redis.Int(retTable[4], nil)
	result["moneyRate"], _ = redis.Float64(retTable[5], nil)
	return
}

func (m *redisDB) changeInviteConfig(keyStr, newValue string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		redis.call("hset", "bindInfos", ARGV[1], ARGV[2])
		return redis.call("hmget", "bindInfos", "userGain", "binderGain", "goldGainMaxUser", "oneMaxMoney", "totalMoney", "moneyRate")
		`)

	retTable, errRedis := redis.Values(luaScript.Do(c, keyStr, newValue))
	if errRedis != nil {
		err = "后台修改公告信息有误：" + errRedis.Error()
		return
	}
	//log.Info("changeInviteConfig : %v", retTable)
	result = make(map[string]interface{}, 6)
	result["userGain"], _ = redis.Int(retTable[0], nil)
	result["binderGain"], _ = redis.Int(retTable[1], nil)
	result["goldGainMaxUser"], _ = redis.Int(retTable[2], nil)
	result["oneMaxMoney"], _ = redis.Int(retTable[3], nil)
	result["totalMoney"], _ = redis.Int(retTable[4], nil)
	result["moneyRate"], _ = redis.Float64(retTable[5], nil)
	return
}

func (m *redisDB) getBGUserInfo(userCurPage string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retTable = {}
		local uAc, idx = "", 1

		for i=tonumber(ARGV[1]), tonumber(ARGV[2]) do
			uAc = redis.call("HGET", "AllUsers", i)
			if uAc then
				retTable[idx],retTable[idx+1],retTable[idx+2],retTable[idx+3],retTable[idx+4] = unpack(redis.call("hmget", uAc, "id", "nick", "gold", "date", "sex"))
				idx=idx+5				
			else
				break
			end
			
		end

		retTable[idx] = redis.call("HLEN", "AllUsers")
		return retTable
		`)

	curPage, _ := strconv.Atoi(userCurPage)
	//log.Info("%v--%v", 100001+(curPage-1)*10, 100000+curPage*10)
	usersList, errRedis := redis.Values(luaScript.Do(c, 100001+(curPage-1)*10, 100000+curPage*10))
	//log.Info("%v -- %d", usersList, len(usersList))

	if errRedis != nil {
		err = "获取邮箱数据有误：" + errRedis.Error()
		return "", err
	}

	retList := new(redisMsg.SCBGUserInfoSum)

	listLen := (len(usersList) - 1) / 5

	retList.Users = make([]redisMsg.SCBGUserInfo, listLen)

	for i := 0; i < listLen; i++ {
		retList.Users[i].ID, _ = strconv.ParseInt(string(usersList[i*5].([]uint8)), 10, 64)
		retList.Users[i].NickName = string(usersList[i*5+1].([]uint8))
		retList.Users[i].Gold, _ = strconv.ParseInt(string(usersList[i*5+2].([]uint8)), 10, 64)
		retList.Users[i].RegisterDate = string(usersList[i*5+3].([]uint8))
		retList.Users[i].Gender, _ = redis.Bool(usersList[i*5+4], nil)
	}

	retList.CurPage = curPage
	retList.TotalPage = int(usersList[listLen*5].(int64))
	retList.TotalPage = (retList.TotalPage / 11) + 1
	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getBGTradesInfo(tradeCurPage string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retTable = {}
		local trades, idx = {}, 3
		redis.call("select", 1)
		trades = redis.call("LRange", "TradeInfos", tonumber(ARGV[1]), tonumber(ARGV[2]))
		local tradeNum = #trades
		retTable[1] = tostring(tradeNum)
		retTable[2] = tostring(redis.call("LLEN", "TradeInfos"))

		if tradeNum > 0 then			
			for _,v in ipairs(trades) do
				retTable[idx],retTable[idx+1], retTable[idx+2], retTable[idx+3],retTable[idx+4],retTable[idx+5],retTable[idx+6],retTable[idx+7]= 
					unpack(redis.call("hmget", v, "ac", "cost", "gold", "itemID", "type", "state", "des", "date"))
				idx = idx+8
			end
			redis.call("select", 0)

			for i=3, 8*tradeNum, 8 do
				retTable[idx] , retTable[idx+1] = unpack(redis.call("hmget", retTable[i], "id", "nick"))
				idx = idx + 2
			end
		else
			redis.call("select", 0)
		end	

		return retTable
		`)

	curPage, _ := strconv.Atoi(tradeCurPage)
	tradeList, errRedis := redis.Strings(luaScript.Do(c, (curPage-1)*10, curPage*10))

	if errRedis != nil {
		err = "获取充值信息有误" + errRedis.Error()
		return "", err
	}

	retList := new(redisMsg.SCAllTradeInfos)

	listLen, _ := strconv.Atoi(tradeList[0])
	retList.CurPage = curPage
	retList.TotalPage, _ = strconv.Atoi(tradeList[1])
	retList.TotalPage = (retList.TotalPage / 11) + 1

	retList.Trades = make([]redisMsg.SCTradeInfo, listLen)

	for i := 0; i < listLen; i++ {

		retList.Trades[i].UserID, _ = strconv.Atoi(tradeList[i*2+listLen*8+2])
		retList.Trades[i].UserNick = tradeList[i*2+listLen*8+3]

		retList.Trades[i].Cost, _ = strconv.Atoi(tradeList[i*8+3])
		retList.Trades[i].Gold, _ = strconv.Atoi(tradeList[i*8+4])
		retList.Trades[i].ItemID, _ = strconv.Atoi(tradeList[i*8+5])
		retList.Trades[i].Type = tradeList[i*8+6]
		retList.Trades[i].State, _ = strconv.Atoi(tradeList[i*8+7])
		retList.Trades[i].Description = tradeList[i*8+8]
		retList.Trades[i].Date = tradeList[i*8+9]
	}

	//log.Info("tradeInfo: %v", retList)
	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getBGContackInfo() (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	bRet, errRedis := redis.Strings(m.doCommand(c, "hmget", "contact", "qq", "wechat", "kefu"))
	if errRedis != nil {
		err = "获取联系信息有误" + errRedis.Error()
		return "", err
	}

	retContactInfo := new(redisMsg.SCContactInfo)
	retContactInfo.QQ = bRet[0]
	retContactInfo.Wechat = bRet[1]
	retContactInfo.KeFu = bRet[2]

	buf, _ := retContactInfo.MarshalJSON()

	return string(buf), ""
}

func (m *redisDB) bgUpWechat(newWeChat string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	_, errRedis := m.doCommand(c, "hset", "contact", "wechat", newWeChat)
	if errRedis != nil {
		err = "设置微信联系信息有误" + errRedis.Error()
		return "", err
	}

	return
}

func (m *redisDB) bgUpQQ(newQQ string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	_, errRedis := m.doCommand(c, "hset", "contact", "qq", newQQ)
	if errRedis != nil {
		err = "设置微信联系信息有误" + errRedis.Error()
		return "", err
	}

	return
}

func (m *redisDB) bgUpKeFu(newKeFu string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	_, errRedis := m.doCommand(c, "hset", "contact", "kefu", newKeFu)
	if errRedis != nil {
		err = "设置微信联系信息有误" + errRedis.Error()
		return "", err
	}

	return
}

func (m *redisDB) getBGUnsendList(prizeCurPage string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		--先遍历一遍所有记录，删除前面已超时的记录
		local IDs = redis.call("LRANGE", "100000-unSendList", 0, -1)
		
		if IDs then
			for k, v in ipairs(IDs) do
				local info = redis.call("get", v)
				if info then					
					break
				else
					redis.call("LRem", "100000-unSendList", 1, v)
				end			
			end
		end

		local retTable, idx = {}, 1
		--列表长度
		local len = redis.call("llen", "100000-unSendList")
		if tonumber(ARGV[1]) >= len then --所选页超出长度，返回最后10个
			IDs = redis.call("LRANGE", "100000-unSendList", -10, -1)
		else
			IDs = redis.call("LRANGE", "100000-unSendList", ARGV[1], ARGV[2])
		end

		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then					
					local data = cjson.decode(retTable[idx])
					data = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+1] = data["des"]
					idx = idx + 2
				else
					retTable[idx] = nil
					redis.call("LRem", "100000-unSendList", 1, v)
				end			
			end
		end
		
		retTable[idx] = tostring(len)

		return retTable
		`)

	curPage, _ := strconv.Atoi(prizeCurPage)
	recordList, errRedis := redis.Strings(luaScript.Do(c, (curPage-1)*10, curPage*10))
	//log.Info("recordList    %v", recordList)
	if errRedis != nil {
		err = "获取邮箱数据有误：" + errRedis.Error()
		return "", err
	}

	retList := new(redisMsg.SCBGRecordSum)

	listLen := (len(recordList) - 1) / 2

	retList.Records = make([]redisMsg.SCOneRecord, listLen)

	for i := 0; i < listLen; i++ {
		retList.Records[i].Records.UnmarshalJSON([]byte(recordList[i*2]))
		retList.Records[i].Records.DeviceID = recordList[i*2+1] //这里用该变量存储物品名称，即娃娃机的描述
	}

	retList.CurPage = curPage
	retList.TotalPage, _ = strconv.Atoi(recordList[listLen*2])
	retList.TotalPage = (retList.TotalPage / 11) + 1
	if retList.CurPage > retList.TotalPage {
		retList.CurPage = retList.TotalPage
	}
	//log.Info("%v", retList)
	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getBGAsksendList(askSendCurPage string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		--先遍历一遍所有记录，删除前面已超时的记录
		local IDs = redis.call("LRANGE", "100000-askSendList", 0, -1)
		
		if IDs then
			for k, v in ipairs(IDs) do
				local info = redis.call("get", v)
				if info then					
					break
				else
					redis.call("LRem", "100000-askSendList", 1, v)
				end			
			end
		end

		local retTable, idx = {}, 1
		--列表长度
		local len = redis.call("llen", "100000-askSendList")
		if tonumber(ARGV[1]) >= len then --所选页超出长度，返回最后10个
			IDs = redis.call("LRANGE", "100000-askSendList", -10, -1)
		else
			IDs = redis.call("LRANGE", "100000-askSendList", ARGV[1], ARGV[2])
		end

		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then					
					local data = cjson.decode(retTable[idx])
					data = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+1] = data["des"]
					retTable[idx+2], retTable[idx+3], retTable[idx+4] = unpack(redis.call("hmget", v.."-Addr", "name", "tel", "addr"))
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", "100000-askSendList", 1, v)
					redis.call("del", v.."-Addr")
				end			
			end
		end
		
		retTable[idx] = tostring(len)

		return retTable
		`)

	curPage, _ := strconv.Atoi(askSendCurPage)
	recordList, errRedis := redis.Strings(luaScript.Do(c, (curPage-1)*10, curPage*10))
	//log.Info("recordList    %v", recordList)
	if errRedis != nil {
		err = "获取邮箱数据有误：" + errRedis.Error()
		return "", err
	}

	retList := new(redisMsg.SCBGRecordSum)

	listLen := (len(recordList) - 1) / 5

	retList.Records = make([]redisMsg.SCOneRecord, listLen)

	for i := 0; i < listLen; i++ {
		retList.Records[i].Records.UnmarshalJSON([]byte(recordList[i*5]))
		retList.Records[i].Records.DeviceID = recordList[i*5+1] //这里用该变量存储物品名称，即娃娃机的描述
		retList.Records[i].Address.Name = recordList[i*5+2]
		retList.Records[i].Address.Tel = recordList[i*5+3]
		retList.Records[i].Address.Addr = recordList[i*5+4] //这里已经合并过了
	}

	retList.CurPage = curPage
	retList.TotalPage, _ = strconv.Atoi(recordList[listLen*5])
	retList.TotalPage = (retList.TotalPage / 11) + 1
	if retList.CurPage > retList.TotalPage {
		retList.CurPage = retList.TotalPage
	}
	//log.Info("%v", retList)
	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) getBGSendedList(askSendCurPage string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		--先遍历一遍所有记录，删除前面已超时的记录
		local IDs = redis.call("LRANGE", "100000-SendedList", 0, -1)
		
		if IDs then
			for k, v in ipairs(IDs) do
				local info = redis.call("get", v)
				if info then					
					break
				else
					redis.call("LRem", "100000-SendedList", 1, v)
				end			
			end
		end

		local retTable, idx = {}, 1
		--列表长度
		local len = redis.call("llen", "100000-SendedList")
		if tonumber(ARGV[1]) >= len then --所选页超出长度，返回最后10个
			IDs = redis.call("LRANGE", "100000-SendedList", -10, -1)
		else
			IDs = redis.call("LRANGE", "100000-SendedList", ARGV[1], ARGV[2])
		end

		if IDs then
			for k, v in ipairs(IDs) do
				retTable[idx] = redis.call("get", v)
				if retTable[idx] then					
					local data = cjson.decode(retTable[idx])
					data = cjson.decode(redis.call("get", "dev"..data["deviceID"]))
					retTable[idx+1] = data["des"]
					retTable[idx+2], retTable[idx+3], retTable[idx+4] = unpack(redis.call("hmget", v.."-Addr", "name", "tel", "addr"))
					idx = idx + 5
				else
					retTable[idx] = nil
					redis.call("LRem", "100000-SendedList", 1, v)
					redis.call("del", v.."-Addr")
				end			
			end
		end
		
		retTable[idx] = tostring(len)

		return retTable
		`)

	curPage, _ := strconv.Atoi(askSendCurPage)
	recordList, errRedis := redis.Strings(luaScript.Do(c, (curPage-1)*10, curPage*10))
	//log.Info("recordList    %v", recordList)
	if errRedis != nil {
		err = "获取邮箱数据有误：" + errRedis.Error()
		return "", err
	}

	retList := new(redisMsg.SCBGRecordSum)

	listLen := (len(recordList) - 1) / 5

	retList.Records = make([]redisMsg.SCOneRecord, listLen)

	for i := 0; i < listLen; i++ {
		retList.Records[i].Records.UnmarshalJSON([]byte(recordList[i*5]))
		retList.Records[i].Records.DeviceID = recordList[i*5+1] //这里用该变量存储物品名称，即娃娃机的描述
		retList.Records[i].Address.Name = recordList[i*5+2]
		retList.Records[i].Address.Tel = recordList[i*5+3]
		retList.Records[i].Address.Addr = recordList[i*5+4] //这里已经合并过了
	}

	retList.CurPage = curPage
	retList.TotalPage, _ = strconv.Atoi(recordList[listLen*5])
	retList.TotalPage = (retList.TotalPage / 11) + 1
	if retList.CurPage > retList.TotalPage {
		retList.CurPage = retList.TotalPage
	}
	//log.Info("%v", retList)
	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) snedPrize(idxStr, askSendCurPage, userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local wawaID = redis.call("LINDEX", "100000-askSendList", ARGV[1])
				
		if wawaID then
			--从玩家申请发送列表中删除该ID
			redis.call("LRem", ARGV[2].."-askSendList", 1,wawaID)
			redis.call("LPUSH", ARGV[2].."-SendedList", wawaID)
			--从系统申请发送中删除该ID
			redis.call("LRem", "100000-askSendList", 1, wawaID)
			redis.call("LPUSH", "100000-SendedList", wawaID)
		end		
		`)

	curPage, _ := strconv.Atoi(askSendCurPage)
	curIdx, _ := strconv.Atoi(idxStr)
	//log.Info("%d -- %s", (curPage-1)*10+curIdx, userAc)
	_, errRedis := (luaScript.Do(c, (curPage-1)*10+curIdx, userAc))

	if errRedis != nil {
		err = "后台发货出错：" + errRedis.Error()
		return "", err
	}
	return
}
