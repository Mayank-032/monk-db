package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/internal/constants"
	"monk-db/internal/io/file"
	"monk-db/internal/sstable"
	"os"
	"strconv"
	"strings"
	"sync"
)

func loadFromWalFile(
	walFile *file.File,
	size int,
	dataChan chan map[string]Metadata,
	errorChan chan error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	walFileBytes, err := walFile.Read()
	if err != nil {
		dataChan <- nil
		errorChan <- err
		return
	}

	var data = make(map[string]Metadata, size)

	var walFileStr = string(walFileBytes)
	if len(walFileStr) == 0 {
		dataChan <- nil
		errorChan <- nil
		return
	}

	var walFileRecords = strings.Split(walFileStr, "\n")
	if len(walFileRecords) == 0 {
		dataChan <- nil
		errorChan <- err
		return
	}

	for _, record := range walFileRecords {
		if len(record) == 0 {
			continue
		}

		var walRecord WalRecord
		err = json.Unmarshal([]byte(record), &walRecord)
		if err != nil {
			log.Println("unable to unmarshal record while loading store")
			dataChan <- nil
			errorChan <- err
			return
		}

		switch walRecord.Operation {
		case constants.PUT:
			data[walRecord.Key] = Metadata{
				Key: walRecord.Key,
				Val: walRecord.Value,
			}
		case constants.GET:
			data[walRecord.Key] = Metadata{
				Key:       walRecord.Key,
				Val:       walRecord.Value,
				isDeleted: true,
			}
		case constants.DELETE:
			data[walRecord.Key] = Metadata{
				Key:       walRecord.Key,
				Val:       walRecord.Value,
				isDeleted: true,
			}
		}
	}

	dataChan <- data
	errorChan <- nil
}

func handleDanglingFileAndGetOffset(
	manifestFile *file.File,
	recordDir string,
	offsetChan chan int,
	errorChan chan error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	fileContent, err := manifestFile.Read()
	if err != nil {
		offsetChan <- 0
		errorChan <- err
		return
	}

	var fileContentStr = string(fileContent)

	var fileContentPart = strings.Split(fileContentStr, "\n")
	if len(fileContentPart) <= 1 {
		offsetChan <- 0
		errorChan <- nil
		return
	}

	files, err := os.ReadDir(recordDir)
	if err != nil {
		offsetChan <- 0
		errorChan <- errors.New(fmt.Sprint("unable to read dir: ", err))
		return
	}

	for _, f := range files {
		var isFileFound bool
		for _, part := range fileContentPart {
			if strings.EqualFold(f.Name(), part) {
				isFileFound = true
				break
			}
		}

		if !isFileFound {
			var newFile = file.NewFile(f.Name(), recordDir)
			err = newFile.Remove()
			if err != nil {
				offsetChan <- 0
				errorChan <- errors.New(fmt.Sprint("unable to remove invalid file on disk: ", err))
				return
			}
			fmt.Println("file removed success")
		}
	}

	offset, err := getOffsetFromManifestFile(fileContentStr, fileContentPart)
	if err != nil {
		offsetChan <- offset
		errorChan <- err
		return
	}

	offsetChan <- offset
	errorChan <- nil
}

func getOffsetFromManifestFile(fileContentStr string, fileContentPart []string) (int, error) {
	var lastLineContent = fileContentPart[len(fileContentPart)-2]
	if lastLineContent == fileContentStr {
		return 0, nil
	}

	var firstRecordFileName = strings.Split(lastLineContent, ".")[0]
	if fileContentStr == firstRecordFileName {
		return 0, nil
	}

	var recordFileCounter = strings.Split(firstRecordFileName, "-")
	if len(recordFileCounter) < 2 {
		return 0, nil
	}

	var counter = recordFileCounter[1]
	counterInt, err := strconv.Atoi(counter)
	if err != nil {
		log.Println("unable to convert counter value to integer")
		return -1, err
	}

	return counterInt, nil
}

func convertToSSTableDataFormat(data map[string]Metadata) []sstable.Record {
	var sstableRecords = make([]sstable.Record, 0, len(data))
	for key, metadata := range data {
		sstableRecords = append(sstableRecords, sstable.Record{
			Key:       key,
			Value:     metadata.Val,
			IsDeleted: metadata.isDeleted,
		})
	}

	return sstableRecords
}
