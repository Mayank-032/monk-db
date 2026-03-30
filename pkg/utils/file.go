package utils

import (
	"errors"
	"fmt"
	"log"
	"monk-db/pkg/constants"
	"os"
	"strings"
)

type OperationType string

// Create Operations
var (
	DEFAULT OperationType = "DEFAULT"
	CREATE  OperationType = "CREATE"
	TRUNC   OperationType = "TRUNC"
)

// Write Operations
var (
	WRITEONLY OperationType = "WRITE-ONLY"
	APPEND    OperationType = "APPEND"
)

type File struct {
	name             string
	path             string
	filePathWithName string
	file             *os.File
}

func NewFile(name, path string) *File {
	return &File{
		name:             name,
		path:             path,
		filePathWithName: fmt.Sprintf("%v/%v", path, name),
	}
}

func (f *File) Get() error {
	if f == nil {
		log.Println("invalid file operation")
		return nil
	}

	file, err := os.Open(f.filePathWithName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Println("unable to open file: ", err.Error())
		return err
	}

	if file == nil {
		log.Println("file does not exist")
		return nil
	}

	f.file = file

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func (f *File) GetName() string {
	if f == nil {
		log.Println("invalid file operation")
		return constants.EMPTYSTRING
	}

	return f.name
}

func (f *File) GetPath() string {
	if f == nil {
		log.Println("invalid file operation")
		return constants.EMPTYSTRING
	}

	return f.path
}

func (f *File) GetFileFullPath() string {
	if f == nil {
		log.Println("invalid file operation")
		return constants.EMPTYSTRING
	}

	return f.filePathWithName
}

func (f *File) Read() ([]byte, error) {
	var err error

	b, err := os.ReadFile(f.filePathWithName)
	if err != nil && !os.IsNotExist(err) {
		log.Println("unable to open file in read-only mode: ", err.Error())
		return nil, err
	}

	return b, nil
}

func (f *File) Create(op OperationType, isSync bool) error {
	if f == nil {
		return errors.New("invalid file")
	}

	var err error
	switch op {
	case CREATE:
		fileInfo, err := os.Stat(f.filePathWithName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Println("unable to get stat of the file: ", err.Error())
			return err
		}

		if fileInfo != nil {
			return nil
		}

		f.file, err = os.OpenFile(f.filePathWithName, os.O_CREATE, constants.FILEPERMISSION)
	case TRUNC:
		f.file, err = os.OpenFile(f.filePathWithName, os.O_TRUNC, constants.FILEPERMISSION)
	default:
		fileInfo, err := os.Stat(f.filePathWithName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Println("unable to get stat of the file: ", err.Error())
			return err
		}

		if fileInfo != nil {
			return nil
		}
		f.file, err = os.Create(f.filePathWithName)
	}

	if err != nil {
		log.Println("unable to create file: ", err.Error())
		return err
	}

	if isSync {
		err = f.file.Sync()
		if err != nil {
			log.Println("unable to sync file: ", err.Error())
			return err
		}
	}

	return nil
}

func (f *File) Write(payload []byte, op OperationType, isSync bool) error {
	if f == nil {
		return errors.New("invalid file")
	}

	var err error

	switch op {
	case APPEND:
		f.file, err = os.OpenFile(f.filePathWithName, os.O_APPEND|os.O_WRONLY, constants.FILEPERMISSION)
		if err != nil {
			log.Println("unable to open file in append mode: ", err.Error())
			return err
		}

		_, err = f.file.Write(payload)
	case WRITEONLY:
		f.file, err = os.OpenFile(f.filePathWithName, os.O_WRONLY, constants.FILEPERMISSION)
		_, err = f.file.Write(payload)
	default:
		_, err = f.file.Write(payload)
	}

	if err != nil {
		log.Println("unable to write file: ", err.Error())
		return err
	}

	if isSync {
		err = f.file.Sync()
		if err != nil {
			log.Println("unable to sync file: ", err.Error())
			return err
		}
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
	err = f.Write(combinedData, DEFAULT, false)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Append(newData []byte) error {
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
	combinedData = append(existingData, combinedData...)

	// 3. Overwrite the file with the combined content
	err = f.Write(combinedData, DEFAULT, false)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) AppendWithTmpFile(newData []byte) error {
	if f == nil {
		return errors.New("invalid file")
	}

	// 1. Read the existing content (if file exists)
	existingData, err := f.Read()
	if err != nil {
		return err
	}

	// 2. Combine: [New Data] + [Existing Data]
	var combinedData = append(newData, []byte("\n")...)
	combinedData = append(existingData, combinedData...)

	// 3. Create a temporary file
	var currFilenameArr = strings.Split(f.name, ".")
	var tmpFileName = fmt.Sprintf("%v.%v", fmt.Sprint(currFilenameArr[0]+"_"+"tmp"), currFilenameArr[1])
	var tmpFile = NewFile(tmpFileName, f.path)
	err = tmpFile.Create(CREATE, false)
	if err != nil {
		return err
	}

	// 4. Write the temporary file
	err = tmpFile.Write(combinedData, WRITEONLY, true)
	if err != nil {
		return err
	}

	err = tmpFile.Close()
	if err != nil {
		return err
	}

	// 5. Rename the temporary file with old file-name to overwrite the data
	err = tmpFile.Rename(f.name)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Rename(name string) error {
	f.name = name

	var renamePath = fmt.Sprintf("%v/%v", f.path, f.name)

	err := os.Rename(f.filePathWithName, renamePath)
	if err != nil {
		log.Println("unable to rename file: ", err.Error())
		return err
	}

	return nil
}

func (f *File) Close() error {
	err := f.file.Close()
	if err != nil {
		log.Println("unable to close file: ", err.Error())
		return err
	}

	return nil
}
