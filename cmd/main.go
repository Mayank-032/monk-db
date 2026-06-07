package main

import (
	"log"
	"monk-db/boot"
	"monk-db/internal/constants"
	"monk-db/internal/io/file"
	"os"
	"strings"
	"time"
)

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

func main() {
	var store, err = boot.Initialize()
	if err != nil {
		log.Println("unable to bootup: ", err.Error())
		os.Exit(1)
		return
	}

	buffer, err := file.ParseFile("./put-delete.txt")
	if err != nil {
		log.Println("unable to parse file; err: ", err.Error())
		os.Exit(1)
		return
	}

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
			var totalTimeTakenInMs = time.Since(compactionStartTime).Milliseconds()
			totalCompactTimeInMs = totalCompactTimeInMs + int(totalTimeTakenInMs)
			totalCompactOperations = totalCompactOperations + 1

			if err != nil {
				log.Printf("OPTIMIZE OPERATION failed with err: %v\n", err.Error())
				os.Exit(1)
			}

			// store.UpdateOffset(newOffset, lastOffset)

			// reset counter
			totalUpdateOperations = 0
		}
	}
	totalTimeTakenInMs := time.Since(startTime).Milliseconds()

	var avgTimeTakenForPutOperationsInMs = float64(totalPutTimeInMs) / float64(totalPutOperations)
	log.Printf("Avg. time taken for all PUT operations: %v ms\n", avgTimeTakenForPutOperationsInMs)

	var avgTimeTakenForGetOperationsInMs = float64(totalGetTimeInMs) / float64(totalGetOperations)
	log.Printf("Avg. time taken for all GET operations: %v ms\n", avgTimeTakenForGetOperationsInMs)

	var avgTimeTakenForDeleteOperationsInMs = float64(totalDeleteTimeInMs) / float64(totalDeleteOperations)
	log.Printf("Avg. time taken for all DELETE operations: %v ms\n", avgTimeTakenForDeleteOperationsInMs)

	var avgTimeTakenForCompactOperationsInMs = float64(totalCompactTimeInMs) / float64(totalCompactOperations)
	log.Printf("Avg. time take for all COMPACT operations: %v ms\n", avgTimeTakenForCompactOperationsInMs)

	log.Printf("total time taken for all 30k combined operations: %v ms\n", totalTimeTakenInMs)
	os.Exit(0)
}
