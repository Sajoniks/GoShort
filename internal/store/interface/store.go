package urlstore

import (
	"errors"
)

var (
	ErrUrlNotFound = errors.New("url does not exist")
	ErrUrlExists   = errors.New("url already exists")
)

type Closeable interface {
	Close()
}

type Store interface {
	SaveURL(src string, alias string) (string, error)
	GetURL(alias string) (string, error)
}

type CloseableStore interface {
	Store
	Closeable
}
