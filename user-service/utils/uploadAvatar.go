package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// SaveAvatar is a function to save avatar
func SaveAvatar(fileData []byte, fileName string) (string, error) {
	// upload directory
	uploadDir := "./uploads/avatars"

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err := os.MkdirAll(uploadDir, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	filePath := filepath.Join(uploadDir, fileName)

	// save file to file upload
	err := os.WriteFile(filePath, fileData, 0666)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/uploads/avatars/%s", fileName), nil
}
