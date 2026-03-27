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
		log.Printf("unable to create manifest file with err: %v\n", err.Error())
		return err
	}

	manifestLogFilePath = path
	return nil
}

func SetSSTableRecordsDirPathAndCreate(path string) error {
	err := os.Mkdir(path, constants.DIRPERMISSION)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Printf("unable to create records dir with err: %v\n", err.Error())
		return err
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
		log.Println("err: ", err.Error())
		return nil, err
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
		log.Printf("unable to marshal records with err: %v\n", err.Error())
		return err
	}

	err = sst.file.Write(recordBytes)
	if err != nil {
		log.Println("unable to write content in record file")
		return err
	}

	var manifestFile = utils.NewFile("manifest.txt", manifestLogFilePath)
	err = manifestFile.Create()
	if err != nil {
		return err
	}

	err = manifestFile.Prepend([]byte(sst.file.GetName()))
	if err != nil {
		log.Println("unable to write content in manifest-log file")
		return err
	}

	return nil
}

func (sst *ssTable) Read(key string) (string, error) {
	if sst == nil {
		return "", errors.New("sstable is not initialized")
	}

	contentBytes, err := sst.file.Read()
	if err != nil {
		log.Println("unable to read file content")
		return "", err
	}

	var records = make([]Record, 0)
	err = json.Unmarshal(contentBytes, &records)
	if err != nil {
		log.Println("unable to unmarshal file content")
		return "", err
	}

	for _, r := range records {
		if r.Key == key {
			return r.Value, nil
		}
	}

	return "", nil
}
