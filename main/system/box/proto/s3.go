package proto

import "io"

type simple struct{}

func (simple) Put(key string, data io.Reader) (err error) {}
