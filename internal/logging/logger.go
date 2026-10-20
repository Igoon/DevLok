package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Dir        string
	Service    string
	MaxBytes   int64
	MaxFiles   int
	AlsoStderr bool
}

type Logger struct {
	mu       sync.Mutex
	opts     Options
	file     *os.File
	size     int64
	redactor func(map[string]any) map[string]any
}

type entry struct {
	TS      string         `json:"ts"`
	Level   string         `json:"level"`
	Service string         `json:"service,omitempty"`
	Event   string         `json:"event,omitempty"`
	Msg     string         `json:"msg,omitempty"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func New(opts Options) (*Logger, error) {
	if opts.Dir == "" {
		return nil, fmt.Errorf("log dir is required")
	}
	if opts.Service == "" {
		opts.Service = "devlok"
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 10 << 20
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 5
	}
	if err := os.MkdirAll(opts.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	l := &Logger{opts: opts}
	if err := l.open(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Logger) path() string {
	return filepath.Join(l.opts.Dir, l.opts.Service+".jsonl")
}

func (l *Logger) open() error {
	f, err := os.OpenFile(l.path(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	l.file = f
	l.size = info.Size()
	return nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) Debug(event, msg string, fields map[string]any) {
	l.write("debug", event, msg, fields)
}
func (l *Logger) Info(event, msg string, fields map[string]any) { l.write("info", event, msg, fields) }
func (l *Logger) Warn(event, msg string, fields map[string]any) { l.write("warn", event, msg, fields) }
func (l *Logger) Error(event, msg string, fields map[string]any) {
	l.write("error", event, msg, fields)
}

func (l *Logger) write(level, event, msg string, fields map[string]any) {
	e := entry{
		TS:      time.Now().UTC().Format(time.RFC3339Nano),
		Level:   level,
		Service: l.opts.Service,
		Event:   event,
		Msg:     msg,
		Fields:  Redact(fields),
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return
	}
	if l.size > 0 && l.size+int64(len(data)) > l.opts.MaxBytes {
		if err := l.rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "devlok: rotate log: %v\n", err)
		}
	}
	n, err := l.file.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "devlok: write log: %v\n", err)
		return
	}
	l.size += int64(n)
	if l.opts.AlsoStderr {
		_, _ = os.Stderr.Write(data)
	}
}

func (l *Logger) rotate() error {
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	base := l.path()
	for i := l.opts.MaxFiles - 1; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", base, i)
		to := fmt.Sprintf("%s.%d", base, i+1)
		if _, err := os.Stat(from); err == nil {
			_ = os.Remove(to)
			if err := os.Rename(from, to); err != nil {
				return err
			}
		}
	}
	if _, err := os.Stat(base); err == nil {
		_ = os.Remove(base + ".1")
		if err := os.Rename(base, base+".1"); err != nil {
			return err
		}
	}
	return l.open()
}

var sensitiveKeys = map[string]bool{
	"password":      true,
	"passwd":        true,
	"pwd":           true,
	"secret":        true,
	"token":         true,
	"key":           true,
	"apikey":        true,
	"api_key":       true,
	"authorization": true,
	"private_key":   true,
	"ca_key":        true,
	"mysql_pwd":     true,
}

func Redact(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	out := make(map[string]any, len(fields))
	for k, v := range fields {
		if isSensitive(k) {
			out[k] = "[REDACTED]"
			continue
		}
		out[k] = redactValue(v)
	}
	return out
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return Redact(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = redactValue(item)
		}
		return out
	default:
		return v
	}
}

func isSensitive(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if sensitiveKeys[k] {
		return true
	}
	for _, suffix := range []string{"_password", "_passwd", "_secret", "_token", "_key", "_apikey", "_api_key"} {
		if strings.HasSuffix(k, suffix) {
			return true
		}
	}
	return false
}
