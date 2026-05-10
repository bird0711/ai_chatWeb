package http

import (
	"errors"
	"io"
	"log"
	"mime/multipart"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type chatFileUpload struct {
	originalName  string
	storagePath   string
	contentType   string
	sizeBytes     int64
	extractedText string
}

func saveChatFileUpload(c *gin.Context) (chatFileUpload, error) {
	file, err := c.FormFile("chat_file")
	if err != nil {
		if errors.Is(err, nethttp.ErrMissingFile) {
			return chatFileUpload{}, errors.New("file is required")
		}
		return chatFileUpload{}, err
	}
	if file.Size <= 0 {
		return chatFileUpload{}, errors.New("file is empty")
	}
	if file.Size > maxChatFileBytes {
		return chatFileUpload{}, errors.New("chat file must be 10MB or smaller")
	}
	ext, err := chatFileExtension(file)
	if err != nil {
		return chatFileUpload{}, err
	}
	src, err := file.Open()
	if err != nil {
		return chatFileUpload{}, err
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Printf("error closing src: %v", err)
		}
	}()
	raw, err := io.ReadAll(io.LimitReader(src, maxChatFileBytes+1))
	if err != nil {
		return chatFileUpload{}, err
	}
	if int64(len(raw)) > maxChatFileBytes {
		return chatFileUpload{}, errors.New("chat file must be 10MB or smaller")
	}
	if err := validateChatFileContent(ext, raw); err != nil {
		return chatFileUpload{}, err
	}
	text, err := extractChatFileText(ext, raw)
	if err != nil {
		return chatFileUpload{}, err
	}
	name, err := randomTokenHex(16)
	if err != nil {
		return chatFileUpload{}, err
	}
	uploadRoot := getenv("CHAT_FILE_DIR", filepath.Join("data", "chat-files"))
	if err := os.MkdirAll(uploadRoot, 0755); err != nil {
		return chatFileUpload{}, err
	}
	filename := name + ext
	dstPath := filepath.Join(uploadRoot, filename)
	if err := os.WriteFile(dstPath, raw, 0644); err != nil {
		return chatFileUpload{}, err
	}
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	return chatFileUpload{
		originalName:  filepath.Base(file.Filename),
		storagePath:   dstPath,
		contentType:   contentType,
		sizeBytes:     int64(len(raw)),
		extractedText: text,
	}, nil
}

func chatFileExtension(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".txt", ".md", ".json", ".csv", ".log", ".docx", ".pdf":
		return ext, nil
	default:
		return "", errors.New("chat file must be txt, md, json, csv, log, docx, or pdf")
	}
}

func saveAvatarUpload(c *gin.Context, existing string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return existing, nil
	}
	file, err := c.FormFile("avatar_file")
	if err != nil {
		if errors.Is(err, nethttp.ErrMissingFile) {
			return existing, nil
		}
		return "", err
	}
	if file.Size > 2*1024*1024 {
		return "", errors.New("avatar image must be 2MB or smaller")
	}
	ext, err := avatarExtension(file)
	if err != nil {
		return "", err
	}
	uploadRoot := getenv("UPLOAD_DIR", "uploads")
	dir := filepath.Join(uploadRoot, "avatars")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Printf("error closing src: %v", err)
		}
	}()

	head := make([]byte, 512)
	n, err := src.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if err := validateAvatarContent(ext, head[:n]); err != nil {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	name, err := randomTokenHex(16)
	if err != nil {
		return "", err
	}
	filename := name + ext
	dstPath := filepath.Join(dir, filename)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := dst.Close(); err != nil {
			log.Printf("error closing dst: %v", err)
		}
	}()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return "/uploads/avatars/" + filename, nil
}

func avatarExtension(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ext, nil
	default:
		return "", errors.New("avatar image must be jpg, png, gif, or webp")
	}
}

func validateChatFileContent(ext string, raw []byte) error {
	detected := strings.ToLower(nethttp.DetectContentType(raw))
	switch ext {
	case ".txt", ".md", ".csv", ".log":
		if strings.HasPrefix(detected, "text/") || detected == "application/octet-stream" {
			return nil
		}
	case ".json":
		if detected == "application/json" || strings.HasPrefix(detected, "text/") {
			return nil
		}
	case ".pdf":
		if detected == "application/pdf" {
			return nil
		}
	case ".docx":
		if detected == "application/zip" || detected == "application/octet-stream" || detected == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
			return nil
		}
	}
	return errors.New("chat file content does not match its file type")
}

func validateAvatarContent(ext string, raw []byte) error {
	detected := strings.ToLower(nethttp.DetectContentType(raw))
	switch ext {
	case ".jpg", ".jpeg":
		if detected == "image/jpeg" {
			return nil
		}
	case ".png":
		if detected == "image/png" {
			return nil
		}
	case ".gif":
		if detected == "image/gif" {
			return nil
		}
	case ".webp":
		if detected == "image/webp" {
			return nil
		}
	}
	return errors.New("avatar image content does not match its file type")
}
