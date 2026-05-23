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
	"strconv"
	"strings"
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
	log.Println("[FLUSH START]")
	log.Println()

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

	err = manifestFile.AppendWithTmpFile([]byte(sst.file.GetName()), false)
	if err != nil {
		log.Println("unable to write content in manifest-log file")
		return errors.New("unable to flush records")
	}

	return nil
}

func (sst *ssTable) Read(key string) (string, error) {
	log.Println("[SST-READ START]")
	log.Println()

	if sst == nil {
		return "", errors.New("sstable is not initialized")
	}

	val, err := utils.Cache.Get(sst.file.GetName())
	if err == nil {
		records, ok := val.([]Record)
		if ok {
			for _, r := range records {
				if r.Key == key && r.IsDeleted {
					return constants.EMPTYSTRING, errors.New(constants.ERRRESOURCEREMOVED)
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
	if err != nil {
		log.Println("unable to read records error: ", err.Error())
		return constants.EMPTYSTRING, errors.New("unable to read records")
	}

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
			if !r.IsDeleted {
				return r.Value, nil
			}

			return constants.EMPTYSTRING, errors.New(constants.ERRRESOURCEREMOVED)
		}
	}

	return constants.EMPTYSTRING, errors.New(constants.ERRNOTFOUND)
}

// This optimizes the sstable storage and returns the latest offset
func Optimize() (int, error) {
	log.Println("[OPTIMIZE START]")
	log.Println()

	/* 1) Let's start with fetching all the data in-memory at once */

	// Read Manifest File to get the list of existing files
	var manifestFile = utils.NewFile(manifestFileName, manifestLogFilePath)
	files, err := readManifestFileData(manifestFile)
	if err != nil {
		log.Println("unable to read manifest file with err: ", err.Error())
		return -1, errors.New("unable to read file")
	}
	log.Println("read all files")

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
	log.Println("unmarshalled all files")

	/* 2) Perform merge k-sorted-list algorithm. */
	var pq = utils.NewPriorityQueue(func(p1, p2 Pair) (comp bool) {
		var val1 string = p1.record.Key
		var val2 string = p2.record.Key

		if val1 < val2 {
			return true
		}

		if val1 == val2 {
			if p1.listIdx > p2.listIdx {
				return true
			}
		}

		return false
	})
	log.Println("created priority queue")

	for i, record := range list {
		pq.Push(Pair{
			record:  record[0],
			listIdx: i,
			idx:     0,
		})

		// data := pq.GetData()
		// fmt.Println("data: ", data)
	}
	log.Println("added first elements priority queue from all files")

	var finalRecord = make([]Record, 0)
	for !pq.IsEmpty() {
		var p, err = pq.Pop()
		if err != nil {
			return -1, err
		}

		// remove the remaining elements
		for !pq.IsEmpty() {
			var topEle, err = pq.Peek()
			if err != nil {
				return -1, err
			}

			if p.record.Key == topEle.record.Key {
				topEle, err = pq.Pop()
				if err != nil {
					return -1, err
				}

				if topEle.idx+1 >= len(list[topEle.listIdx]) {
					continue
				}

				var newP = Pair{
					record:  list[topEle.listIdx][topEle.idx+1],
					listIdx: topEle.listIdx,
					idx:     topEle.idx + 1,
				}
				pq.Push(newP)

				continue
			}

			break
		}

		// append in current final list
		if !p.record.IsDeleted {
			finalRecord = append(finalRecord, p.record)
		}

		// if for current list the elements are over.. pop another element, as no point of moving further
		if p.idx+1 >= len(list[p.listIdx]) {
			continue
		}

		var newP = Pair{
			record:  list[p.listIdx][p.idx+1],
			listIdx: p.listIdx,
			idx:     p.idx + 1,
		}
		pq.Push(newP)
	}
	log.Println("priority queue processing finished")

	finalRecordB, err := json.MarshalIndent(finalRecord, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		log.Printf("marshal records err: %v\n", err.Error())
		return -1, errors.New("unable to marshal records while compaction")
	}
	log.Println("marshalled final records")

	// 3) calculate the new offset
	lastFile := files[len(files)-1]
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
	log.Println("calculate new offset")

	// For now if the file exceeds the limit of 2000 not an issue
	var newFileName = fmt.Sprintf("sst-%d.json", (lastOffset + 1))
	var newFile = utils.NewFile(newFileName, ssTableRecordsDirPath)
	if err = newFile.Create(utils.CREATE, true); err != nil {
		log.Printf("create compact file err: %v\n", err.Error())
		return -1, errors.New("unable to create compact record file")
	}

	if err = newFile.Write(finalRecordB, utils.WRITEONLY, true); err != nil {
		log.Printf("write compact file err: %v\n", err.Error())
		return -1, errors.New("unable to compact records")
	}
	log.Println("file created with final records")

	// overwrite the manifest file with new data
	if err = manifestFile.AppendWithTmpFile([]byte(newFileName), true); err != nil {
		log.Println("unable to write content in manifest-log file")
		return -1, errors.New("unable to flush records")
	}
	log.Println("overwrite the manifest file")

	// cleanup stale files
	for _, fileName := range files {
		var file = utils.NewFile(fileName, ssTableRecordsDirPath)
		if err = file.Remove(); err != nil {
			log.Println("unable to records of the file with err: ", err.Error())
			return 0, errors.New("unable to read records")
		}
	}
	log.Println("cleanup the stale files")
	log.Println("optimized_file_offset: ", (lastOffset + 1))

	return lastOffset + 1, nil
}

// euelf
