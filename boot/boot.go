package boot

import (
	"fmt"
	"log"
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

	diskStorage, err := sstable.NewSSTable(RECORDSDIRPATH, storageCache, MANIFESTFILENAME, MANIFESTFILEPATH)
	if err != nil {
		log.Fatal("unable to init disk based storage")
		return nil, err
	}

	var walFilepath = filepath.Join(GetBaseDir(), WALFILEPATH)

	var initStoreStartTime = time.Now()
	store, err := storage.InitStore(MEMTABLESIZE, WALFILENAME, walFilepath, diskStorage)
	var initStoreTimeDuration = time.Since(initStoreStartTime).Milliseconds()

	if err != nil || store == nil {
		log.Fatal("unable to init store")
		return nil, err
	}

	err = handleDanglingFileAndGetOffset(diskStorage.GetManifestFile(), diskStorage.GetRecordsDirPath())
	if err != nil {
		log.Fatal("unable to clean dangling files: ", err)
	}

	log.Printf("memtable init success, total initialization time taken: %v ms\n", initStoreTimeDuration)

	return store, nil
}

func GetBaseDir() string {
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)

	return baseDir
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
