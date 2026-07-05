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
	"monk-db/internal/models"
	"os"
	"sort"
)

type ssTable struct {
	offset         int
	lastOffset     int
	cache          *cache.Cache[[]models.Record]
	manifestFile   *file.File
	recordsDirPath string
}

func NewSSTable(
	recordsDirPath string,
	cache *cache.Cache[[]models.Record],
	manifestFilename, manifestFilepath string,
) (*ssTable, error) {
	err := os.Mkdir(recordsDirPath, constants.DIRPERMISSION)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Println("create records dir")
		return nil, err
	}

	var sst = &ssTable{
		cache:          cache,
		recordsDirPath: recordsDirPath,
	}

	err = sst.SetManifestFile(manifestFilename, manifestFilepath)
	if err != nil {
		log.Println("unable to set manifest file")
		return nil, err
	}

	return sst, nil
}

// it will convert to respective data structure and flush it to the log file
func (sst *ssTable) Flush(sstableRecords []models.Record) error {
	// check if sstable is initialized or not
	if sst == nil {
		return fmt.Errorf("sstable is not initialized")
	}

	// sort the content need to be written in sstable
	sort.SliceStable(sstableRecords, func(i, j int) bool {
		return sstableRecords[i].Key < sstableRecords[j].Key
	})

	// marshal the json content
	recordBytes, err := json.MarshalIndent(sstableRecords, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		return fmt.Errorf("unable to flush records, with marshalling error: %w", err)
	}

	// create a file with new offset
	var newFile = file.NewFile(fmt.Sprintf("sst-%v.json", sst.offset+1), sst.recordsDirPath)
	err = newFile.Create(file.DEFAULT, true)
	if err != nil {
		return fmt.Errorf("unable to flush records, with create file error: %w", err)
	}

	// flush the contents to new file
	err = newFile.Write(recordBytes, file.DEFAULT, true)
	if err != nil {
		return fmt.Errorf("unable to flush records, with write file error: %w", err)
	}

	// read the manifest file, TODO: Update the fetching of manifest file from central place
	// err = manifestFile.Get()
	// if err != nil {
	// 	return fmt.Errorf("unable to flush records, with get manifest file error: %w", err)
	// }

	// write the manifest file, TODO: Should be done via taking a lock once we maintain at central place
	err = sst.manifestFile.AppendWithTmpFile([]byte(newFile.GetName()), false)
	if err != nil {
		return fmt.Errorf("unable to flush records, with write manifest file error: %w", err)
	}

	// TODO: perform this operation under a lock
	sst.offset = sst.offset + 1

	return nil
}

func (sst *ssTable) Read(key string) (string, error) {
	// check if sstable is initialized or not
	if sst == nil {
		return constants.EMPTYSTRING, fmt.Errorf("sstable is not initialized")
	}

	var currOffset = sst.offset
	if currOffset == 0 {
		currOffset = 1
	}

	for currOffset >= sst.lastOffset {
		var file = file.NewFile(fmt.Sprintf("sst-%d.json", currOffset), sst.recordsDirPath)
		records, err := sst._readSSTable(file)
		if err != nil {
			return constants.EMPTYSTRING, fmt.Errorf("unable to read sstable: %w", err)
		}

		// return the respective value or precise error
		for _, r := range records {
			if r.Key == key {
				if !r.IsDeleted {
					return r.Value, nil
				}

				return constants.EMPTYSTRING, errors.New(constants.ERRRESOURCEREMOVED)
			}
		}

		currOffset--
	}

	return constants.EMPTYSTRING, errors.New(constants.ERRNOTFOUND)
}

func (sst *ssTable) _readSSTable(file *file.File) ([]models.Record, error) {
	// get the content from the cache
	records, err := sst.cache.Get(file.GetName())
	if err != nil && err.Error() != constants.ERRNOTFOUND {
		return nil, fmt.Errorf("unable to read content with cache err: %w", err)
	}

	// if the data is present in cache, serve it from there
	if len(records) > 0 {
		return records, nil
	}

	// read the file content if not present in the cache
	records, err = readRecordsFileData(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read content with err: %w", err)
	}

	// since the content is new, keep it in the cache as frequently used
	sst.cache.PUT(file.GetName(), records)

	return records, nil
}

// This optimizes the sstable storage and returns the latest offset
func (sst *ssTable) Optimize() error {
	/* 1) Let's start with fetching all the data in-memory at once */

	// Read Manifest File to get the list of existing files
	var manifestFile = sst.manifestFile
	files, err := readManifestFileData(manifestFile)
	if err != nil {
		return fmt.Errorf("unable to compact, with read manifest file err %w", err)
	}
	log.Println("read all files")

	var list = make([][]models.Record, 0)
	for _, fileName := range files {
		var file = file.NewFile(fileName, sst.recordsDirPath)
		records, err := readRecordsFileData(file)
		if err != nil {
			log.Println("unable to records of the file with err: ", err.Error())
			return fmt.Errorf("unable to compact, with read records err %w", err)
		}

		list = append(list, records)
	}
	log.Println("unmarshalled all files")

	// 2) calculate the new offset
	lastOffset, err := calculateOffset(files[len(files)-1])
	if err != nil {
		return fmt.Errorf("unable to compact, with calculate offset err %w", err)
	}
	log.Println("calculate new offset")

	var tempOffset = lastOffset

	// 3) Merge Records from all the files
	tempOffset, err = sst._mergeAndWriteRecords(tempOffset, list, manifestFile)
	if err != nil {
		return fmt.Errorf("unable to compact, with merge records err: %w", err)
	}

	// time.Sleep(1 * time.Minute)

	// 4) cleanup stale files
	for _, fileName := range files {
		var file = file.NewFile(fileName, sst.recordsDirPath)
		if err = file.Remove(); err != nil {
			return fmt.Errorf("unable to compact, while cleanup stale files err %w", err)
		}
	}
	log.Println("cleanup the stale files")
	log.Println("optimized_file_new_offset: ", (tempOffset))
	log.Println("optimized_file_last_offset: ", (lastOffset))

	// time.Sleep(2 * time.Minute)

	sst.offset = tempOffset
	sst.lastOffset = lastOffset

	return nil
}

func (sst *ssTable) _mergeAndWriteRecords(tempOffset int, list [][]models.Record, manifestFile *file.File) (int, error) {
	/* 1) Create a new heap */
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

	/* 2) Push first records from each list in a new heap */
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

	/* 3) Iterate over the heap and for each 2000 merged records, create a file in records dir and update the manifest file */
	var finalRecord = make([]models.Record, 0)
	var overwrite bool = true
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

			if len(finalRecord) == 2000 {
				err = createFileAndWriteData(tempOffset, overwrite, finalRecord, manifestFile, sst.recordsDirPath)
				if err != nil {
					return -1, fmt.Errorf("unable to compact into multiple files %w", err)
				}

				tempOffset = tempOffset + 1
				finalRecord = make([]models.Record, 0)
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

	// 4) If any unfinished records left, create create the file in records dir and update manifest
	if len(finalRecord) > 0 {
		err := createFileAndWriteData(tempOffset, overwrite, finalRecord, manifestFile, sst.recordsDirPath)
		if err != nil {
			return -1, fmt.Errorf("unable to compact, with marshal err %w", err)
		}
		log.Println("marshalled final records")
		tempOffset = tempOffset + 1
	}

	return tempOffset, nil
}
