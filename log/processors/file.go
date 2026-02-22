package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileProcessor struct {
	Directory string // default: ./storage/logs
}

func NewFileProcessor(directory string) *FileProcessor {
	if directory == "" {
		directory = "./storage/logs"
	}

	return &FileProcessor{
		Directory: directory,
	}
}

func (p *FileProcessor) Process(record []byte) {
	now := time.Now()

	// Build path: ./storage/logs/2025/06/27/15/log.txt
	dirPath := filepath.Join(
		p.Directory,
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	// Ensure the directory exists
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		return
	}

	// Open (or create) the log file in append mode
	filePath := filepath.Join(dirPath, fmt.Sprintf("%02d.txt", now.Hour()))
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}
	defer file.Close()

	// Write log record with a newline
	if _, err := file.Write(append(record, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to log file: %v\n", err)
	}
}
