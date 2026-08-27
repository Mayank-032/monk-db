package sstable

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/internal/constants"
	"monk-db/internal/io/file"
	"monk-db/internal/models"
	"strconv"
	"strings"
)

func readManifestFileData(manifestFile *file.File, level int) ([]string, error) {
	fileContent, err := manifestFile.Read()
	if err != nil {
		return nil, err
	}

	var fileContentStr = string(fileContent)

	var fileContentPart = strings.Split(fileContentStr, "\n")
	if len(fileContentPart) <= 1 {
		return nil, err
	}

	var files = fileContentPart[:len(fileContentPart)-1]

	if level < 0 {
		return files, nil
	}

	var currLevelPath string = fmt.Sprintf("l%d", level)
	var currLevelFiles = make([]string, 0)
	for _, file := range files {
		if strings.Contains(file, currLevelPath) {
			currLevelFiles = append(currLevelFiles, file)
		}
	}

	return currLevelFiles, nil
}

func readManifestFileDataForHigherLevels(manifestFle *file.File, level, levelPlusOne int) ([]string, error) {
	return []string{}, nil
}

func readRecordsFileData(file *file.File) ([]models.Record, error) {
	contentBytes, err := file.Read()
	if err != nil {
		return nil, errors.New("unable to read file")
	}

	var records = make([]models.Record, 0)

	if len(contentBytes) == 0 {
		return records, nil
	}

	err = json.Unmarshal(contentBytes, &records)
	if err != nil {
		fmt.Println("contentBytes: ", string(contentBytes))
		fmt.Println("sst-file-name: ", file.GetName())
		fmt.Println("sst-file-path: ", file.GetPath())
		fmt.Println("sst-file-fullpath: ", file.GetFileFullPath())

		return nil, err
	}

	return records, nil
}

func calculateOffset(lastFile string) (int, error) {
	fileParts := strings.Split(lastFile, ".")
	if len(fileParts) < 1 {
		return -1, errors.New("invalid file format")
	}

	fileName := fileParts[0]
	fileNameParts := strings.Split(fileName, "-")
	if len(fileParts) < 2 {
		return -1, errors.New("invalid filename format")
	}

	lastOffset, err := strconv.Atoi(fileNameParts[1])
	if err != nil {
		return -1, errors.New("unable to convert offset to int")
	}

	return lastOffset, nil
}

func createFileAndWriteData(level, offset int, overwrite bool, finalRecord []models.Record, manifestFile *file.File) error {
	finalRecordB, err := json.MarshalIndent(finalRecord, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		log.Printf("marshal records err: %v\n", err.Error())
		return errors.New("unable to marshal records while compaction")
	}
	log.Println("marshalled final records")

	var firstKey = finalRecord[0]
	var lastKey = finalRecord[len(finalRecord)-1]

	var newFileName = fmt.Sprintf("%s-%s: sst-%d.json", firstKey.Key, lastKey.Key, (offset + 1))
	var newFile = file.NewFile(newFileName, fmt.Sprintf("%s/%d", constants.RECORDS_DIRPATH, level))
	if err = newFile.Create(file.CREATE, true); err != nil {
		log.Printf("create compact file err: %v\n", err.Error())
		return errors.New("unable to create compact record file")
	}

	if err = newFile.Write(finalRecordB, file.WRITEONLY, true); err != nil {
		log.Printf("write compact file err: %v\n", err.Error())
		return errors.New("unable to compact records")
	}
	log.Println("file created with final records")

	// overwrite the manifest file with new data
	if err = manifestFile.AppendWithTmpFile([]byte(newFileName), overwrite); err != nil {
		log.Println("unable to write content in manifest-log file")
		return errors.New("unable to flush records")
	}
	log.Println("overwrite the manifest file")

	return nil
}
