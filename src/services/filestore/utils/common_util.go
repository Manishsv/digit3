package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"gin/models"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type CommonUtil struct{}

// Helper to write JSON
func (c *CommonUtil) WriteJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (c *CommonUtil) MapFileToArtifact(files []*multipart.FileHeader, module string, tag string, tenantId string) []models.Artifact {

	folderName := c.GetFolderName(module, tenantId)
	var artifactList []models.Artifact

	for _, file := range files {

		fileLocation := c.BuildFileLocation(file.Filename, module, tenantId, tag, folderName)

		inputStreamAsString := c.ReadFileIntoString(file)

		artifact := c.BuildArtifact(file, fileLocation, inputStreamAsString)

		artifactList = append(artifactList, artifact)
	}

	return artifactList
}

// mapArtifactsToEntities converts a list of Artifact to ArtifactEntity
func (c *CommonUtil) MapArtifactsToEntities(artifacts []models.Artifact, source string) []models.ArtifactEntity {
	var artifactEntities []models.ArtifactEntity

	for _, artifact := range artifacts {
		entity := models.ArtifactEntity{

			FileStoreID:      artifact.FileLocation.FileStoreID,
			FileName:         artifact.FileLocation.FileName,
			ContentType:      artifact.MultipartFile.Header.Get("Content-Type"),
			Module:           artifact.FileLocation.Module,
			Tag:              artifact.FileLocation.Tag,
			TenantID:         artifact.FileLocation.TenantID,
			FileSource:       source,
			CreatedBy:        artifact.CreatedBy,
			LastModifiedBy:   artifact.LastModifiedBy,
			CreatedTime:      artifact.CreatedTime.Unix(),
			LastModifiedTime: artifact.LastModifiedTime.Unix(),
		}
		artifactEntities = append(artifactEntities, entity)
	}

	return artifactEntities
}

func (c *CommonUtil) BuildArtifact(file *multipart.FileHeader, fileLocation models.FileLocation, inputStreamAsString string) models.Artifact {
	artifact := models.Artifact{
		FileContentInString: inputStreamAsString,
		MultipartFile:       *file,
		FileLocation:        fileLocation,
	}
	return artifact
}

func (c *CommonUtil) ReadFileIntoString(file *multipart.FileHeader) string {

	// Read file content
	src, err := file.Open()
	if err != nil {
		log.Printf("Error opening file: %v", err)
	}
	defer src.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, src)
	if err != nil {
		log.Printf("Error reading file: %v", err)
	}
	inputStreamAsString := buf.String()
	return inputStreamAsString
}

func (c *CommonUtil) BuildFileLocation(fileName string, module string, tenantId string, tag string, folderName string) models.FileLocation {

	randomString, err := c.SecureRandomString(8, true, true) // assuming utility for random string
	if err != nil {
		log.Printf("Error generating random string: %v", err)
	}

	originalFileName := fileName
	imageType := strings.ToLower(filepath.Ext(originalFileName)) // includes the "."
	newFileName := fmt.Sprintf("%s%d%s%s", folderName, time.Now().UnixMilli(), randomString, imageType)
	id := uuid.New().String()

	fileLocation := models.FileLocation{
		FileStoreID: id,
		Module:      module,
		Tag:         tag,
		TenantID:    tenantId,
		FileName:    newFileName,
	}
	return fileLocation
}

func (c *CommonUtil) GetFolderName(module string, tenantId string) string {
	now := time.Now()
	// TO DO: get bucket name from minio config
	bucketName := os.Getenv("MINIO_BUCKET")
	return fmt.Sprintf("%s/%s", bucketName, c.GetFolderNameWithTime(module, tenantId, now))
}

func (c *CommonUtil) GetFolderNameWithTime(module, tenantId string, t time.Time) string {
	monthName := t.Month().String()
	day := t.Day()

	// Use cases.Title instead of strings.Title
	formattedMonth := cases.Title(language.English).String(strings.ToLower(monthName))

	return fmt.Sprintf("%s/%s/%s/%d/", tenantId, module, formattedMonth, day)
}

func (c *CommonUtil) SecureRandomString(length int, useLetters, useNumbers bool) (string, error) {
	charset := ""
	if useLetters {
		charset += "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if useNumbers {
		charset += "0123456789"
	}
	if len(charset) == 0 {
		return "", nil
	}

	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}

func (c *CommonUtil) GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
