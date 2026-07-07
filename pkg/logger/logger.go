package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	debugEnabled bool
	mu           sync.RWMutex
	listener     func(string)
)

func Configure(debug bool) {
	debugEnabled = debug
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func SetListener(l func(string)) {
	mu.Lock()
	defer mu.Unlock()
	listener = l
}

func notifyListener(msg string) {
	mu.RLock()
	l := listener
	mu.RUnlock()
	if l != nil {
		l(msg)
	}
}

func Debugf(format string, args ...any) {
	if debugEnabled {
		msg := fmt.Sprintf("[DEBUG] "+format, args...)
		log.Output(2, msg)
		notifyListener(msg)
	}
}

func Infof(format string, args ...any) {
	msg := fmt.Sprintf("[INFO] "+format, args...)
	log.Output(2, msg)
	notifyListener(msg)
}

func Errorf(format string, args ...any) {
	msg := fmt.Sprintf("[ERROR] "+format, args...)
	log.Output(2, msg)
	notifyListener(msg)
}
