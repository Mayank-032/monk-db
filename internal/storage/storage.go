package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/internal/constants"
	"monk-db/internal/io/file"
	"strings"
)

var (
	walFileName string
	walFilePath string
)

func SetWALFilePathAndCreate(name, path string) (*file.File, error) {
	var walFile = file.NewFile(name, path)
	err := walFile.Create(file.CREATE, true)
	if err != nil {
		log.Printf("create wal file err: %v\n", err.Error())
		return nil, errors.New("unable to create wal file")
	}

	log.Println("wal file created successfully")

	walFileName = name
	walFilePath = path
	return walFile, nil
}

type Metadata struct {
	Key       string
	Val       string
	isDeleted bool
}

type Store struct {
	data        map[string]Metadata
	diskStorage IDiskStorage
	walFile     *file.File
	size        int
}

type WalRecord struct {
	Operation string `json:"op"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func InitStore(
	size int,
	walFileName, walFilePath string,
	diskStorage IDiskStorage,
) (*Store, error) {
	// 1. Create and Set the file
	walFile, err := SetWALFilePathAndCreate(walFileName, walFilePath)
	if err != nil {
		return nil, err
	}

	var store = &Store{
		data:        make(map[string]Metadata, size),
		size:        size,
		walFile:     walFile,
		diskStorage: diskStorage,
	}

	// 2.1 load data from wal file
	data, err := loadFromWalFile(walFile, size)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return store, nil
	}
	store.data = data

	return store, nil
}

func (s *Store) Put(key, val string, isDeleted bool) (bool, error) {
	log.Println("[PUT START] - ", key)

	if s == nil || s.data == nil {
		return false, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)

	var op = constants.PUT
	if isDeleted {
		op = constants.DELETE
	}

	var walRecord = WalRecord{
		Operation: op,
		Key:       key,
		Value:     val,
	}
	walRecordBytes, err := json.Marshal(walRecord)
	if err != nil {
		return false, fmt.Errorf("unable to put while marshal wal records err %w", err)
	}
	walRecordBytes = append(walRecordBytes, []byte("\n")...)

	err = s.walFile.Write(walRecordBytes, file.APPEND, true)
	if err != nil {
		return false, fmt.Errorf("unable to put while write to wal file err %w", err)
	}

	s.data[key] = Metadata{
		Key:       key,
		Val:       val,
		isDeleted: isDeleted,
	}

	// if store is not filled, abort the function
	if len(s.data) < s.size {
		return true, nil
	}

	// fetch a new instance to store 2000 (or the given size) records in log files
	var sstableRecords = convertToSSTableDataFormat(s.data)

	// flush to disk if limit reached
	err = s.diskStorage.Flush(sstableRecords)
	if err != nil {
		return false, fmt.Errorf("unable to put while flush err %w", err)
	}

	// reset the file
	err = s.walFile.Reset(true)
	if err != nil {
		return false, fmt.Errorf("unable to put while wal file reset err %w", err)
	}

	err = s.walFile.Close()
	if err != nil {
		return false, fmt.Errorf("unable to put while wal file close err %w", err)
	}

	// reset the memtable
	s.data = make(map[string]Metadata, s.size)

	return true, nil
}

func (s *Store) Delete(key string) (bool, error) {
	log.Println("[DELETE START] - ", key)

	val, err := s.Get(key)
	if err != nil {
		return false, fmt.Errorf("unable to delete from disk while read err %w", err)
	}

	return s.Put(key, val, true)
}

func (s *Store) Get(key string) (string, error) {
	log.Println("[GET START] - ", key)

	if s == nil || s.data == nil {
		return constants.EMPTYSTRING, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	key = strings.ToLower(key)

	metadata, ok := s.data[key]
	if ok {
		if metadata.isDeleted {
			return "NOT_FOUND", errors.New(constants.ERRNOTFOUND)
		}

		return metadata.Val, nil
	}

	val, err := s.diskStorage.Read(key)
	if err != nil {
		if err.Error() == constants.ERRRESOURCEREMOVED {
			return "NOT_FOUND", errors.New(constants.ERRNOTFOUND)
		}

		if err.Error() != constants.ERRNOTFOUND {
			return constants.EMPTYSTRING, fmt.Errorf("unable to read from disk with err %w", err)
		}
	}

	if len(val) > 0 {
		return val, nil
	}

	return "NOT_FOUND", errors.New(constants.ERRNOTFOUND)
}

func (s *Store) Compact() error {
	return s.diskStorage.Optimize()
}
