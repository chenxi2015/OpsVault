package logger

import (
	"log"
	"os"
)

var debugEnabled bool

func Configure(debug bool) {
	debugEnabled = debug
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func Debugf(format string, args ...any) {
	if debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func Infof(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}

func Errorf(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
}
