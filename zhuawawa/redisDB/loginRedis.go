/*Package redisDB 数据库模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package redisDB

import (
	"time"
	redisMsg "zhuawawa/msg"

	"github.com/garyburd/redigo/redis"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/log"
)

func (m *redisDB) register(session gate.Session, registerMsg []byte) (result map[string]interface{}, err string) {

	c := m.getRedisCon()
	defer c.Close()

	getQAScript := redis.NewScript(0, `
		local uid = tonumber(redis.call("INCRBY", "SPUserID", 1)) + 100000
		redis.call("HMSET", ARGV[1],  "id", uid, "pwd", ARGV[2], "nick",ARGV[3], "head", ARGV[4], "sex", ARGV[5], "openid", ARGV[6], "date", ARGV[7],
			 	"lastsession", "","gold", 1000, "reward", 0, "bindedInvite", "", "keepSign", 0, "startSign", "0",
				"weekCard", "", "weekGet", "", "monthCard", "", "monthGet", ""
			)
		return redis.call("HSET", "AllUsers", tostring(uid), ARGV[1])
	`)
	createTime := time.Now().Format("2006-01-02 15:04:05")
	registerInfo := new(redisMsg.CSRegister)
	registerInfo.UnmarshalJSON(registerMsg)
	// In a function, use the script Do method to evaluate the script. The Do
	// method optimistically uses the EVALSHA command. If the script is not
	// loaded, then the Do method falls back to the EVAL command.
	if _, errRedis := getQAScript.Do(c, registerInfo.Username, registerInfo.Password, registerInfo.NickName,
		registerInfo.HeadURL, registerInfo.Gender, registerInfo.OpenID, createTime); errRedis != nil {
		err = "注册出错:" + errRedis.Error()
		return
	}

	err = session.Bind(registerInfo.Username)
	if err != "" {
		return
	}

	//推送到网关
	session.SetPush("lastplace", "Lobby")

	return
}

func (m *redisDB) checklogin( /*session gate.Session,*/ userName string, password string) (result map[string]interface{}, err string) {
	c := m.getRedisCon()
	defer c.Close()

	c.Send("MULTI")
	c.Send("HGET", userName, "pwd")
	c.Send("GET", "PubNotice")
	r, errRedis := redis.Values(m.doCommand(c, "EXEC"))
	if errRedis != nil {
		err = "获取用户信息有误！" + errRedis.Error()
		return
	}

	if r[0] != nil {
		if string(r[0].([]byte)) == password {
			//登陆成功
			result = make(map[string]interface{})
			result["success"] = true
			result["pub"], _ = redis.String(r[1], nil)
		} else {
			log.Info("密码有误： %s  :   %s", password, string(r[0].([]byte)))
			//密码错误
			err = "密码错误！"
			return
		}
	} else {
		//注册用户
		result = make(map[string]interface{})
		result["register"] = true
		return
	}

	//log.Info("user %s checklogin end ... ", userName)
	/*
		err = session.Bind(userName)
		if err != "" {
			return
		}

		if !userInfo.Online {
			//未登陆则推送到网关
			session.Set("online", "true")
			session.Push()
		}
	*/
	return
}

//saveSession 持久化session信息，下次登陸可用, (ret string, err string) RPC调用必须返回两个结果， 具体是否传递回调用方看调用的方式是否不带NR
func (m *redisDB) saveSession(userName string, lastsession []byte) (ret string, err string) {

	c := m.getRedisCon()
	defer c.Close()

	_, errRedis := m.doCommand(c, "HSET", userName, "lastsession", lastsession)

	if errRedis != nil {
		log.Warning("获取用户信息有误！ %s", errRedis.Error())
		return
	}

	return
}

func (m *redisDB) loadSession(userName string) (lastsession []byte, err string) {
	c := m.getRedisCon()
	defer c.Close()

	r, errRedis := redis.Bytes(m.doCommand(c, "HGET", userName, "lastsession"))
	if errRedis != nil {
		log.Warning("获取用户lastsession信息有误！ %s", errRedis.Error())
		err = "获取用户上次所在位置出错！"
		return
	}
	lastsession = r
	return
}
