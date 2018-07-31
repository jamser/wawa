/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"fmt"
	"strings"
	"time"
	redisMsg "zhuawawa/msg"

	"github.com/garyburd/redigo/redis"
	"github.com/liangdas/mqant/log"
)

func (m *redisDB) updateUserGold(uID string, newGold int64) (result bool, err string) {

	c := m.getRedisCon()
	defer c.Close()

	if _, errRedis := redis.Int64(m.doCommand(c, "HSET", uID, "gold", newGold)); errRedis != nil {
		log.Warning("更新用户金币出错： %s", errRedis.Error())
		err = "更新用户金币出错："
		return
	}
	return
}

func (m *redisDB) devicegolive(dInfo []byte) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	deviceInfo := new(redisMsg.SCDeviceInfo)
	deviceInfo.UnmarshalJSON(dInfo)

	luaScript := redis.NewScript(0, `
		local isMember = redis.call("SISMEMBER", "allDevices", ARGV[1])
		if( isMember == 1) then
			--已存储过
			return redis.call("get", ARGV[1])
		else
			--第一次注册
			redis.call("set", ARGV[1], ARGV[2])
			redis.call("sadd", "allDevices", ARGV[1])

			return ARGV[2]
		end
	`)
	/*
		iForceToSet, errRedis := redis.Int64(luaScript.Do(c, deviceInfo.DeviceID, deviceInfo.Group, deviceInfo.SubGroup, deviceInfo.Destription, deviceInfo.Thumbnail, deviceInfo.Cost, deviceInfo.LeftCount, deviceInfo.Force, deviceInfo.Exchange))
		log.Info("设备存储的基数为：%d", iForceToSet)
		if errRedis != nil {
			return -1, "启动娃娃机失败：" + errRedis.Error()
		}

		return iForceToSet, ""
	*/
	result, errRedis := redis.String(luaScript.Do(c, "dev"+deviceInfo.DeviceID, string(dInfo)))
	if errRedis != nil {
		return "", "启动娃娃机失败：" + errRedis.Error()
	}

	return strings.Replace(result, "\\", "", -1), ""
}

func (m *redisDB) getDevicesInfo() (retMsg []byte, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `	

		local devs, retTable = redis.call("Smembers", "allDevices"), {}
		for _,v in ipairs(devs) do
			retTable[#retTable+1] = redis.call("get", v)
		end

		return retTable
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	bRet, errRedis := redis.Strings(luaScript.Do(c))

	if errRedis != nil {
		err = errRedis.Error()
		return []byte(""), err
	}
	retData := new(redisMsg.SCAllDevices)
	retData.AllDevices = make([]redisMsg.SCDeviceInfo, len(bRet))
	for i := 0; i < len(bRet); i++ {
		retData.AllDevices[i].UnmarshalJSON(([]byte)(strings.Replace(bRet[i], "\\", "", -1)))
	}
	log.Info("设备： %v", retData.AllDevices)
	/*
		for i := 0; i < len(bRet); i++ {
			oneDevice := bRet[i].([]interface{})
			retData.AllDevices[i].Group, _ = strconv.Atoi(string(oneDevice[0].([]uint8)))
			retData.AllDevices[i].SubGroup, _ = strconv.Atoi(string(oneDevice[1].([]uint8)))
			retData.AllDevices[i].Destription = string(oneDevice[2].([]uint8))
			retData.AllDevices[i].Thumbnail = string(oneDevice[3].([]uint8))
			retData.AllDevices[i].Cost, _ = strconv.Atoi(string(oneDevice[4].([]uint8)))
			retData.AllDevices[i].DesImg = string(oneDevice[5].([]uint8))
			retData.AllDevices[i].Play, _ = strconv.Atoi(string(oneDevice[6].([]uint8)))
			retData.AllDevices[i].Success, _ = strconv.Atoi(string(oneDevice[7].([]uint8)))
			retData.AllDevices[i].LeftCount, _ = strconv.Atoi(string(oneDevice[8].([]uint8)))
			retData.AllDevices[i].Force, _ = strconv.Atoi(string(oneDevice[9].([]uint8)))
			retData.AllDevices[i].DeviceID = string(oneDevice[10].([]uint8))
			retData.AllDevices[i].Exchange, _ = strconv.Atoi(string(oneDevice[11].([]uint8)))
		}
	*/
	retMsg, _ = retData.MarshalJSON()
	return
}

func (m *redisDB) changeDeviceConfig(newDeviceConfig []byte) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	retData := new(redisMsg.SCDeviceInfo)
	retData.UnmarshalJSON(newDeviceConfig)

	luaScript := redis.NewScript(0, `
		local dInfo = cjson.decode(redis.call("get", ARGV[1]))
		dInfo["des"] = ARGV[2]
		dInfo["cost"] = tonumber(ARGV[3])
		dInfo["left"] = tonumber(ARGV[4])
		dInfo["force"] = tonumber(ARGV[5])
		dInfo["group"] = tonumber(ARGV[6])
		dInfo["thub"] = ARGV[7]
		dInfo["desImgUrl"] = ARGV[8]
		dInfo["exchange"] = tonumber(ARGV[9])
		local newData = cjson.encode(dInfo)
		redis.call("set", ARGV[1], newData)
		return newData
	`)
	bRet, errRedis := redis.String(luaScript.Do(c, "dev"+retData.DeviceID, retData.Destription, retData.Cost, retData.LeftCount, retData.Force,
		retData.Group, retData.Thumbnail, retData.DesImg, retData.Exchange))

	if errRedis != nil {
		err = errRedis.Error()
		return false, err
	}
	/*
		_, errRedis := m.doCommand(c, "hmset", "dev"+retData.DeviceID, "des", retData.Destription, "cost", retData.Cost, "left", retData.LeftCount,
			"force", retData.Force, "group", retData.Group, "thub", retData.Thumbnail, "desImg", retData.DesImg, "exchange", retData.Exchange)

		if errRedis != nil {
			err = errRedis.Error()
			return false, err
		}
	*/
	m.RpcInvoke("MatchRoom", "changeConfig", retData.DeviceID, strings.Replace(bRet, "\\", "", -1))
	return true, ""
}

func (m *redisDB) recordSuccess(deviceID string, rInfo []byte) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	successInfo := new(redisMsg.SCSuccessRecord)
	successInfo.UnmarshalJSON(rInfo)
	timeNow := time.Now()
	successInfo.Date = timeNow.Format("2006/01/02 15:04:05")

	luaScript := redis.NewScript(0, `
		local mid = tonumber(redis.call("INCRBY", "SPSuccessID", 1))
		local wawaID = "wawagot"..mid
		redis.call("set", wawaID, ARGV[2])
		--redis.call("expire", wawaID, 1296000) -- 15天过期
		redis.call("expire", wawaID, 36000) -- 测试6分钟过期

		local count = redis.call("LPUSH", ARGV[1].."-List", wawaID)

		if (count > 10) then
			redis.call("LTRIM", ARGV[1].."-List", 0, 9)
		end

		redis.call("LPUSH", ARGV[3].."-unSendList", wawaID)
		redis.call("LPUSH", "100000-unSendList", wawaID)
		
		redis.call("zincrby", "successRank", 1, ARGV[3])
		--当只有一个成员时，更新超时时间
		if (redis.call("zcard", "successRank") == 1) then
			redis.call("expire", "successRank", ARGV[4])
		end

		local devInfo = cjson.decode(redis.call("get", "dev"..ARGV[1]))
		devInfo["play"] = devInfo["play"] + 1
		devInfo["success"] = devInfo["success"] + 1
		devInfo["left"] = devInfo["left"] - 1
		
		redis.call("set", "dev"..ARGV[1], cjson.encode(devInfo))

		return 1
	`)
	mailInfo := new(redisMsg.SCMainInfo)
	mailInfo.SystemMsg = false
	mailInfo.ID = ""
	mailInfo.Date = successInfo.Date
	mailInfo.Title = "中奖"
	mailInfo.MailDes = fmt.Sprintf("恭喜你在%s抓中了娃娃!", deviceID)
	mailInfo.Read = false
	mailInfo.Reward = 0
	buf, _ := mailInfo.MarshalJSON()
	if _, mailErr := m.pushUserMail(successInfo.UserID, buf); mailErr != "" {
		log.Warning("成功记录到邮箱内失败：%s", mailErr)
	}

	nowTime, timeSub := time.Now(), 0
	weekDay := nowTime.Weekday()
	//星期天，只算24点到现在的时间差,其他时间计算到下周一的天数乘以86400秒
	if weekDay != time.Sunday {
		timeSub = int((time.Saturday - weekDay + 1) * 86400)
	}
	timeSub += (23-nowTime.Hour())*3600 + (59-nowTime.Minute())*60 + (60 - nowTime.Second())
	rInfo, _ = successInfo.MarshalJSON()
	_, errRedis := luaScript.Do(c, deviceID, rInfo, successInfo.UserID, timeSub)

	if errRedis != nil {
		err = "成功记录登记有误：" + errRedis.Error()
		log.Warning("%s", err)
		return false, err
	}

	return true, err
}

func (m *redisDB) recordPlay(deviceID string) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local devInfo = cjson.decode(redis.call("get", ARGV[1]))
		devInfo["play"] = devInfo["play"] + 1

		redis.call("set", ARGV[1], cjson.encode(devInfo))
		
	`)

	_, errRedis := luaScript.Do(c, "dev"+deviceID)

	if errRedis != nil {
		err = "记录设备游戏有误：" + errRedis.Error()
		log.Warning("%s", err)
		return false, err
	}

	return true, ""
}

func (m *redisDB) getSuccessList(deviceID string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()
	luaScript := redis.NewScript(0, `
		local IDs = redis.call("LRANGE", ARGV[1], 0, 9)
		local retTable, idx = {}, 1
		if not IDs then
			return retTable
		end
		for k, v in ipairs(IDs) do
			retTable[idx] = redis.call("get", v)
			if retTable[idx] then
				idx = idx + 1
			else
				retTable[idx] = nil
				redis.call("LRem", ARGV[1], 1, v)
			end			
		end

		return retTable
		`)
	//log.Info("%s", deviceID)
	retInfo, errRedis := redis.Strings(luaScript.Do(c, deviceID+"-List"))
	//retInfo, errRedis := redis.Strings(m.doCommand(c, "LRANGE", deviceID+"-List", 0, 9))

	if errRedis != nil {
		err = "获取成功记录有误：" + errRedis.Error()
		log.Info("%s", err)
		return "", err
	}
	//log.Info("%v", retInfo)
	retList := new(redisMsg.SCSuccessList)
	recordLen := len(retInfo)

	retList.Records = make([]redisMsg.SCSuccessRecord, recordLen)
	for i := 0; i < recordLen; i++ {
		retList.Records[i].UnmarshalJSON([]byte(retInfo[i]))
	}

	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}
