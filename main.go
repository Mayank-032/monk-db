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
	var size = 2000
	store := storage.InitStore(size)
	if store == nil {
		log.Println("unable to init store")
		os.Exit(1)
		return
	}

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)

	manifestFilepath := filepath.Join(baseDir, "./pkg/sstable/manifest.txt")
	sstableRecordsDirPath := filepath.Join(baseDir, "./pkg/sstable/records")
	if err := sstable.SetManifestLogfilePathAndCreate("manifest.txt", manifestFilepath); err != nil {
		os.Exit(1)
		return
	}
	if err := sstable.SetSSTableRecordsDirPathAndCreate(sstableRecordsDirPath); err != nil {
		os.Exit(1)
		return
	}

	buffer, err := utils.ParseFile("./put.txt")
	if err != nil {
		log.Println("unable to parse file; err: ", err.Error())
		os.Exit(1)
		return
	}

	startTime := time.Now()

	for index, block := range buffer {
		if strings.EqualFold(block[0], constants.PUT) {
			key := block[1]
			val := block[2]

			success, err := store.Put(key, val)
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

		/*
			// Commenting it out for testing write functionality only
			if strings.EqualFold(block[0], constants.GET) {
				key := block[1]
				expectedVal := block[2]

				res, err := store.Get(key)
				if err != nil && err.Error() != constants.ERRNOTFOUND {
					log.Printf("GET OPERATION with index: %v, for key: %v, failed with err: %v\n", index, key, err.Error())
					os.Exit(1)
					return
				}

				if res != expectedVal {
					log.Printf("INVALID GET OPERATION with index: %v, for key: %v\n", index, key)
					os.Exit(1)
				}

				log.Printf("GET OPERATION SUCCESS with index: %v, for key: %v\n", index, key)
				continue
			}
		*/

	}

	totalTimeTakenInMs := time.Since(startTime).Milliseconds()
	log.Printf("total time taken: %vms\n", totalTimeTakenInMs)

	os.Exit(0)
}
