package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParseFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file with err %v", err.Error())
	}
	defer file.Close()

	var buffer = make([][]string, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		block := strings.Split(line, " ")
		if len(block) > 0 {
			buffer = append(buffer, block)
		}
	}

	return buffer, nil
}
