package types

const (
	// Driver is the Redis-family backend driver name.
	Driver = "redis"

	// CommandHSet is the Redis HSET command.
	CommandHSet = "hset"
	// CommandHDel is the Redis HDEL command.
	CommandHDel = "hdel"
	// CommandSAdd is the Redis SADD command.
	CommandSAdd = "sadd"
	// CommandSRem is the Redis SREM command.
	CommandSRem = "srem"
	// CommandLPush is the Redis LPUSH command.
	CommandLPush = "lpush"
	// CommandLPop is the Redis LPOP command.
	CommandLPop = "lpop"
	// CommandRPush is the Redis RPUSH command.
	CommandRPush = "rpush"
	// CommandRPop is the Redis RPOP command.
	CommandRPop = "rpop"
	// CommandIncr is the Redis INCR command.
	CommandIncr = "incr"
	// CommandDecr is the Redis DECR command.
	CommandDecr = "decr"
	// CommandIncrBy is the Redis INCRBY command.
	CommandIncrBy = "incrby"
	// CommandDecrBy is the Redis DECRBY command.
	CommandDecrBy = "decrby"
)
