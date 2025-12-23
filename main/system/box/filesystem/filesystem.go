package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/bim-z/mathrock/main/system/red"
	"github.com/redis/go-redis/v9"
)

type Filesystem struct {
	home string
}

func (f *Filesystem) Put(key string, data io.Reader) (err error) {
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

func (f *Filesystem) Get(key string) (data io.Reader, err error) {
	if !f.Exist(key) {
		return nil, fmt.Errorf("object not found")
	}

	file, err := os.Open(path.Join(f.home, key))
	if err != nil {
		return
	}
	defer file.Close()

	return file, nil
}

func (f *Filesystem) Delete(key string) (err error) {
	if !f.Exist(key) {
		return fmt.Errorf("object not found")
	}

	return os.Remove(path.Join(f.home, key))
}

func (f *Filesystem) Exist(key string) (ok bool) {
	status := red.Red.Get(context.Background(), key)
	if status.Err() != nil {
		if errors.Is(status.Err(), redis.Nil) {
			return
		}
	}

	return true
}

func Setup() (f *Filesystem) {
	f = new(Filesystem)
	f.home = xdg.DataHome
	return
}
