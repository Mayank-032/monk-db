package sstable

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
)

type ssTable struct {
	count int
}

func NewSSTable(count int) *ssTable {
	return &ssTable{
		count: count,
	}
}

// it will convert to respective data structure and flush it to the log file
func (sst *ssTable) Flush(data map[string]string) error {
	if sst == nil {
		log.Println("sstable is not initialized")
		return errors.New("sstable is not initialized")
	}

	var sstableRecords = make([]Record, 0, len(data))
	for key, val := range data {
		sstableRecords = append(sstableRecords, Record{
			Key:   key,
			Value: val,
		})
	}

	file, err := os.Create(fmt.Sprintf("./pkg/sstable/records/sst-%v.json", sst.count))
	if err != nil {
		log.Printf("unable to create a file with err: %v\n", err.Error())
		return err
	}

	recordBytes, err := json.MarshalIndent(sstableRecords, "", "    ")
	if err != nil {
		log.Printf("unable to marshal records with err: %v\n", err.Error())
		return err
	}

	_, err = file.Write(recordBytes)
	if err != nil {
		log.Printf("unable to write data to file with err: %v\n", err.Error())
		return err
	}

	err = prependToFile("./pkg/sstable/manifest.txt", []byte(file.Name()))
	if err != nil {
		log.Printf("unable to write data to manifest file with err: %v\n", err.Error())
		return err
	}

	return nil
}

func prependToFile(filename string, newData []byte) error {
	// 1. Read the existing content (if file exists)
	existingData, err := os.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// 2. Combine: [New Data] + [Existing Data]
	combinedData := append(newData, []byte("\n")...)
	combinedData = append(combinedData, existingData...)

	// 3. Overwrite the file with the combined content
	return os.WriteFile(filename, combinedData, 0644)
}
