package storage

import (
	"errors"
	"monk-db/pkg/constants"
	"monk-db/pkg/sstable"
	"strings"
)

type Store struct {
	data   map[string]string
	offset int
	size   int
}

func InitStore(size int) *Store {
	return &Store{
		data:   make(map[string]string, size),
		offset: 0,
		size:   size,
	}
}

func (s *Store) Put(key, val string) (bool, error) {
	if s == nil || s.data == nil {
		return false, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)
	s.data[key] = val

	if len(s.data) == s.size {
		// fetch a new instance to store 2000 (or the given size) records in log files
		sstable := sstable.NewSSTable(s.offset + 1)
		err := sstable.Flush(s.data)
		if err != nil {
			return false, err
		}

		s.data = make(map[string]string, s.size)
		s.offset = s.offset + 1
	}

	return true, nil
}

func (s *Store) Get(key string) (string, error) {
	if s == nil || s.data == nil {
		return constants.EMPTYSTRING, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)

	val, ok := s.data[key]
	if !ok {
		return "NOT_FOUND", errors.New(constants.ERRNOTFOUND)
	}

	return val, nil
}
