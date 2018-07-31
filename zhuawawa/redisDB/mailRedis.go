/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"fmt"
	redisMsg "zhuawawa/msg"

	"github.com/garyburd/redigo/redis"
	"github.com/liangdas/mqant/log"
)

func (m *redisDB) getMailLists(userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local selfList = redis.call("LRANGE", ARGV[1] .. "-Mail", 0, -1)
		local retTable, idx = {}, 1
		for _, v in ipairs(selfList) do
			retTable[idx] = redis.call("get", v)
			idx = idx + 1
		end

		local systemList = redis.call("LRANGE", "system-Mail", 0, -1)
		for _, v in ipairs(systemList) do
			retTable[idx] = redis.call("get", v)
			local mail = cjson.decode(retTable[idx])
			local retInt = redis.call("sismember", "system-Mail-state" .. mail["mailID"], ARGV[1])
			if retInt == 1 then
				mail["read"] = 	true	
			else
				mail["read"] = 	false	
			end
			retTable[idx] = cjson.encode(mail)
			idx = idx + 1
		end
		return retTable
	`)

	mailList, errRedis := redis.Strings(luaScript.Do(c, userAc))
	fmt.Println(mailList)

	if errRedis != nil {
		err = "获取邮箱数据有误：" + errRedis.Error()
		return "", err
	}

	retList := new(redisMsg.SCAllMainInfo)

	listLen := len(mailList)

	retList.Mails = make([]redisMsg.SCMainInfo, listLen)

	for i := 0; i < listLen; i++ {
		retList.Mails[i].UnmarshalJSON([]byte(mailList[i]))
	}

	buf, _ := retList.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) pushUserMail(userAc string, mail []byte) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	mailMsg := new(redisMsg.SCMainInfo)
	mailMsg.UnmarshalJSON(mail)

	luaScript := redis.NewScript(0, `
		local mid = tonumber(redis.call("INCRBY", "SPMailID", 1))
		local mailID = "mail-" .. mid
		local mailInfo = cjson.decode(ARGV[2])
		mailInfo["mailID"] = mailID

		redis.call("set", mailID, cjson.encode(mailInfo))

		local mailList = ARGV[1] .. "-Mail"
		local listLen = redis.call("LPUSH", mailList, mailID)
		if listLen > 50 then
			local mail = redis.call("BLPOP", mailList, 0, 1)
			redis.call("del", mail)
		end
	`)

	_, errRedis := luaScript.Do(c, userAc, string(mail))

	if errRedis != nil {
		err = "推送用户邮箱有误：" + errRedis.Error()
		return false, err
	}

	return true, ""
}

func (m *redisDB) readUserMail(userAc string, msgID string) (result, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local msg = redis.call("get", ARGV[2])
		if not msg then
			return
		end

		local info = cjson.decode(msg)
		info["read"] = true

		redis.call("set", ARGV[2], cjson.encode(info))
	`)
	log.Info("msgID is ： %s", msgID)
	_, errRedis := (luaScript.Do(c, userAc, msgID))

	if errRedis != nil {
		err = "更新邮件状态有误：" + errRedis.Error()
		return "failed", err
	}

	return "ok", ""
}

func (m *redisDB) pushSystemMail(mail []byte) (result bool, err string) {
	c := m.getRedisCon()
	defer c.Close()

	mailMsg := new(redisMsg.SCMainInfo)
	mailMsg.UnmarshalJSON(mail)

	luaScript := redis.NewScript(0, `
		local mailList = "system-Mail"
		local mid = tonumber(redis.call("INCRBY", "SPMailID", 1))
		local mailID = "mail-" .. mid

		local mailInfo = cjson.decode(ARGV[1])
		mailInfo["mailID"] = mailID
		redis.call("set", mailID, ARGV[2])

		local listLen = redis.call("LPUSH", mailList, mailID)

		if listLen > 50 then
			local mail = redis.call("BLPOP", mailList, 0.1)
			redis.call("del", mail)
			redis.call("del", "system-Mail-state" .. mail)
		end
		
	`)

	_, errRedis := (luaScript.Do(c, mail))

	if errRedis != nil {
		err = "推送系统邮箱有误：" + errRedis.Error()
		return false, err
	}

	return true, ""
}

func (m *redisDB) readSystemMail(userAc string, msgID string) (result, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local retInt = redis.call("exists", ARGV[1])
		if retInt == 1 then
			redis.call("sadd", "system-Mail-state" .. ARGV[1], ARGV[2])
		end
	`)

	_, errRedis := (luaScript.Do(c, msgID, userAc))

	if errRedis != nil {
		err = "更新系统邮件状态有误：" + errRedis.Error()
		return "failed", err
	}

	return "ok", ""
}
