package storage

import (
	"encoding/json"
	"log"
	"monk-db/internal/constants"
	"monk-db/internal/io/file"
	"monk-db/internal/models"
	"strconv"
	"strings"
)

func loadFromWalFile(walFile *file.File, size int) (map[string]Metadata, error) {
	walFileBytes, err := walFile.Read()
	if err != nil {
		return nil, err
	}

	var data = make(map[string]Metadata, size)

	var walFileStr = string(walFileBytes)
	if len(walFileStr) == 0 {
		return nil, nil
	}

	var walFileRecords = strings.Split(walFileStr, "\n")
	if len(walFileRecords) == 0 {
		return nil, nil
	}

	for _, record := range walFileRecords {
		if len(record) == 0 {
			continue
		}

		var walRecord WalRecord
		err = json.Unmarshal([]byte(record), &walRecord)
		if err != nil {
			log.Println("unable to unmarshal record while loading store")
			return nil, err
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

	return data, nil
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

func convertToSSTableDataFormat(data map[string]Metadata) []models.Record {
	var sstableRecords = make([]models.Record, 0, len(data))
	for key, metadata := range data {
		sstableRecords = append(sstableRecords, models.Record{
			Key:       key,
			Value:     metadata.Val,
			IsDeleted: metadata.isDeleted,
		})
	}

	return sstableRecords
}
