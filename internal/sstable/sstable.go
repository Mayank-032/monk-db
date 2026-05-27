package sstable

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"monk-db/internal/constants"
	"monk-db/internal/ds/heap"
	cache "monk-db/internal/ds/lru_cache"
	"monk-db/internal/io/file"
	"os"
	"sort"
)

var (
	ssTableRecordsDirPath string

	manifestFileName    string
	manifestLogFilePath string
)

func SetManifestLogfilePathAndCreate(name, path string) error {
	var manifestFile = file.NewFile(name, path)
	err := manifestFile.Create(file.CREATE, true)
	if err != nil {
		log.Printf("create manifest file err: %v\n", err.Error())
		return errors.New("unable to create manifest file")
	}

	manifestFileName = name
	manifestLogFilePath = path
	return nil
}

func GetManifestLogfileMetadata() *file.File {
	var manifestFile = file.NewFile(manifestFileName, manifestLogFilePath)
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
	file *file.File
}

func NewSSTable(count int, operation string) (*ssTable, error) {
	var fileName = fmt.Sprintf("sst-%v.json", count)
	var pathToFile = ssTableRecordsDirPath

	var newFile *file.File
	var err error
	switch operation {
	case constants.FLUSH:
		newFile = file.NewFile(fileName, pathToFile)
		err = newFile.Create(file.DEFAULT, true)
	case constants.READ:
		newFile = file.NewFile(fileName, pathToFile)
		err = newFile.Get()
	}

	if err != nil {
		log.Println("sstable init err: ", err.Error())
		return nil, errors.New("unable to init sstable")
	}

	return &ssTable{
		file: newFile,
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

	err = sst.file.Write(recordBytes, file.DEFAULT, true)
	if err != nil {
		log.Println("unable to write content in record file")
		return errors.New("unable to flush records")
	}

	var manifestFile = file.NewFile(manifestFileName, manifestLogFilePath)
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

	val, err := cache.Cache.Get(sst.file.GetName())
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
	cache.Cache.PUT(sst.file.GetName(), records)

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
func Optimize() (int, int, error) {
	log.Println("[OPTIMIZE START]")
	log.Println()

	/* 1) Let's start with fetching all the data in-memory at once */

	// Read Manifest File to get the list of existing files
	var manifestFile = file.NewFile(manifestFileName, manifestLogFilePath)
	files, err := readManifestFileData(manifestFile)
	if err != nil {
		log.Println("unable to read manifest file with err: ", err.Error())
		return -1, -1, errors.New("unable to read file")
	}
	log.Println("read all files")

	var list = make([][]Record, 0)
	for _, fileName := range files {
		var file = file.NewFile(fileName, ssTableRecordsDirPath)
		records, err := readRecordsFileData(file)
		if err != nil {
			log.Println("unable to records of the file with err: ", err.Error())
			return -1, -1, errors.New("unable to read records")
		}

		list = append(list, records)
	}
	log.Println("unmarshalled all files")

	// 2) calculate the new offset
	lastOffset, err := calculateOffset(files[len(files)-1])
	if err != nil {
		return -1, -1, errors.New("unable to convert offset to int")
	}
	log.Println("calculate new offset")

	var tempOffset = lastOffset

	/* 3) Perform merge k-sorted-list algorithm. */
	var pq = heap.NewHeap(func(p1, p2 Pair) (comp bool) {
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
	var overwrite bool = true
	for !pq.IsEmpty() {
		var p, err = pq.Pop()
		if err != nil {
			return -1, -1, err
		}

		// remove the remaining elements
		for !pq.IsEmpty() {
			var topEle, err = pq.Peek()
			if err != nil {
				return -1, -1, err
			}

			if p.record.Key == topEle.record.Key {
				topEle, err = pq.Pop()
				if err != nil {
					return -1, -1, err
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

			if len(finalRecord) == 2000 {
				err = createFileAndWriteData(tempOffset, overwrite, finalRecord, manifestFile)
				if err != nil {
					log.Printf("marshal records err: %v\n", err.Error())
					return -1, -1, errors.New("unable to marshal records while compaction")
				}
				log.Println("marshalled final records")

				tempOffset = tempOffset + 1
				finalRecord = make([]Record, 0)
				overwrite = false
			}
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

	if len(finalRecord) > 0 {
		err = createFileAndWriteData(tempOffset, overwrite, finalRecord, manifestFile)
		if err != nil {
			log.Printf("marshal records err: %v\n", err.Error())
			return -1, -1, errors.New("unable to marshal records while compaction")
		}
		log.Println("marshalled final records")
	}
	// time.Sleep(1 * time.Minute)

	// cleanup stale files
	for _, fileName := range files {
		var file = file.NewFile(fileName, ssTableRecordsDirPath)
		if err = file.Remove(); err != nil {
			log.Println("unable to records of the file with err: ", err.Error())
			return -1, -1, errors.New("unable to read records")
		}
	}
	log.Println("cleanup the stale files")
	log.Println("optimized_file_offset: ", (tempOffset + 1))
	// time.Sleep(2 * time.Minute)

	return tempOffset + 1, lastOffset, nil
}
