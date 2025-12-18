package proto

import "io"

type box interface {
	Put(key string, file io.Closer) (err error)
	Get()
	Delete()
	Exist()
}
