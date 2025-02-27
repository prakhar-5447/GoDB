package audit

import (
	"log"
	"os"
)

var AuditLogger *log.Logger

// InitAuditLogger initializes the audit logger.
func InitAuditLogger() {
	// Open (or create) an audit log file in append mode.
	file, err := os.OpenFile("audit.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open audit log file: %v", err)
	}
	// Create a logger with a specific prefix and log format.
	AuditLogger = log.New(file, "AUDIT: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogEvent logs an audit event.
func LogEvent(event string) {
	if AuditLogger != nil {
		AuditLogger.Println(event)
	} else {
		log.Println("AUDIT: ", event)
	}
}
