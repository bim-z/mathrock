// this is for amazon simple storage implementation
package simple

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func (s Simple) Get(key string) (data io.Reader, err error) {
	out, err := s.storage.GetObject(context.Background(), &s3.GetObjectInput{
		Key: &key,
	})

	if err != nil {
		return
	}

	data = out.Body
	return
}

func (s Simple) Delete(key string) (err error) {
	_, err = s.storage.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Key: &key,
	})

	return
}

func (s Simple) Exist(key string) (ok bool) {
	status := red.Red.Get(context.Background(), key)
	if status.Err() != nil {
		if errors.Is(status.Err(), redis.Nil) {
			return
		}
	}

	return true
}

func Setup() (s *Simple, err error) {
	endpoint := os.Getenv("BUCKET_ENDPOINT")
	accesskey := os.Getenv("BUCKET_ACCESS_KEY")
	secretkey := os.Getenv("BUCKET_SECRET_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithBaseEndpoint(endpoint),
		config.WithCredentialsProvider(
			credentials.
				NewStaticCredentialsProvider(
					accesskey,
					secretkey,
					"",
				),
		),
		config.WithRegion("us-east-1"),
	)

	if err != nil {
		return
	}

	s = new(Simple)
	s.storage = s3.NewFromConfig(cfg)

	_, err = s.storage.CreateBucket(
		context.TODO(),
		&s3.CreateBucketInput{
			Bucket: aws.String("default"),
		},
	)

	return s, nil
}
