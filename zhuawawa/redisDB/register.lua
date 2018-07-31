
		local uid = tonumber(redis.call("INCRBY", "SPUserID", 1)) + 100000

        if uid == 100001 then
            --注册管理用户ID agPwd 123456的md5值
            redis.call("HMSET", "agentRoot",  "id", uid, "pwd", ARGV[2], "agPwd", "e10adc3949ba59abbe56e057f20f883e", "nick","总代理", "head", "", "sex", ARGV[5], "openid", "", "lastsession", "","gold", 1000, "agentID", 100000)
            local nowTime=redis.call('TIME') ;
            redis.call("HMSET", "ag100000", "agLv", 0, "ParentID", "ag100000", "rName", "总代", "tel", 13013141314,"cTime", nowTime[1]*1000000+nowTime[2])
            redis.call("HSET", "AllUsers", "100000", "agentRoot")
        end

        redis.call("HMSET", ARGV[1],  "id", uid, "pwd", ARGV[2],"agPwd","", "nick",ARGV[3], "head", ARGV[4], "sex", ARGV[5], "openid", ARGV[6], "lastsession", "","gold", 1000, "agentID", 0)
        return redis.call("HMSET", "AllUsers", tostring(uid), ARGV[1])