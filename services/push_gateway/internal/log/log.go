package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func parseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

var (
	mu       sync.Mutex
	minLevel = LevelInfo
	stdLog   = log.New(os.Stdout, "", 0)
	file     *os.File
)

// Init opens the log file and configures the logger to write to both stdout
// and the file. If the directory does not exist it will be created.
// level: "debug" / "info" / "warn" / "error" (case-insensitive).
func Init(level, logDir, fileName string) error {
	mu.Lock()
	defer mu.Unlock()

	minLevel = parseLevel(level)

	if logDir == "" {
		logDir = "logs"
	}
	if fileName == "" {
		fileName = "push_gateway.log"
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir failed: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(logDir, fileName),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}
	if file != nil {
		_ = file.Close()
	}
	file = f
	stdLog = log.New(io.MultiWriter(os.Stdout, f), "", 0)
	return nil
}

// Close closes the underlying log file (best-effort).
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Close()
		file = nil
	}
}

func output(lvl Level, calldepth int, msg string) {
	if lvl < minLevel {
		return
	}
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}
	line0 := time.Now().Format("2006-01-02 15:04:05") +
		" " + lvl.String() +
		" " + fmt.Sprintf("%s:%d", file, line) +
		" " + msg
	mu.Lock()
	stdLog.Println(line0)
	mu.Unlock()
}

func Debugf(format string, args ...any) { output(LevelDebug, 2, fmt.Sprintf(format, args...)) }
func Infof(format string, args ...any)  { output(LevelInfo, 2, fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...any)  { output(LevelWarn, 2, fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...any) { output(LevelError, 2, fmt.Sprintf(format, args...)) }

// Fatalf logs at error level and exits the process.
func Fatalf(format string, args ...any) {
	output(LevelError, 2, fmt.Sprintf(format, args...))
	os.Exit(1)
}
