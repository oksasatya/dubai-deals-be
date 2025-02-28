package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// UploadAvatar is a function to upload avatar
func UploadAvatar(file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	// upload directory
	uploadDir := "./uploads/avatars"

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err := os.MkdirAll(uploadDir, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	fileName := fmt.Sprintf("%d_%s_%d", fileHeader.Size, filepath.Ext(fileHeader.Filename), time.Now().UnixNano())
	filePath := filepath.Join(uploadDir, fileName)

	// save file to file upload
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(dst)

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/uploads/avatars/%s", fileName), nil
}

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
