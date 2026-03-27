package utils

import (
	"encoding/json"
	"errors"
	"log"
	"monk-db/pkg/constants"
	"os"
)

type File struct {
	name string
	path string
	file *os.File
}

func NewFile(name, path string) *File {
	return &File{
		name: name,
		path: path,
	}
}

func (f *File) GetName() string {
	if f == nil {
		return constants.EMPTYSTRING
	}

	return f.name
}

func (f *File) Create() error {
	fileInfo, err := os.Stat(f.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if fileInfo != nil {
		return nil
	}

	file, err := os.Create(f.path)
	if err != nil {
		return err
	}

	f.file = file
	return nil
}

func (f *File) Write(content any) error {
	recordBytes, err := json.MarshalIndent(content, constants.EMPTYSTRING, constants.MARSHALSPACING)
	if err != nil {
		log.Printf("unable to marshal records with err: %v\n", err.Error())
		return err
	}

	err = os.WriteFile(f.path, recordBytes, constants.FILEPERMISSION)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Prepend(newData []byte) error {
	// 1. Read the existing content (if file exists)
	existingData, err := os.ReadFile(f.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// 2. Combine: [New Data] + [Existing Data]
	combinedData := append(newData, []byte("\n")...)
	combinedData = append(combinedData, existingData...)

	// 3. Overwrite the file with the combined content
	err = os.WriteFile(f.path, combinedData, 0644)
	if err != nil {
		return err
	}

	return nil
}
