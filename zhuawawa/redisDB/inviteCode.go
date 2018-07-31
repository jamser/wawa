/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"math"
	"strconv"
	"time"
	redisMsg "zhuawawa/msg"

	"github.com/garyburd/redigo/redis"
)

var (
	sourceString = "2YU9P6ASDFG8QWERTHJ7KLZX4CV5B3NM" //自定义32进制
	padString    = "0"                                //补足字符（不能在source_string出现过）
)

func generateInviteCode(userID int) (inviteCode string) {
	mod := 0
	//log.Info("uid : %d", userID)
	for userID > 0 {
		mod = userID % 32
		userID = (userID - mod) / 32
		inviteCode = string(sourceString[mod]) + inviteCode

	}

	for i := 6 - len(inviteCode); i > 0; i-- {
		inviteCode = padString + inviteCode
	}
	return
}

func getUserIDFormInviteCode(inviteCode string) (userID int) {
	idx := 0
	for i := 0; i < len(inviteCode); i++ {
		if inviteCode[i] != '0' {
			idx = i
			break
		}
	}

	inviteCode = string(([]byte(inviteCode))[idx:len(inviteCode)])

	for i := 0; i < len(inviteCode); i++ {
		for j := 0; j < 32; j++ {
			if inviteCode[i] == sourceString[j] {
				userID += j * int(math.Pow(32, float64(len(inviteCode)-i-1)))
			}
		}
	}

	return
}

func (m *redisDB) getInviteInfo(userAc string) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	luaScript := redis.NewScript(0, `
		local uID, leftReward, bindedInvite = unpack(redis.call("hmget", ARGV[1], "id", "reward", "bindedInvite"))
		if bindedInvite ~= "" then
			bindedInvite = redis.call("hget", bindedInvite, "id")
		end
		local inviteUsers = redis.call("Smembers", ARGV[1].."Invites")
		if not inviteUsers then
		 	return {uID, leftReward, bindedInvite}
		else
			return {uID, leftReward, bindedInvite, unpack(inviteUsers)}
		end		
	`)

	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	ret, errRedis := redis.Values(luaScript.Do(c, userAc))
	//log.Info("%v", ret)
	if errRedis != nil {
		return "", errRedis.Error()
	}

	inviteInfo := new(redisMsg.SCInviteInfo)
	uid, _ := strconv.Atoi(string(ret[0].([]uint8)))
	inviteInfo.Code = generateInviteCode(uid)
	inviteInfo.LeftReward, _ = redis.Int(ret[1], nil)
	uid, _ = strconv.Atoi(string(ret[2].([]uint8)))
	inviteInfo.BindedUser = generateInviteCode(uid)
	//log.Info("ssss %v", inviteInfo)
	userCount := len(ret) - 3
	inviteInfo.Binders = make([]redisMsg.SCInviteUserInfo, userCount)
	for i := 0; i < userCount; i++ {
		inviteInfo.Binders[i].UnmarshalJSON([]byte(ret[i+3].([]uint8)))
	}

	buf, _ := inviteInfo.MarshalJSON()
	return string(buf), ""
}

func (m *redisDB) BindUser(userAc string, msg []byte) (result string, err string) {
	c := m.getRedisCon()
	defer c.Close()

	bindInfo := new(redisMsg.CSBindCode)
	bindInfo.UnmarshalJSON(msg)
	userID := getUserIDFormInviteCode(bindInfo.Code)
	//log.Info("uid : %d", userID)
	luaScript := redis.NewScript(0, `
		local bindUser, nick, head = unpack(redis.call("hmget", ARGV[1], "bindedInvite", "nick", "head"))
		if bindUser ~="" then
			return {0} --已绑定其他玩家，不可更改绑定玩家
		end
		--获得对应ID的用户账号
		local bindUserAc = redis.call("hget", "AllUsers", ARGV[2])

		--不能绑定自己的邀请码
		if bindUserAc == ARGV[1] then
			return {1}
		end

		local binderInviteName =  bindUserAc.."Invites"

		-- 不可重复绑定
		if (redis.call("SISMember", binderInviteName, ARGV[1]) ~= 0) then
			return {2}
		end

		local userGain, binderGain, goldGainMaxUser = unpack(redis.call("hmget", "bindInfos", "userGain", "binderGain", "goldGainMaxUser"))
		--绑定成功，设置当前玩家的绑定玩家,以及奖励金币
		redis.call("hset", ARGV[1], "bindedInvite", bindUserAc)
		redis.call("hincrby", ARGV[1], "gold", userGain)
		--给被绑定玩家金币奖励(大于goldGainMaxUser数目的玩家不会再奖励被绑定玩家金币)
		if redis.call("SCARD", binderInviteName) < tonumber(goldGainMaxUser) then
			redis.call("hincrby", bindUserAc, "gold", binderGain)
		end
		--添加到被绑定玩家的邀请列表里面
		local bindInfo = {}
		bindInfo["ac"] = ARGV[1]
		bindInfo["money"] = 0
		bindInfo["nick"] = nick
		bindInfo["head"] = head
		redis.call("sadd", bindUserAc .."Invites", cjson.encode(bindInfo))

		--给被绑定玩家发送邮件
		local mailInfo = {}

		local mid = redis.call("INCRBY", "SPMailID", 1)
		local mailID = "mail-" .. mid

		mailInfo["mailID"] = mailID
		mailInfo["sys"] = false
		mailInfo["title"] = "邀请奖励"
		mailInfo["mail"] = "邀请玩家注册，奖励".. binderGain .. "娃娃币"
		mailInfo["read"] = false
		mailInfo["reward"] = 0
		mailInfo["date"] = ARGV[3]

		redis.call("set", mailID, cjson.encode(mailInfo))

		local mailList = bindUserAc .. "-Mail"
		local listLen = redis.call("LPUSH", mailList, mailID)
		if listLen > 50 then
			local mail = redis.call("BLPOP", mailList, 0, 1)
			redis.call("del", mail.. "-Infos")
		end

		--给当前玩家发送邮件和奖励
		mid = redis.call("INCRBY", "SPMailID", 1)
		mailID = "mail-" .. mid

		mailInfo["mailID"] = mailID
		mailInfo["title"] = "绑定奖励"
		mailInfo["mail"] = "绑定邀请码，奖励".. userGain .. "娃娃币"
		mailInfo["read"] = false
		mailInfo["reward"] = 0
		mailInfo["date"] = ARGV[3]

		redis.call("set", mailID, cjson.encode(mailInfo))
		
		mailList = ARGV[1] .. "-Mail"
		local listLen = redis.call("LPUSH", mailList, mailID)
		if listLen > 50 then
			local mail = redis.call("BLPOP", mailList, 0, 1)
			redis.call("del", mail.. "-Infos")
		end
		
		return {3, bindUserAc}	
	`)

	ret, errRedis := redis.Values(luaScript.Do(c, userAc, userID, time.Now().Format("2006/01/02 15:04:05")))

	if errRedis != nil {
		return "", "绑定失败：" + errRedis.Error()
	}

	retInfo := new(redisMsg.SCBindRet)

	switch ret[0].(int64) {
	case 0:
		retInfo.State = false
		retInfo.Description = "已绑定其他玩家，不能更改绑定玩家"
		//log.Warning("玩家%s绑定失败：%s", userAc, retInfo.Description)
	case 1:
		retInfo.State = false
		retInfo.Description = "不能绑定自己的邀请码"
	case 2:
		retInfo.State = false
		retInfo.Description = "已绑定该玩家，请勿重复绑定"
		//log.Warning("玩家%s绑定失败：%s", userAc, retInfo.Description)
	case 3:
		retInfo.State = true
		retInfo.Description = "绑定成功！"
	}

	buf, _ := retInfo.MarshalJSON()
	if retInfo.State {
		m.RpcInvokeNR("Gate", "inviteNotice", string(ret[1].([]uint8)))
	}

	return string(buf), err
}
