package filesystem

import (
	"context"
	"errors"
	"io"
	"os"
	"path"

	"github.com/bim-z/mathrock/main/system/red"
	"github.com/redis/go-redis/v9"
)

type Filesystem struct {
	home string
}

func (f Filesystem) Put(key string, data io.Reader) (err error) {
	if f.Exist(key) {
		return
	}

	ctx := context.Background()

	descriptor, err := os.Create(path.Join(f.home, key))
	if err != nil {
		return
	}

	if _, err = io.Copy(descriptor, data); err != nil {
		return
	}

	// save to redis for faster search operation
	status := red.Red.Set(ctx, key, "", 0)
	return status.Err()
}

func (f Filesystem) Exist(key string) (ok bool) {
	status := red.Red.Get(context.Background(), key)
	if status.Err() != nil {
		if errors.Is(status.Err(), redis.Nil) {
			return
		}
	}

	return true
}
