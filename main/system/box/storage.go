package box

import "io"

type Storage interface {
	Put(key string, data io.Reader) error
	Get(key string) error
	Delete(key string) error
}
