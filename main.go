package main

import (
	"log"
	"math"
	"monk-db/pkg/constants"
	"monk-db/pkg/sstable"
	"monk-db/pkg/storage"
	"monk-db/pkg/utils"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	utils.NewLRUCache(3)
	log.Println("cache init success")

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)

	manifestFilename := "manifest.txt"
	manifestFilepath := filepath.Join(baseDir, "./pkg/sstable")
	sstableRecordsDirPath := filepath.Join(baseDir, "./pkg/sstable/records")
	if err := sstable.SetManifestLogfilePathAndCreate(manifestFilename, "./pkg/sstable"); err != nil {
		os.Exit(1)
		return
	}
	log.Println("records dir init success")

	if err := sstable.SetSSTableRecordsDirPathAndCreate(sstableRecordsDirPath); err != nil {
		os.Exit(1)
		return
	}
	log.Println("manifest file init success")

	var offset, err = getOffsetFromManifestFile(manifestFilename, manifestFilepath)
	if err != nil {
		os.Exit(1)
		return
	}

	var walFilename = "wal.db"
	var walFilepath = filepath.Join(baseDir, "./pkg/storage")
	var size = 2000

	var initStoreStartTime = time.Now()
	store, err := storage.InitStore(size, offset, walFilename, walFilepath)
	var initStoreTimeDuration = time.Since(initStoreStartTime).Milliseconds()

	if err != nil || store == nil {
		log.Println("unable to init store")
		os.Exit(1)
		return
	}
	log.Println("memtable init success, total initialization time taken (in ms): ", initStoreTimeDuration)

	buffer, err := utils.ParseFile("./put.txt")
	if err != nil {
		log.Println("unable to parse file; err: ", err.Error())
		os.Exit(1)
		return
	}

	startTime := time.Now()

	var (
		// PUT OPERATIONS
		totalPutTimeInMs   = 0
		totalPutOperations = 0

		// GET OPERATIONS
		totalGetTimeInMs   = 0
		totalGetOperations = 0
	)
	for index, block := range buffer {
		if strings.EqualFold(block[0], constants.PUT) {
			key := block[1]
			val := block[2]

			putStartTime := time.Now()
			success, err := store.Put(key, val)
			totalTimeTakenInMs := time.Since(putStartTime).Milliseconds()
			totalPutTimeInMs = totalPutTimeInMs + int(totalTimeTakenInMs)
			totalPutOperations = totalPutOperations + 1

			if err != nil {
				log.Printf("PUT OPERATION with index: %v, for key: %v, failed with err: %v\n", index, key, err.Error())
				os.Exit(1)
				return
			}
			if !success {
				log.Printf("INVALID PUT OPERATION with index: %v, for key: %v\n", index, key)
				os.Exit(1)
				return
			}

			// log.Printf("PUT OPERATION SUCCESS with index: %v, for key: %v\n", index, key)
			continue
		}

		if strings.EqualFold(block[0], constants.GET) {
			key := block[1]
			expectedVal := block[2]

			getStartTime := time.Now()
			res, err := store.Get(key)
			totalTimeTakenInMs := time.Since(getStartTime).Milliseconds()
			totalGetTimeInMs = totalGetTimeInMs + int(totalTimeTakenInMs)
			totalGetOperations = totalGetOperations + 1

			if err != nil && err.Error() != constants.ERRNOTFOUND {
				log.Printf("GET OPERATION with index: %v, for key: %v, failed with err: %v\n", index, key, err.Error())
				os.Exit(1)
				return
			}

			if res != expectedVal {
				log.Printf("INVALID GET OPERATION with index: %v, for key: %v\n", index, key)
				os.Exit(1)
			}

			// log.Printf("GET OPERATION SUCCESS with index: %v, for key: %v\n", index, key)
			continue
		}

	}

	totalTimeTakenInMs := time.Since(startTime).Milliseconds()

	var avgTimeTakenForPutOperationsInMs = math.Round(float64(totalPutTimeInMs) / float64(totalPutOperations))
	var avgTimeTakenForGetOperationsInMs = math.Round(float64(totalGetTimeInMs) / float64(totalGetOperations))

	log.Printf("Avg. time taken for all PUT operations: %v ms\n", avgTimeTakenForPutOperationsInMs)
	log.Printf("Avg. time taken for all GET operations: %v ms\n", avgTimeTakenForGetOperationsInMs)
	log.Printf("total time taken for all 30k combined operations: %v ms\n", totalTimeTakenInMs)

	os.Exit(0)
}

func getOffsetFromManifestFile(manifestFileName, manifestFilePath string) (int, error) {
	var file = utils.NewFile(manifestFileName, manifestFilePath)
	fileContent, err := file.Read()
	if err != nil {
		return -1, err
	}

	var fileContentStr = string(fileContent)

	var fileContentPart = strings.Split(fileContentStr, "\n")
	if len(fileContentPart) <= 1 {
		return 0, nil
	}

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
