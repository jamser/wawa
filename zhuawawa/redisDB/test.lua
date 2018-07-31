
    local ret1,ret2,ret3
    local rst ,idx = {} , 1
    if(tonumber(ARGV[1]) > 0) then
        ret1= redis.call("SRANDMEMBER", KEYS[1] , ARGV[1])

        for i=1, #ret1 do
            rst[idx] = redis.call("hgetall", "title" .. tostring(ret1[i]))
            idx=idx+1
        end
    end
    if (tonumber(ARGV[2]) > 0) then
        ret2 = redis.call("SRANDMEMBER", KEYS[2] , ARGV[2])

        for i=1, #ret2 do
            rst[idx] = redis.call("hgetall", "title"..tostring(ret2[i]))
            idx=idx+1
        end
    end

    if (tonumber(ARGV[3]) > 0) then
        ret3 = redis.call("SRANDMEMBER", KEYS[3] , ARGV[3])

        for i=1, #ret3 do
            rst[idx] = redis.call("hgetall", "title"..tostring(ret3[i]))
            idx=idx+1
        end
    end

    return rst