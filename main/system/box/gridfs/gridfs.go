package gf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bim-z/mathrock/main/system/red"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Gridfs struct {
	db     *mongo.Database
	bucket *gridfs.Bucket
}

func (g *Gridfs) Put(key string, data io.Reader) (err error) {
	if g.Exist(key) {
		return
	}

	if _, err = g.bucket.UploadFromStream(key, data); err != nil {
		return
	}

	return
}

func (g *Gridfs) Get(key string) (data io.Reader, err error) {
	if !g.Exist(key) {
		return nil, fmt.Errorf("object not found")
	}

	buff := bytes.NewBuffer([]byte{})
	if _, err = g.bucket.DownloadToStreamByName(key, buff); err != nil {
		return nil, err
	}

	return buff, nil
}

func (g *Gridfs) Delete(key string) (err error) {
	if !g.Exist(key) {
		return
	}

	return g.bucket.Delete(key)
}

func (g Gridfs) Exist(key string) bool {
	status := red.Red.Get(context.Background(), key)
	if err := status.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return false
		}

		return false
	}

	return true
}

func Setup() (box *Gridfs, err error) {
	// mongo client
	client, _ := mongo.Connect(context.Background(), options.Client().ApplyURI(""))

	// instance
	box = &Gridfs{
		db: client.Database("default"),
	}

	// bucket
	box.bucket, err = gridfs.NewBucket(box.db)
	return
}
