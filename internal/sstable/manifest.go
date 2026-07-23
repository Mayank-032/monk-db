package sstable

import (
	"errors"
	"log"
	"monk-db/internal/io/file"
)

func (sst *ssTable) SetManifestFile(name, path string) error {
	var manifestFile = file.NewFile(name, path)
	err := manifestFile.Create(file.CREATE, true)
	if err != nil {
		log.Printf("create manifest file err: %v\n", err.Error())
		return errors.New("unable to create manifest file")
	}

	sst.manifestFile = manifestFile
	return nil
}

func (sst *ssTable) GetManifestFile() *file.File {
	return sst.manifestFile
}
