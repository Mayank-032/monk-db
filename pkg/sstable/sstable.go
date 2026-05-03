package sstable

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/pkg/constants"
	"monk-db/pkg/utils"
	"os"
	"sort"
)

var (
	ssTableRecordsDirPath string

	manifestFileName    string
	manifestLogFilePath string
)

func SetManifestLogfilePathAndCreate(name, path string) error {
	var manifestFile = utils.NewFile(name, path)
	err := manifestFile.Create(utils.CREATE, true)
	if err != nil {
		log.Printf("create manifest file err: %v\n", err.Error())
		return errors.New("unable to create manifest file")
	}

	manifestFileName = name
	manifestLogFilePath = path
	return nil
}

func GetManifestLogfileMetadata() *utils.File {
	var manifestFile = utils.NewFile(manifestFileName, manifestLogFilePath)
	return manifestFile
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

func GetRecordDirMetadata() string {
	return ssTableRecordsDirPath
}

type ssTable struct {
	file *utils.File
}

func NewSSTable(count int, operation string) (*ssTable, error) {
	var fileName = fmt.Sprintf("sst-%v.json", count)
	var pathToFile = ssTableRecordsDirPath

	var file *utils.File
	var err error
	switch operation {
	case constants.FLUSH:
		file = utils.NewFile(fileName, pathToFile)
		err = file.Create(utils.DEFAULT, true)
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
func (sst *ssTable) Flush(sstableRecords []Record) error {
	if sst == nil {
		return errors.New("sstable is not initialized")
	}

	sort.SliceStable(sstableRecords, func(i, j int) bool {
		return sstableRecords[i].Key < sstableRecords[j].Key
	})

	recordBytes, err := json.MarshalIndent(sstableRecords, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		log.Printf("marshal records err: %v\n", err.Error())
		return errors.New("unable to flush records")
	}

	err = sst.file.Write(recordBytes, utils.DEFAULT, true)
	if err != nil {
		log.Println("unable to write content in record file")
		return errors.New("unable to flush records")
	}

	var manifestFile = utils.NewFile(manifestFileName, manifestLogFilePath)
	err = manifestFile.Get()
	if err != nil {
		return err
	}

	err = manifestFile.AppendWithTmpFile([]byte(sst.file.GetName()))
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
				if r.IsDeleted {
					return constants.EMPTYSTRING, errors.New(constants.ERRNOTFOUND)
				}

				if r.Key == key {
					return r.Value, nil
				}
			}

			return constants.EMPTYSTRING, errors.New(constants.ERRNOTFOUND)
		}
	}

	if err != nil && err.Error() != constants.ERRNOTFOUND {
		log.Println("unable to check from cache with err: ", err.Error())
		return constants.EMPTYSTRING, errors.New("unable to check from cache")
	}

	contentBytes, err := sst.file.Read()
	if err != nil {
		log.Println("read file content error: ", err.Error())
		return constants.EMPTYSTRING, errors.New("unable to read file")
	}

	records, err := readRecordsFileData(sst.file)
	err = json.Unmarshal(contentBytes, &records)
	if err != nil {
		fmt.Println("contentBytes: ", string(contentBytes))
		fmt.Println("sst-file-name: ", sst.file.GetName())
		fmt.Println("sst-file-path: ", sst.file.GetPath())
		fmt.Println("sst-file-fullpath: ", sst.file.GetFileFullPath())

		log.Println("unmarshal file content err: ", err.Error())
		return constants.EMPTYSTRING, errors.New("unable to read file")
	}
	utils.Cache.PUT(sst.file.GetName(), records)

	for _, r := range records {
		if r.Key == key {
			return r.Value, nil
		}
	}

	return constants.EMPTYSTRING, errors.New(constants.ERRNOTFOUND)
}

// This optimizes the sstable storage and returns the latest offset
// TODO: We gonna optimize after init implementation
func (sst *ssTable) Optimize() (int, error) {
	/* 1) Let's start with fetching all the data in-memory at once */
	
	// Read Manifest File to get the list of existing files
	var manifestFile = utils.NewFile(manifestFileName, manifestLogFilePath)
	files, err := readManifestFileData(manifestFile)
	if err != nil {
		log.Println("unable to read manifest file with err: ", err.Error())
		return -1, errors.New("unable to read file")
	}

	var list = make([][]Record, 0)
	for _, fileName := range files {
		var file = utils.NewFile(fileName, ssTableRecordsDirPath)
		records, err := readRecordsFileData(file)
		if err != nil {
			log.Println("unable to records of the file with err: ", err.Error())
			return 0, errors.New("unable to read records")
		}

		list = append(list, records)
	}

	/* 2) Perform merge k-sorted-list algorithm. */

	// For now if the file exceeds the limit of 2000 not an issue
	

	// 3) calculate the new offset


	return 0, nil
}