package utils

import (
	"bufio"
	"fmt"
	"os"
)

func ReadFileContents(filePath string) ([]byte, error) {

	privateKeyFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("file Open error: %s", err)
	}

	fileInfo, _ := privateKeyFile.Stat()
	fileSize := fileInfo.Size()
	fileContentInBytes := make([]byte, fileSize)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(fileContentInBytes)

    return fileContentInBytes, nil
}
