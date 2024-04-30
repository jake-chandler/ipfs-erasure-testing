package filegenerator

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/testground/sdk-go/runtime"
)

var TEMP string

// FileGenerator provides functionality to generate a file with Lorem Ipsum style content.
type FileGenerator struct {
}

// NewFileGenerator creates a new instance of FileGenerator.
func New() *FileGenerator {
	TEMP = runtime.CurrentRunEnv().StringParam("tempFileDir")
	if TEMP == "" {
		TEMP = "tmp/"
	}
	// Check if TEMP directory exists, create it if it doesn't
	if _, err := os.Stat(TEMP); os.IsNotExist(err) {
		if err := os.MkdirAll(TEMP, 0755); err != nil {
			log.Printf("Error Creating Temp File Directory...")
		}
	}
	return &FileGenerator{}
}

func (fg *FileGenerator) TearDown() {
	os.RemoveAll(TEMP)
}

// randomString generates a random string of the specified length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (fg *FileGenerator) GenerateFilename() string {
	rand.Seed(time.Now().UnixNano())
	randString := randomString(10)
	return fmt.Sprintf("%s.txt", randString)
}

// GenerateFile generates a file with Lorem Ipsum style content.
func (fg *FileGenerator) GenerateFile(fileName string, sizeMB int) (string, error) {
	content := randomString2(sizeMB)
	err := os.WriteFile(TEMP+fileName, []byte(content), 0644)
	if err != nil {
		return TEMP + fileName, fmt.Errorf("error writing file: %v", err)
	}
	return TEMP + fileName, nil
}

// loremIpsum generates Lorem Ipsum style content.
func loremIpsum(sizeMB int) string {
	// Lorem ipsum content
	lorem := `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	// Calculate the number of times the Lorem Ipsum content needs to be repeated to achieve the desired file size
	contentSize := len(lorem)
	repetitions := int(math.Ceil(float64(sizeMB*1024*1024) / float64(contentSize)))

	// Repeat Lorem Ipsum content to achieve approximately the desired file size
	var sb strings.Builder
	for i := 0; i < repetitions; i++ {
		sb.WriteString(lorem)
		sb.WriteString("\n")
	}
	return sb.String()
}
func randomString2(sizeMB int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	// Calculate the number of characters needed to achieve the desired file size
	numChars := sizeMB * 1024 * 1024

	// Generate random characters to achieve approximately the desired file size
	var sb strings.Builder
	for i := 0; i < numChars; i++ {
		randomIndex := rand.Intn(len(charset))
		sb.WriteByte(charset[randomIndex])
	}
	return sb.String()
}
