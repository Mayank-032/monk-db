package utils

import (
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

func (f *File) Get() error {
	if f == nil {
		log.Println("invalid file operation")
		return nil
	}

	file, err := os.Open(f.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Println("unable to open file: ", err.Error())
		return err
	}

	if file == nil {
		log.Println("file does not exist")
		return nil
	}

	f.file = file
	return nil
}

func (f *File) GetName() string {
	if f == nil {
		log.Println("invalid file operation")
		return constants.EMPTYSTRING
	}

	return f.name
}

func (f *File) Read() ([]byte, error) {
	b, err := os.ReadFile(f.path)
	if err != nil && !os.IsNotExist(err) {
		log.Println("unable to read file: ", err.Error())
		return nil, err
	}

	return b, nil
}

func (f *File) Create() error {
	if f == nil {
		return errors.New("invalid file")
	}

	fileInfo, err := os.Stat(f.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Println("unable to get stat of the file: ", err.Error())
		return err
	}

	if fileInfo != nil {
		return nil
	}

	file, err := os.Create(f.path)
	if err != nil {
		log.Println("unable to create file: ", err.Error())
		return err
	}

	f.file = file
	return nil
}

func (f *File) Write(payload []byte) error {
	if f == nil {
		return errors.New("invalid file")
	}

	err := os.WriteFile(f.path, payload, constants.FILEPERMISSION)
	if err != nil {
		log.Println("unable to write file: ", err.Error())
		return err
	}

	return nil
}

func (f *File) Prepend(newData []byte) error {
	if f == nil {
		return errors.New("invalid file")
	}

	// 1. Read the existing content (if file exists)
	existingData, err := f.Read()
	if err != nil {
		return err
	}

	// 2. Combine: [New Data] + [Existing Data]
	combinedData := append(newData, []byte("\n")...)
	combinedData = append(combinedData, existingData...)

	// 3. Overwrite the file with the combined content
	err = f.Write(combinedData)
	if err != nil {
		return err
	}

	return nil
}
