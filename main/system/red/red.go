package red

import (
	"os"

	"github.com/redis/go-redis/v9"
)

var Red = redis.NewClient(&redis.Options{
	Addr: os.Getenv("REDIS_ADDR"),
})
