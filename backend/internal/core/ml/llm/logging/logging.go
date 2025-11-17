package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Log levels
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var currentLevel = INFO

func init() {
	if os.Getenv("DEBUG") == "true" {
		currentLevel = DEBUG
	}
}

func Debug(msg string, args ...interface{}) {
	if currentLevel <= DEBUG {
		logWithLevel("DEBUG", msg, args...)
	}
}

func Info(msg string, args ...interface{}) {
	if currentLevel <= INFO {
		logWithLevel("INFO", msg, args...)
	}
}

func Warn(msg string, args ...interface{}) {
	if currentLevel <= WARN {
		logWithLevel("WARN", msg, args...)
	}
}

func Error(msg string, args ...interface{}) {
	if currentLevel <= ERROR {
		logWithLevel("ERROR", msg, args...)
	}
}

func InfoPersist(msg string) {
	logWithLevel("INFO", msg)
}

func WarnPersist(msg string, args ...interface{}) {
	logWithLevel("WARN", msg, args...)
}

func ErrorPersist(msg string) {
	logWithLevel("ERROR", msg)
}

func logWithLevel(level string, msg string, args ...interface{}) {
	if len(args) > 0 {
		// Handle key-value pairs
		formattedArgs := ""
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				formattedArgs += fmt.Sprintf(" %v=%v", args[i], args[i+1])
			} else {
				formattedArgs += fmt.Sprintf(" %v", args[i])
			}
		}
		log.Printf("[%s] %s%s", level, msg, formattedArgs)
	} else {
		log.Printf("[%s] %s", level, msg)
	}
}

func RecoverPanic(context string, fallback func()) {
	if r := recover(); r != nil {
		ErrorPersist(fmt.Sprintf("Panic in %s: %v", context, r))
		if fallback != nil {
			fallback()
		}
	}
}

func WriteToolResultsJson(sessionID string, seqId int, toolResults interface{}) string {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		ErrorPersist(fmt.Sprintf("Failed to create logs directory: %v", err))
		return ""
	}

	filename := fmt.Sprintf("tool_results_%s_%d.json", sessionID, seqId)
	filepath := filepath.Join(logsDir, filename)

	data, err := json.MarshalIndent(toolResults, "", "  ")
	if err != nil {
		ErrorPersist(fmt.Sprintf("Failed to marshal tool results: %v", err))
		return ""
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		ErrorPersist(fmt.Sprintf("Failed to write tool results file: %v", err))
		return ""
	}

	return filepath
}

// PersistTimeArg creates a time argument for logging
func PersistTimeArg(key string, value interface{}) interface{} {
	return fmt.Sprintf("%s=%v", key, value)
}

// WriteRequestMessageJson writes request message to a JSON file
func WriteRequestMessageJson(sessionID string, seqId int, message interface{}) string {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		ErrorPersist(fmt.Sprintf("Failed to create logs directory: %v", err))
		return ""
	}

	filename := fmt.Sprintf("request_message_%s_%d.json", sessionID, seqId)
	filepath := filepath.Join(logsDir, filename)

	data, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		ErrorPersist(fmt.Sprintf("Failed to marshal request message: %v", err))
		return ""
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		ErrorPersist(fmt.Sprintf("Failed to write request message file: %v", err))
		return ""
	}

	return filepath
}
