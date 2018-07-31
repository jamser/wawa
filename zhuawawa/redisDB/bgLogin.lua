local ac = redis.call("hget", "AllUsers", ARGV[1])
if not ac then
	return 0
end

local pwd, agID = unpack(redis.call("hmget", ac, "agPwd", "agentID"))
		
if (tonumber(agID) ~= 0 and pwd == ARGV[2]) 
then
	return 1
end
		
return 0