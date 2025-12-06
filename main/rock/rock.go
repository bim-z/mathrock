package rock

import "github.com/redis/go-redis/v9"

var Rock = redis.NewClient(&redis.Options{})
