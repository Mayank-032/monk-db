package sstable

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/pkg/utils"
	"strings"
)

func readManifestFileData(manifestFile *utils.File) ([]string, error) {
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

func readRecordsFileData(file *utils.File) ([]Record, error) {
	contentBytes, err := file.Read()
	if err != nil {
		log.Println("read file content error: ", err.Error())
		return nil, errors.New("unable to read file")
	}

	var records = make([]Record, 0)
	err = json.Unmarshal(contentBytes, &records)
	if err != nil {
		fmt.Println("contentBytes: ", string(contentBytes))
		fmt.Println("sst-file-name: ", file.GetName())
		fmt.Println("sst-file-path: ", file.GetPath())
		fmt.Println("sst-file-fullpath: ", file.GetFileFullPath())

		log.Println("unmarshal file content err: ", err.Error())
		return nil, errors.New("unable to read file")
	}

	return records, nil
}
