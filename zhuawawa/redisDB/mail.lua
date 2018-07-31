local selfList = redis.call("LRANGE", ARGV[1] .. "-Mail", 0, -1)
		
local systemList = redis.call("LRANGE", "sysMail", 0, -1)

if table.getn(selfList) > 0 then
	if table.getn(systemList) then
		return selfList, systemList
	end

	return selfList
end

if table.getn(systemList)  then
	return systemList
end