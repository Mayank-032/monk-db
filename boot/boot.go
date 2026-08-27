package boot

import (
	"fmt"
	"log"
	"monk-db/internal/constants"
	cache "monk-db/internal/ds/lru_cache"
	"monk-db/internal/io/file"
	"monk-db/internal/models"
	"monk-db/internal/sstable"
	"monk-db/internal/storage"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Initialize() (*storage.Store, error) {
	// init cache
	var storageCache = cache.NewLRUCache[[]models.Record](10)
	log.Println("cache init success")

	diskStorage, err := sstable.NewSSTable(storageCache, constants.MANIFEST_FILENAME, constants.MANIFEST_FILEPATH)
	if err != nil {
		return nil, err
	}

	var walFilepath = filepath.Join(GetBaseDir(true), constants.WAL_FILEPATH)

	var initStoreStartTime = time.Now()
	store, err := storage.InitStore(constants.MEMTABLE_SIZE, constants.WAL_FILENAME, walFilepath, diskStorage)
	var initStoreTimeDuration = time.Since(initStoreStartTime).Milliseconds()

	if err != nil || store == nil {
		return nil, err
	}

	err = handleDanglingFileAndGetOffset(diskStorage.GetManifestFile(), constants.RECORDS_DIRPATH)
	if err != nil {
		return nil, err
	}

	log.Printf("memtable init success, total initialization time taken: %v ms\n", initStoreTimeDuration)

	return store, nil
}

func GetBaseDir(isRootDir bool) string {
	if isRootDir {
		rootDir, err := os.Getwd()
		if err != nil {
			return "."
		}

		return rootDir
	}

	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func handleDanglingFileAndGetOffset(manifestFile *file.File, recordDir string) error {
	fileContent, err := manifestFile.Read()
	if err != nil {
		log.Println("unable to read manifest file")
		return err
	}

	var fileContentStr = string(fileContent)

	var fileContentPart = strings.Split(fileContentStr, "\n")
	if len(fileContentPart) <= 1 {
		return nil
	}

	files, err := os.ReadDir(recordDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		var isFileFound bool
		for _, part := range fileContentPart {
			if strings.EqualFold(f.Name(), part) {
				isFileFound = true
				break
			}
		}

		if !isFileFound {
			var newFile = file.NewFile(f.Name(), recordDir)
			err = newFile.Remove()
			if err != nil {
				log.Println("unable to remove invalid file on disk")
				return err
			}
			fmt.Println("file removed success")
		}
	}

	return nil
}
