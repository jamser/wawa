
local count = redis.call("LPUSH", ARGV[1], ARGV[2])

if (count > 10) then
	redis.call("LTRIM", ARGV[1], 0, 9)
end

return 1