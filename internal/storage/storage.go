package storage

import "io"

type Storage interface {
	Put(key string, r io.Reader) error
	Get(key string) (io.ReadCloser, error)
	Delete(key string) error
	URL(key string) string
}
