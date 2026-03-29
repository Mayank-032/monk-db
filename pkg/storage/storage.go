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

func InitStore(size, offset int) *Store {
	return &Store{
		data:   make(map[string]string, size),
		offset: offset,
		size:   size,
	}
}

func (s *Store) Put(key, val string) (bool, error) {
	if s == nil || s.data == nil {
		return false, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)
	s.data[key] = val

	// if store is not filled, abort the function
	if len(s.data) < s.size {
		return true, nil
	}

	// fetch a new instance to store 2000 (or the given size) records in log files
	sstable, err := sstable.NewSSTable(s.offset+1, constants.FLUSH)
	if err != nil {
		return false, err
	}

	err = sstable.Flush(s.data)
	if err != nil {
		return false, err
	}

	s.data = make(map[string]string, s.size)
	s.offset = s.offset + 1

	return true, nil
}

func (s *Store) Get(key string) (string, error) {
	if s == nil || s.data == nil {
		return constants.EMPTYSTRING, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)

	val, ok := s.data[key]
	if ok {
		return val, nil
	}

	var c = s.offset
	for c > 0 {
		sstable, err := sstable.NewSSTable(c, constants.READ)
		if err != nil {
			return constants.EMPTYSTRING, err
		}

		val, err = sstable.Read(key)
		if err != nil && err.Error() != constants.ERRNOTFOUND {
			return constants.EMPTYSTRING, err
		}

		if len(val) > 0 {
			return val, nil
		}

		c--
	}

	return "NOT_FOUND", errors.New(constants.ERRNOTFOUND)
}
