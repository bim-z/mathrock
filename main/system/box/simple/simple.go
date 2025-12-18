// this is for amazon simple storage implementation
package simple

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bim-z/mathrock/main/system/red"
	"github.com/redis/go-redis/v9"
)

type Simple struct {
	storage *s3.Client
}

func (s Simple) Put(key string, data io.Reader) (err error) {
	ctx := context.Background()

	if s.Exist(key) {
		return
	}

	if _, err = s.storage.PutObject(context.Background(), &s3.PutObjectInput{
		Key:  &key,
		Body: data,
	}); err != nil {
		return
	}

	// save to redis for faster search operation
	status := red.Red.Set(ctx, key, "", 0)
	return status.Err()
}

func (s Simple) Get(key string) {
}

func (s Simple) Delete(key string) {}

func (s Simple) Exist(key string) (ok bool) {
	status := red.Red.Get(context.Background(), key)
	if status.Err() != nil {
		if errors.Is(status.Err(), redis.Nil) {
			return
		}
	}

	return true
}
