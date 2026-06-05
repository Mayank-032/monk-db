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

func readManifestFileData(manifestFile *file.File) ([]string, error) {
	fileContent, err := manifestFile.Read()
	if err != nil {
		return nil, err
	}

	var fileContentStr = string(fileContent)

	var fileContentPart = strings.Split(fileContentStr, "\n")
	if len(fileContentPart) <= 1 {
		return nil, err
	}

	return fileContentPart[:len(fileContentPart)-1], nil
}

func readRecordsFileData(file *file.File) ([]models.Record, error) {
	contentBytes, err := file.Read()
	if err != nil {
		return nil, errors.New("unable to read file")
	}

	var records = make([]models.Record, 0)
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

func createFileAndWriteData(offset int, overwrite bool, finalRecord []models.Record, manifestFile *file.File, recordsDirPath string) error {
	finalRecordB, err := json.MarshalIndent(finalRecord, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		log.Printf("marshal records err: %v\n", err.Error())
		return errors.New("unable to marshal records while compaction")
	}
	log.Println("marshalled final records")

	var newFileName = fmt.Sprintf("sst-%d.json", (offset + 1))
	var newFile = file.NewFile(newFileName, recordsDirPath)
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
