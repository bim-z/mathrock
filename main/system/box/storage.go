package box

import "io"

type Storage interface {
	Put(key string, data io.Reader) error
	Get(key string) (data io.Reader, err error)
	Delete(key string) error
	Exist(key string) bool
}
