package main

import (
	"log"
	"monk-db/pkg/constants"
	"monk-db/pkg/sstable"
	"monk-db/pkg/storage"
	"monk-db/pkg/utils"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	utils.NewLRUCache(10)
	log.Println("cache init success")

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)

	manifestFilename := "manifest.txt"
	// manifestFilepath := filepath.Join(baseDir, "./pkg/sstable")
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

	var walFilename = "wal.db"
	var walFilepath = filepath.Join(baseDir, "./pkg/storage")
	var size = 2000

	var initStoreStartTime = time.Now()
	store, err := storage.InitStore(size, walFilename, walFilepath)
	var initStoreTimeDuration = time.Since(initStoreStartTime).Milliseconds()

	if err != nil || store == nil {
		log.Println("unable to init store")
		os.Exit(1)
		return
	}
	log.Printf("memtable init success, total initialization time taken: %v ms\n", initStoreTimeDuration)

	buffer, err := utils.ParseFile("./put-delete.txt")
	if err != nil {
		log.Println("unable to parse file; err: ", err.Error())
		os.Exit(1)
		return
	}

	var (
		// PUT OPERATIONS
		totalPutTimeInMs   = 0
		totalPutOperations = 0

		// GET OPERATIONS
		totalGetTimeInMs   = 0
		totalGetOperations = 0

		// DELETE OPERATIONS
		totalDeleteTimeInMs   = 0
		totalDeleteOperations = 0

		totalUpdateOperations = 0

		// Compact Operation
		totalCompactTimeInMs   = 0
		totalCompactOperations = 0
	)

	startTime := time.Now()
	for index, block := range buffer {
		switch strings.ToUpper(block[0]) {
		case constants.PUT:
			key := block[1]
			val := block[2]

			putStartTime := time.Now()
			success, err := store.Put(key, val, false)
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

			totalUpdateOperations = totalUpdateOperations + 1

			// log.Printf("PUT OPERATION SUCCESS with index: %v, for key: %v\n", index, key)
		case constants.GET:
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
				log.Printf("actual_val: %v, expected_val: %v\n", res, expectedVal)
				log.Printf("INVALID GET OPERATION with index: %v, for key: %v\n", index, key)
				os.Exit(1)
				return
			}

			// log.Printf("GET OPERATION SUCCESS with index: %v, for key: %v\n", index, key)
		case constants.DELETE:
			key := block[1]

			deleteStartTime := time.Now()
			success, err := store.Delete(key)
			totalTimeTakenInMs := time.Since(deleteStartTime).Milliseconds()
			totalDeleteTimeInMs = totalDeleteTimeInMs + int(totalTimeTakenInMs)
			totalDeleteOperations = totalDeleteOperations + 1

			if err != nil && err.Error() != constants.ERRNOTFOUND {
				log.Printf("DELETE OPERATION with index: %v, for key: %v, failed with err: %v\n", index, key, err.Error())
				os.Exit(1)
				return
			}
			if !success {
				log.Printf("INVALID DELETE OPERATION with index: %v, for key: %v\n", index, key)
				os.Exit(1)
				return
			}

			totalUpdateOperations = totalUpdateOperations + 1

			// log.Printf("DELETE OPERATION SUCCESS with index: %v, for key: %v\n", index, key)
		default:
			log.Println("invalid operation")
			os.Exit(1)
			return
		}

		// For every 10,000 requests completion, trigger compaction
		if totalUpdateOperations == 10000 {
			// trigger compaction
			var compactionStartTime = time.Now()
			sstable.Optimize()
			var totalTimeTakenInMs = time.Since(compactionStartTime).Milliseconds()
			totalCompactTimeInMs = totalCompactTimeInMs + int(totalTimeTakenInMs)
			totalCompactOperations = totalCompactOperations + 1

			// reset counter
			totalUpdateOperations = 0
		}
	}
	totalTimeTakenInMs := time.Since(startTime).Milliseconds()

	var avgTimeTakenForPutOperationsInMs = float64(totalPutTimeInMs) / float64(totalPutOperations)
	var avgTimeTakenForGetOperationsInMs = float64(totalGetTimeInMs) / float64(totalGetOperations)
	var avgTimeTakenForDeleteOperationsInMs = float64(totalDeleteTimeInMs) / float64(totalDeleteOperations)
	var avgTimeTakenForCompactOperationsInMs = float64(totalCompactTimeInMs) / float64(totalCompactOperations)

	log.Printf("Avg. time taken for all PUT operations: %v ms\n", avgTimeTakenForPutOperationsInMs)
	log.Printf("Avg. time taken for all GET operations: %v ms\n", avgTimeTakenForGetOperationsInMs)
	log.Printf("Avg. time taken for all DELETE operations: %v ms\n", avgTimeTakenForDeleteOperationsInMs)
	log.Printf("Avg. time take for all COMPACT operations: %v ms\n", avgTimeTakenForCompactOperationsInMs)

	log.Printf("total time taken for all 30k combined operations: %v ms\n", totalTimeTakenInMs)

	os.Exit(0)
}
