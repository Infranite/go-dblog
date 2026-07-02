package redis

import "github.com/Infranite/go-dblog/redis/decode/events/types"

const (
	// Driver is the Redis-family backend driver name.
	Driver = types.Driver

	// CommandHSet is the Redis HSET command.
	CommandHSet = types.CommandHSet
	// CommandHDel is the Redis HDEL command.
	CommandHDel = types.CommandHDel
	// CommandSAdd is the Redis SADD command.
	CommandSAdd = types.CommandSAdd
	// CommandSRem is the Redis SREM command.
	CommandSRem = types.CommandSRem
	// CommandLPush is the Redis LPUSH command.
	CommandLPush = types.CommandLPush
	// CommandLPop is the Redis LPOP command.
	CommandLPop = types.CommandLPop
	// CommandRPush is the Redis RPUSH command.
	CommandRPush = types.CommandRPush
	// CommandRPop is the Redis RPOP command.
	CommandRPop = types.CommandRPop
	// CommandIncr is the Redis INCR command.
	CommandIncr = types.CommandIncr
	// CommandDecr is the Redis DECR command.
	CommandDecr = types.CommandDecr
	// CommandIncrBy is the Redis INCRBY command.
	CommandIncrBy = types.CommandIncrBy
	// CommandDecrBy is the Redis DECRBY command.
	CommandDecrBy = types.CommandDecrBy
	// CommandHIncrBy is the Redis HINCRBY command.
	CommandHIncrBy = types.CommandHIncrBy
	// CommandHIncrByFloat is the Redis HINCRBYFLOAT command.
	CommandHIncrByFloat = types.CommandHIncrByFloat
	// CommandZIncrBy is the Redis ZINCRBY command.
	CommandZIncrBy = types.CommandZIncrBy
)
