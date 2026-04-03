package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/pkg/constants"
	"monk-db/pkg/sstable"
	"monk-db/pkg/utils"
	"strings"
	"sync"
)

var (
	walFileName string
	walFilePath string
)

func SetWALFilePathAndCreate(name, path string) (*utils.File, error) {
	var walFile = utils.NewFile(name, path)
	err := walFile.Create(utils.CREATE, true)
	if err != nil {
		log.Printf("create wal file err: %v\n", err.Error())
		return nil, errors.New("unable to create wal file")
	}

	log.Println("wal file created successfully")

	walFileName = name
	walFilePath = path
	return walFile, nil
}

type Store struct {
	data    map[string]string
	walFile *utils.File
	offset  int
	size    int
}

type WalRecord struct {
	Operation string `json:"op"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func InitStore(size int, walFileName, walFilePath string) (*Store, error) {
	// 1. Create and Set the file
	walFile, err := SetWALFilePathAndCreate(walFileName, walFilePath)
	if err != nil {
		return nil, err
	}

	var store = &Store{
		data:    make(map[string]string, size),
		size:    size,
		walFile: walFile,
	}

	// 2. load data from wal file and handle dangling files on disk
	var dataChan = make(chan map[string]string, size)
	var errorChan = make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go loadFromWalFile(walFile, size, dataChan, errorChan, &wg)

	var offsetChan = make(chan int, 1)
	wg.Add(1)
	go handleDanglingFileAndGetOffset(sstable.GetManifestLogfileMetadata(), sstable.GetRecordDirMetadata(), offsetChan, errorChan, &wg)

	go func() {
		wg.Wait()
		close(errorChan)
		close(dataChan)
		close(offsetChan)
	}()

	for e := range errorChan {
		if e != nil {
			return nil, e
		}
	}

	var data = make(map[string]string, 0)
	for d := range dataChan {
		data = d
	}

	for o := range offsetChan {
		fmt.Println("offset: ", o)
		store.offset = o
	}

	if data != nil || len(data) != 0 {
		store.data = data
	}

	return store, nil
}

func (s *Store) Put(key, val string) (bool, error) {
	if s == nil || s.data == nil {
		return false, errors.New(constants.ERRORSTORAGENOTINITIALIZED)
	}

	var walRecord = WalRecord{
		Operation: constants.PUT,
		Key:       key,
		Value:     val,
	}
	walRecordBytes, err := json.Marshal(walRecord)
	if err != nil {
		return false, errors.New("unable to marshal data")
	}
	walRecordBytes = append(walRecordBytes, []byte("\n")...)

	err = s.walFile.Write(walRecordBytes, utils.APPEND, true)
	if err != nil {
		return false, errors.New("unable to write wal")
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

	// flush to disk if limit reached
	err = sstable.Flush(s.data)
	if err != nil {
		return false, err
	}

	// reset the file
	err = s.walFile.Reset(true)
	if err != nil {
		return false, err
	}

	err = s.walFile.Close()
	if err != nil {
		return false, errors.New("unable to close file")
	}

	// reset the memtable
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
