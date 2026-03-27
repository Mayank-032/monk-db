package sstable

import (
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
	count int
}

func NewSSTable(count int) *ssTable {
	return &ssTable{
		count: count,
	}
}

// it will convert to respective data structure and flush it to the log file
func (sst *ssTable) Flush(data map[string]string) error {
	if sst == nil {
		log.Println("sstable is not initialized")
		return errors.New("sstable is not initialized")
	}

	var sstableRecords = make([]Record, 0, len(data))
	for key, val := range data {
		sstableRecords = append(sstableRecords, Record{
			Key:   key,
			Value: val,
		})
	}

	var fileName = fmt.Sprintf("sst-%v.json", sst.count)
	var pathToFile = fmt.Sprintf("%v/%v", ssTableRecordsDirPath, fileName)
	var file = utils.NewFile(fileName, pathToFile)
	var err = file.Create()
	if err != nil {
		log.Printf("unable to create a file with err: %v\n", err.Error())
		return err
	}

	err = file.Write(sstableRecords)
	if err != nil {
		log.Printf("unable to write data to file with err: %v\n", err.Error())
		return err
	}

	var manifestFile = utils.NewFile("manifest.txt", manifestLogFilePath)
	err = manifestFile.Create()
	if err != nil {
		log.Printf("unable to create a file with err: %v\n", err.Error())
		return err
	}

	err = manifestFile.Prepend([]byte(file.GetName()))
	if err != nil {
		log.Printf("unable to write data to manifest file with err: %v\n", err.Error())
		return err
	}

	return nil
}
