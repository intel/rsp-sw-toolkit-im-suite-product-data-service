package helper

import (
	"io"
	"log"
	"os"
	"strings"
)

// GetSecret attempts to retrieve a secret from /run/secrets and will return error if not found
func GetSecret(secretFileName string) (string, error) {
	var filePath string
	//i put this in so I can put a file in the same directory for tests and be able to specify a path instead of assuming /run/secrets and converts to a string properly
	if strings.ContainsAny(secretFileName, "/") {
		filePath = secretFileName
	} else {
		filePath = "/run/secrets/" + secretFileName
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	fs, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := fs.Close(); closeErr != nil {
			log.Println(closeErr)
		}
	}()
	buf := make([]byte, fileInfo.Size())
	_, err = io.ReadFull(fs, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
