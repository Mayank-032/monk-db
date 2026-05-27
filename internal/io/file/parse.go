package file

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func ParseFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("unable to open file: ", err.Error())
		return nil, err
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
