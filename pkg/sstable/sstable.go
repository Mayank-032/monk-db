package sstable

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/pkg/constants"
	"monk-db/pkg/utils"
	"os"
)

var (
	ssTableRecordsDirPath string

	manifestFileName    string
	manifestLogFilePath string
)

func SetManifestLogfilePathAndCreate(name, path string) error {
	var manifestFile = utils.NewFile(manifestFileName, path)
	err := manifestFile.Create()
	if err != nil {
		log.Printf("create manifest file err: %v\n", err.Error())
		return errors.New("unable to create manifest file")
	}

	manifestLogFilePath = path
	return nil
}

func SetSSTableRecordsDirPathAndCreate(path string) error {
	err := os.Mkdir(path, constants.DIRPERMISSION)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Printf("create records dir err: %v\n", err.Error())
		return errors.New("unable to create records dir")
	}

	ssTableRecordsDirPath = path
	return nil
}

type ssTable struct {
	file *utils.File
}

func NewSSTable(count int, operation string) (*ssTable, error) {
	var fileName = fmt.Sprintf("sst-%v.json", count)
	var pathToFile = fmt.Sprintf("%v/%v", ssTableRecordsDirPath, fileName)

	var file *utils.File
	var err error
	switch operation {
	case constants.FLUSH:
		file = utils.NewFile(fileName, pathToFile)
		err = file.Create()
	case constants.READ:
		file = utils.NewFile(fileName, pathToFile)
		err = file.Get()
	}

	if err != nil {
		log.Println("sstable init err: ", err.Error())
		return nil, errors.New("unable to init sstable")
	}

	return &ssTable{
		file: file,
	}, nil
}

// it will convert to respective data structure and flush it to the log file
func (sst *ssTable) Flush(data map[string]string) error {
	if sst == nil {
		return errors.New("sstable is not initialized")
	}

	var sstableRecords = make([]Record, 0, len(data))
	for key, val := range data {
		sstableRecords = append(sstableRecords, Record{
			Key:   key,
			Value: val,
		})
	}

	recordBytes, err := json.MarshalIndent(sstableRecords, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		log.Printf("marshal records err: %v\n", err.Error())
		return errors.New("unable to flush records")
	}

	err = sst.file.Write(recordBytes)
	if err != nil {
		log.Println("unable to write content in record file")
		return errors.New("unable to flush records")
	}

	var manifestFile = utils.NewFile("manifest.txt", manifestLogFilePath)
	err = manifestFile.Create()
	if err != nil {
		return err
	}

	err = manifestFile.Prepend([]byte(sst.file.GetName()))
	if err != nil {
		log.Println("unable to write content in manifest-log file")
		return errors.New("unable to flush records")
	}

	return nil
}

func (sst *ssTable) Read(key string) (string, error) {
	if sst == nil {
		return "", errors.New("sstable is not initialized")
	}

	val, err := utils.Cache.Get(sst.file.GetName())
	if err == nil {
		records, ok := val.([]Record)
		if ok {
			for _, r := range records {
				if r.Key == key {
					return r.Value, nil
				}
			}
			return "", nil
		}
	}

	if err != nil && err.Error() != constants.ERRNOTFOUND {
		log.Println("unable to check from cache with err: ", err.Error())
		return "", errors.New("unable to check from cache")
	}

	contentBytes, err := sst.file.Read()
	if err != nil {
		log.Println("read file content error: ", err.Error())
		return "", errors.New("unable to read file")
	}

	var records = make([]Record, 0)
	err = json.Unmarshal(contentBytes, &records)
	if err != nil {
		log.Println("unmarshal file content err: ", err.Error())
		return "", errors.New("unable to read file")
	}
	utils.Cache.PUT(sst.file.GetName(), records)

	for _, r := range records {
		if r.Key == key {
			return r.Value, nil
		}
	}

	return "", nil
}
