package storage

import (
	"errors"
	"monk-db/pkg/constants"
	"strings"
)

type Store struct {
	Data map[string]string
}

func InitStore() *Store {
	return &Store{
		Data: make(map[string]string),
	}
}

func (s *Store) Put(key, val string) (bool, error) {
	if s == nil || s.Data == nil {
		return false, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)
	s.Data[key] = val

	return true, nil
}

func (s *Store) Get(key string) (string, error) {
	key = strings.ToLower(key)

	val, ok := s.Data[key]
	if !ok {
		return "NOT_FOUND", errors.New(constants.ERRNOTFOUND)
	}

	return val, nil
}
