package file

import (
	"errors"
	"fmt"
	"log"
	"monk-db/internal/constants"
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
		return fmt.Errorf("invalid file operation")
	}

	file, err := os.Open(f.filePathWithName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("unable to open file with err: %w", err)
	}

	if file == nil {
		return fmt.Errorf("file does not exist")
	}

	f.file = file

	err = f.Close()
	if err != nil {
		return fmt.Errorf("unable to close file")
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
		return nil, fmt.Errorf("unable to open file in read-only mode: %w", err)
	}

	return b, nil
}

func (f *File) Create(op OperationType, isSync bool) error {
	if f == nil {
		return errors.New("invalid file")
	}

	var (
		fileInfo os.FileInfo
		err      error
	)

	switch op {
	case CREATE:
		fileInfo, err = os.Stat(f.filePathWithName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unable to get stat of the file: %w", err)
		}

		if fileInfo != nil {
			return nil
		}

		f.file, err = os.OpenFile(f.filePathWithName, os.O_CREATE|os.O_RDWR, constants.FILEPERMISSION)
	case TRUNC:
		f.file, err = os.OpenFile(f.filePathWithName, os.O_TRUNC|os.O_RDWR, constants.FILEPERMISSION)
	default:
		fileInfo, err = os.Stat(f.filePathWithName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unable to get stat of the file: %w", err)
		}

		if fileInfo != nil {
			return nil
		}
		f.file, err = os.Create(f.filePathWithName)
	}

	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}

	if isSync {
		err = f.file.Sync()
		if err != nil {
			return fmt.Errorf("unable to sync file: %w", err)
		}
	}

	return nil
}

func (f *File) Write(payload []byte, op OperationType, isSync bool) error {
	if f == nil {
		return fmt.Errorf("invalid file")
	}

	var err error

	switch op {
	case APPEND:
		f.file, err = os.OpenFile(f.filePathWithName, os.O_APPEND|os.O_WRONLY, constants.FILEPERMISSION)
		if err != nil {
			log.Println()
			return fmt.Errorf("unable to open file in append mode: %w", err)
		}

		_, err = f.file.Write(payload)
	case WRITEONLY:
		f.file, err = os.OpenFile(f.filePathWithName, os.O_WRONLY, constants.FILEPERMISSION)
		_, err = f.file.Write(payload)
	default:
		_, err = f.file.Write(payload)
	}

	if err != nil {
		return fmt.Errorf("unable to write file: %w", err)
	}

	if isSync {
		err = f.file.Sync()
		if err != nil {
			return fmt.Errorf("unable to sync file: %w", err)
		}
	}

	return nil
}

func (f *File) Prepend(newData []byte) error {
	if f == nil {
		return fmt.Errorf("invalid file")
	}

	// 1. Read the existing content (if file exists)
	existingData, err := f.Read()
	if err != nil {
		return fmt.Errorf("unable to read file content: %w", err)
	}

	// 2. Combine: [New Data] + [Existing Data]
	combinedData := append(newData, []byte("\n")...)
	combinedData = append(combinedData, existingData...)

	// 3. Overwrite the file with the combined content
	err = f.Write(combinedData, DEFAULT, false)
	if err != nil {
		return fmt.Errorf("unable to write file content: %w", err)
	}

	return nil
}

func (f *File) Append(newData []byte) error {
	if f == nil {
		return fmt.Errorf("invalid file")
	}

	// 1. Read the existing content (if file exists)
	existingData, err := f.Read()
	if err != nil {
		return fmt.Errorf("unable to read file content: %w", err)
	}

	// 2. Combine: [New Data] + [Existing Data]
	combinedData := append(newData, []byte("\n")...)
	combinedData = append(existingData, combinedData...)

	// 3. Overwrite the file with the combined content
	err = f.Write(combinedData, DEFAULT, false)
	if err != nil {
		return fmt.Errorf("unable to write file content: %w", err)
	}

	return nil
}

func (f *File) AppendWithTmpFile(newData []byte, overwrite bool) error {
	if f == nil {
		return fmt.Errorf("invalid file")
	}

	var combinedData = append(newData, []byte("\n")...)

	if !overwrite {
		// 1. Read the existing content (if file exists)
		existingData, err := f.Read()
		if err != nil {
			return fmt.Errorf("unable to read file content: %w", err)
		}

		// 2. Combine: [New Data] + [Existing Data]
		combinedData = append(existingData, combinedData...)
	}

	// 3. Create a temporary file
	var currFilenameArr = strings.Split(f.name, ".")
	var tmpFileName = fmt.Sprintf("%v.%v", fmt.Sprint(currFilenameArr[0]+"_"+"tmp"), currFilenameArr[1])
	var tmpFile = NewFile(tmpFileName, f.path)
	err := tmpFile.Create(CREATE, false)
	if err != nil {
		return fmt.Errorf("unable to create file: %w", err)
	}

	// 4. Write the temporary file
	err = tmpFile.Write(combinedData, WRITEONLY, true)
	if err != nil {
		return fmt.Errorf("unable to write file content: %w", err)
	}

	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("unable to close file: %w", err)
	}

	// 5. Rename the temporary file with old file-name to overwrite the data
	err = tmpFile.Rename(f.name)
	if err != nil {
		return fmt.Errorf("unable to rename file: %w", err)
	}

	return nil
}

func (f *File) Rename(name string) error {
	f.name = name

	var renamePath = fmt.Sprintf("%v/%v", f.path, f.name)

	err := os.Rename(f.filePathWithName, renamePath)
	if err != nil {
		log.Println()
		return fmt.Errorf("unable to rename file: %w", err)
	}

	return nil
}

func (f *File) Reset(isSync bool) error {
	if err := f.Create(TRUNC, isSync); err != nil {
		return fmt.Errorf("unable to reset file: %w", err)
	}

	return nil
}

func (f *File) Close() error {
	err := f.file.Close()
	if err != nil {
		return fmt.Errorf("unable to close file: %w", err)
	}

	return nil
}

func (f *File) Remove() error {
	err := os.Remove(f.GetFileFullPath())
	if err != nil {
		return fmt.Errorf("unable to remove file: %w", err)
	}

	return nil
}
