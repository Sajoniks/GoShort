package urlstore

import (
	"errors"
)

var (
	ErrUrlNotFound = errors.New("url does not exist")
	ErrUrlExists   = errors.New("url already exists")
	ErrAliasEmpty  = errors.New("alias is empty")
	ErrUrlEmpty    = errors.New("url is empty")
)

type Closeable interface {
	Close()
}

type Store interface {
	SaveURL(src, alias string) (string, error)
	GetURL(alias string) (string, error)
}

type CloseableStore interface {
	Store
	Closeable
}
