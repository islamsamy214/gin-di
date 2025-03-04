package middlewares

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
	"web-app/configs"
)

func NewLogIOWriterMiddleware() io.Writer {
	logDir := "./storage/logs/"
	os.MkdirAll(logDir, os.ModePerm) // Ensure log directory exists

	// Generate log file name based on the current date
	logFileName := filepath.Join(logDir, configs.NewAppConfig().Name+"-"+time.Now().Format("2006-01-02")+".log")

	// Open the log file in append mode, create if not exists
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// MultiWriter: log to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Start a separate goroutine to clean up old logs
	go cleanupOldLogs(logDir, 30)

	return multiWriter
}

// cleanupOldLogs removes log files older than the specified retention days
func cleanupOldLogs(logDir string, retentionDays int) {
	files, err := os.ReadDir(logDir)
	if err != nil {
		log.Printf("Error reading log directory: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, file := range files {
		filePath := filepath.Join(logDir, file.Name())

		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Error getting file info for %s: %v", filePath, err)
			continue
		}

		// Delete old log files
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				log.Printf("Error deleting old log file %s: %v", filePath, err)
			} else {
				log.Printf("Deleted old log file: %s", filePath)
			}
		}
	}
}
