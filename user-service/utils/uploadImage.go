package utils

import (
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	AvatarDir = "./uploads/avatars"
)

// UploadAvatarFromBase64 upload avatar from base64 data
func UploadAvatarFromBase64(base64Data string, username, mimeType string) (string, error) {
	err := os.MkdirAll(AvatarDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("gagal membuat direktori upload: %v", err)
	}

	// Decode Base64
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("gagal decode base64: %v", err)
	}

	fileExt := ".jpg" // default
	if mimeType != "" {
		switch mimeType {
		case "image/jpeg":
			fileExt = ".jpg"
		case "image/png":
			fileExt = ".png"
		case "image/gif":
			fileExt = ".gif"
		}
	}

	timestamp := time.Now().UnixNano()
	newFilename := fmt.Sprintf("%s_%d%s", username, timestamp, fileExt)

	filePath := filepath.Join(AvatarDir, newFilename)

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %v", err)
	}

	relativePath := fmt.Sprintf("/avatars/%s", newFilename)
	logrus.Infof("File avatar berhasil disimpan: %s", relativePath)

	return relativePath, nil
}

// DeleteOldAvatar delete old avatar if exist
func DeleteOldAvatar(oldAvatarPath string) error {
	if oldAvatarPath == "" {
		return nil
	}

	if !strings.HasPrefix(oldAvatarPath, "/avatars/") {
		return nil
	}

	filename := strings.TrimPrefix(oldAvatarPath, "/avatars/")

	filePath := filepath.Join(AvatarDir, filename)

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		logrus.Warnf("File avatar lama tidak ditemukan: %s", filePath)
		return nil
	}

	// delete file
	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("gagal menghapus file avatar lama: %v", err)
	}

	logrus.Infof("File avatar lama berhasil dihapus: %s", filePath)
	return nil
}
