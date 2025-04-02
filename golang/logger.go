// Package vlog implements a versatile logging system with hierarchical levels
// and performance optimizations for both production and development environments.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Level represents the severity/verbosity level of a log message
type Level int32

// Log levels in increasing order of verbosity
const (
	LevelEmergency Level = iota // System is unusable
	LevelAlert                  // Action must be taken immediately
	LevelCritical               // Critical conditions
	LevelError                  // Error conditions
	LevelWarning                // Warning conditions
	LevelNotice                 // Normal but significant condition
	LevelInfo                   // Informational
	LevelDebug                  // Debug-level messages
	LevelVerbose                // Verbose debug messages
	LevelTrace                  // Extremely detailed tracing
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelEmergency:
		return "EMERG"
	case LevelAlert:
		return "ALERT"
	case LevelCritical:
		return "CRIT"
	case LevelWarning:
		return "WARN"
	case LevelNotice:
		return "NOTICE"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelVerbose:
		return "VERB"
	case LevelTrace:
		return "TRACE"
	default:
		return fmt.Sprintf("LEVEL%d", l)
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Component  string                 `json:"component,omitempty"`
	File       string                 `json:"file,omitempty"`
	Line       int                    `json:"line,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	InstanceID string                 `json:"instance_id,omitempty"`
}

// OutputFormat defines how logs should be formatted
type OutputFormat int

const (
	// FormatText outputs logs in human-readable text format
	FormatText OutputFormat = iota
	// FormatJSON outputs logs in JSON format for machine processing
	FormatJSON
)

// Output defines where logs should be written
type Output interface {
	Write(entry *LogEntry) error
	Close() error
}

// FileOutput implements Output to write logs to a file
type FileOutput struct {
	mu             sync.Mutex
	file           *os.File
	path           string
	format         OutputFormat
	maxSize        int64
	currentSize    int64
	rotateCallback func(string)
}

// NewFileOutput creates a new file output
func NewFileOutput(path string, format OutputFormat, maxSizeMB int) (*FileOutput, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &FileOutput{
		file:        file,
		path:        path,
		format:      format,
		maxSize:     int64(maxSizeMB) * 1024 * 1024,
		currentSize: info.Size(),
	}, nil
}

// SetRotateCallback sets a function to be called after log rotation
func (o *FileOutput) SetRotateCallback(fn func(string)) {
	o.rotateCallback = fn
}

// Write writes a log entry to the file
func (o *FileOutput) Write(entry *LogEntry) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	var data []byte
	var err error

	if o.format == FormatJSON {
		data, err = json.Marshal(entry)
		if err != nil {
			return err
		}
		data = append(data, '\n')
	} else {
		// Text format
		timeStr := entry.Timestamp.Format("2006-01-02 15:04:05.000")
		location := ""
		if entry.File != "" {
			location = fmt.Sprintf(" [%s:%d]", entry.File, entry.Line)
		}
		component := ""
		if entry.Component != "" {
			component = " (" + entry.Component + ")"
		}

		line := fmt.Sprintf("%s [%s]%s%s %s", timeStr, entry.Level, component, location, entry.Message)
		if len(entry.Fields) > 0 {
			fieldsData, _ := json.Marshal(entry.Fields)
			line += " " + string(fieldsData)
		}
		line += "\n"
		data = []byte(line)
	}

	// Check if we need to rotate the log file
	if o.maxSize > 0 && o.currentSize+int64(len(data)) > o.maxSize {
		err := o.rotate()
		if err != nil {
			return err
		}
	}

	n, err := o.file.Write(data)
	if err == nil {
		o.currentSize += int64(n)
	}
	return err
}

// rotate performs log rotation
func (o *FileOutput) rotate() error {
	if err := o.file.Close(); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", o.path, timestamp)

	if err := os.Rename(o.path, rotatedPath); err != nil {
		// Try to reopen the original file
		var reopenErr error
		o.file, reopenErr = os.OpenFile(o.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if reopenErr != nil {
			return fmt.Errorf("failed to rotate log: %v and failed to reopen: %v", err, reopenErr)
		}
		return err
	}

	// Open a new log file
	file, err := os.OpenFile(o.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	o.file = file
	o.currentSize = 0

	// Call rotation callback if set
	if o.rotateCallback != nil {
		go o.rotateCallback(rotatedPath)
	}

	return nil
}

// Close closes the file output
func (o *FileOutput) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Close()
}

// ConsoleOutput implements Output to write logs to the console
type ConsoleOutput struct {
	mu     sync.Mutex
	writer io.Writer
	format OutputFormat
}

// NewConsoleOutput creates a new console output
func NewConsoleOutput(writer io.Writer, format OutputFormat) *ConsoleOutput {
	return &ConsoleOutput{
		writer: writer,
		format: format,
	}
}

// Write writes a log entry to the console
func (o *ConsoleOutput) Write(entry *LogEntry) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.format == FormatJSON {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(o.writer, string(data))
		return err
	}

	// Text format with ANSI colors
	var levelColor string
	switch entry.Level {
	case "EMERG", "ALERT", "CRIT":
		levelColor = "\033[1;31m" // Bold Red
	case "ERROR":
		levelColor = "\033[31m" // Red
	case "WARN":
		levelColor = "\033[33m" // Yellow
	case "NOTICE":
		levelColor = "\033[1;34m" // Bold Blue
	case "INFO":
		levelColor = "\033[32m" // Green
	case "DEBUG":
		levelColor = "\033[36m" // Cyan
	case "VERB", "TRACE":
		levelColor = "\033[35m" // Magenta
	default:
		levelColor = "\033[0m" // Reset
	}
	resetColor := "\033[0m"

	timeStr := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	location := ""
	if entry.File != "" {
		location = fmt.Sprintf(" \033[90m[%s:%d]\033[0m", entry.File, entry.Line)
	}
	component := ""
	if entry.Component != "" {
		component = " (" + entry.Component + ")"
	}

	line := fmt.Sprintf("%s [%s%s%s]%s%s %s",
		timeStr,
		levelColor, entry.Level, resetColor,
		component, location, entry.Message)

	if len(entry.Fields) > 0 {
		fieldsData, _ := json.Marshal(entry.Fields)
		line += " \033[90m" + string(fieldsData) + "\033[0m"
	}

	_, err := fmt.Fprintln(o.writer, line)
	return err
}

// Close is a no-op for console output
func (o *ConsoleOutput) Close() error {
	return nil
}

// Logger is the main logging structure
type Logger struct {
	level           int32 // Atomic access
	outputs         []Output
	defaultFields   map[string]interface{}
	instanceID      string
	component       string
	componentLevels map[string]Level
	mu              sync.RWMutex
	asyncQueue      chan *LogEntry
	wg              sync.WaitGroup
	done            chan struct{}
	sampler         *rateSampler
}

// rateSampler implements log sampling to reduce volume
type rateSampler struct {
	mu            sync.Mutex
	samplingRates map[string]int
	counters      map[string]int
}

func newRateSampler() *rateSampler {
	return &rateSampler{
		samplingRates: make(map[string]int),
		counters:      make(map[string]int),
	}
}

// SetSamplingRate sets how often a log with a given key should be emitted
// A rate of 1 means every log, 100 means 1 out of every 100 logs
func (s *rateSampler) SetSamplingRate(key string, rate int) {
	if rate < 1 {
		rate = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samplingRates[key] = rate
	delete(s.counters, key) // Reset counter when rate changes
}

// ShouldLog determines if a log with the given key should be emitted
func (s *rateSampler) ShouldLog(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	rate, exists := s.samplingRates[key]
	if !exists || rate <= 1 {
		return true // Log everything if no sampling rate is set
	}

	counter, _ := s.counters[key]
	counter = (counter + 1) % rate
	s.counters[key] = counter

	return counter == 0 // Only log when counter is 0
}

// NewLogger creates a new logger
func NewLogger() *Logger {
	logger := &Logger{
		level:           int32(LevelInfo),
		outputs:         make([]Output, 0),
		defaultFields:   make(map[string]interface{}),
		componentLevels: make(map[string]Level),
		asyncQueue:      make(chan *LogEntry, 1000),
		done:            make(chan struct{}),
		sampler:         newRateSampler(),
	}

	// Generate a unique instance ID
	logger.instanceID = fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())

	// Start background worker for async logging
	logger.wg.Add(1)
	go logger.processLogQueue()

	return logger
}

// processLogQueue handles asynchronous logging
func (l *Logger) processLogQueue() {
	defer l.wg.Done()

	for {
		select {
		case entry := <-l.asyncQueue:
			l.writeLogEntry(entry)
		case <-l.done:
			// Process remaining logs before exiting
			for {
				select {
				case entry := <-l.asyncQueue:
					l.writeLogEntry(entry)
				default:
					return
				}
			}
		}
	}
}

// writeLogEntry writes a log entry to all outputs
func (l *Logger) writeLogEntry(entry *LogEntry) {
	l.mu.RLock()
	outputs := l.outputs
	l.mu.RUnlock()

	for _, output := range outputs {
		if err := output.Write(entry); err != nil {
			// If we can't write to the log, try to write to stderr
			fmt.Fprintf(os.Stderr, "ERROR: Failed to write log: %v\n", err)
		}
	}
}

// AddOutput adds a new output destination
func (l *Logger) AddOutput(output Output) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outputs = append(l.outputs, output)
}

// SetLevel sets the global log level
func (l *Logger) SetLevel(level Level) {
	atomic.StoreInt32((*int32)(&l.level), int32(level))
}

// GetLevel gets the current global log level
func (l *Logger) GetLevel() Level {
	return Level(atomic.LoadInt32((*int32)(&l.level)))
}

// SetComponentLevel sets the log level for a specific component
func (l *Logger) SetComponentLevel(component string, level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.componentLevels[component] = level
}

// isLoggable checks if a message at the given level should be logged
func (l *Logger) isLoggable(level Level, component string) bool {
	// Check component-specific level first
	if component != "" {
		l.mu.RLock()
		compLevel, exists := l.componentLevels[component]
		l.mu.RUnlock()

		if exists {
			return level <= compLevel
		}
	}

	// Fall back to global level
	return level <= Level(atomic.LoadInt32((*int32)(&l.level)))
}

// SetDefaultField sets a field that will be included in all log entries
func (l *Logger) SetDefaultField(key string, value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.defaultFields[key] = value
}

// With creates a new logger with the given component
func (l *Logger) With(component string) *Logger {
	newLogger := &Logger{
		level:           l.level,
		outputs:         l.outputs,
		instanceID:      l.instanceID,
		component:       component,
		componentLevels: l.componentLevels,
		asyncQueue:      l.asyncQueue,
		done:            l.done,
		wg:              l.wg,
		sampler:         l.sampler,
	}

	// Copy default fields
	l.mu.RLock()
	newLogger.defaultFields = make(map[string]interface{}, len(l.defaultFields))
	for k, v := range l.defaultFields {
		newLogger.defaultFields[k] = v
	}
	l.mu.RUnlock()

	return newLogger
}

// WithFields creates a new logger with additional default fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:           l.level,
		outputs:         l.outputs,
		instanceID:      l.instanceID,
		component:       l.component,
		componentLevels: l.componentLevels,
		asyncQueue:      l.asyncQueue,
		done:            l.done,
		wg:              l.wg,
		sampler:         l.sampler,
	}

	// Copy and merge default fields
	l.mu.RLock()
	newLogger.defaultFields = make(map[string]interface{}, len(l.defaultFields)+len(fields))
	for k, v := range l.defaultFields {
		newLogger.defaultFields[k] = v
	}
	l.mu.RUnlock()

	for k, v := range fields {
		newLogger.defaultFields[k] = v
	}

	return newLogger
}

// WithField creates a new logger with an additional default field (convenience method)
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(map[string]interface{}{key: value})
}

// log logs a message at the given level
func (l *Logger) log(level Level, skip int, format string, args ...interface{}) {
	if !l.isLoggable(level, l.component) {
		return
	}

	entry := &LogEntry{
		Timestamp:  time.Now(),
		Level:      level.String(),
		Component:  l.component,
		InstanceID: l.instanceID,
	}

	// Check if the last argument is a fields map
	var fields map[string]interface{}
	if len(args) > 0 {
		lastArg := args[len(args)-1]
		if fieldsMap, ok := lastArg.(map[string]interface{}); ok {
			// Create a defensive copy of the fields map
			fields = make(map[string]interface{}, len(fieldsMap))
			for k, v := range fieldsMap {
				fields[k] = v
			}
			// Remove the fields map from args for message formatting
			args = args[:len(args)-1]
		}
	}

	// Format the message
	if len(args) > 0 {
		entry.Message = fmt.Sprintf(format, args...)
	} else {
		entry.Message = format
	}

	// Add source file and line information
	if pc, file, line, ok := runtime.Caller(skip + 1); ok {
		entry.File = filepath.Base(file)
		entry.Line = line

		// Optionally add function name to fields
		if l.isLoggable(LevelTrace, l.component) {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				if entry.Fields == nil {
					entry.Fields = make(map[string]interface{})
				}
				entry.Fields["func"] = filepath.Base(fn.Name())
			}
		}
	}

	// Add default fields
	l.mu.RLock()
	if len(l.defaultFields) > 0 {
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{}, len(l.defaultFields))
		}
		for k, v := range l.defaultFields {
			entry.Fields[k] = v
		}
	}
	l.mu.RUnlock()

	// Add per-message fields if provided
	if fields != nil {
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{}, len(fields))
		}
		for k, v := range fields {
			entry.Fields[k] = v
		}
	}

	// Send to async queue
	select {
	case l.asyncQueue <- entry:
		// Successfully queued
	default:
		// Queue is full, log to stderr as fallback
		fmt.Fprintf(os.Stderr, "WARNING: Log queue full, dropping log: %s\n", entry.Message)
	}
}

// logWithSampling logs a message with rate limiting based on the sampling key
func (l *Logger) logWithSampling(level Level, samplingKey string, skip int, format string, args ...interface{}) {
	if !l.isLoggable(level, l.component) {
		return
	}

	if samplingKey != "" && !l.sampler.ShouldLog(samplingKey) {
		return
	}

	l.log(level, skip+1, format, args...)
}

// Emergency logs at emergency level
func (l *Logger) Emergency(format string, args ...interface{}) {
	l.log(LevelEmergency, 1, format, args...)
}

// Alert logs at alert level
func (l *Logger) Alert(format string, args ...interface{}) {
	l.log(LevelAlert, 1, format, args...)
}

// Critical logs at critical level
func (l *Logger) Critical(format string, args ...interface{}) {
	l.log(LevelCritical, 1, format, args...)
}

// Error logs at error level
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, 1, format, args...)
}

// Warning logs at warning level
func (l *Logger) Warning(format string, args ...interface{}) {
	l.log(LevelWarning, 1, format, args...)
}

// Notice logs at notice level
func (l *Logger) Notice(format string, args ...interface{}) {
	l.log(LevelNotice, 1, format, args...)
}

// Info logs at info level
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, 1, format, args...)
}

// Debug logs at debug level
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, 1, format, args...)
}

// Verbose logs at verbose level
func (l *Logger) Verbose(format string, args ...interface{}) {
	l.log(LevelVerbose, 1, format, args...)
}

// Trace logs at trace level
func (l *Logger) Trace(format string, args ...interface{}) {
	l.log(LevelTrace, 1, format, args...)
}

// SampledInfo logs at info level with rate limiting
func (l *Logger) SampledInfo(key string, rate int, format string, args ...interface{}) {
	l.sampler.SetSamplingRate(key, rate)
	l.logWithSampling(LevelInfo, key, 1, format, args...)
}

// SampledError logs at error level with rate limiting
func (l *Logger) SampledError(key string, rate int, format string, args ...interface{}) {
	l.sampler.SetSamplingRate(key, rate)
	l.logWithSampling(LevelError, key, 1, format, args...)
}

// SampledDebug logs at debug level with rate limiting
func (l *Logger) SampledDebug(key string, rate int, format string, args ...interface{}) {
	l.sampler.SetSamplingRate(key, rate)
	l.logWithSampling(LevelDebug, key, 1, format, args...)
}

// Flush flushes all pending log entries
func (l *Logger) Flush() {
	// Wait for the async queue to empty
	for len(l.asyncQueue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
}

// Close closes the logger and all outputs
func (l *Logger) Close() {
	// Signal the worker to stop
	close(l.done)

	// Wait for worker to finish
	l.wg.Wait()

	// Close all outputs
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, output := range l.outputs {
		output.Close()
	}
}

// Default logger instance
var defaultLogger *Logger
var once sync.Once

// init initializes the default logger
func init() {
	once.Do(func() {
		defaultLogger = NewLogger()
		defaultLogger.AddOutput(NewConsoleOutput(os.Stdout, FormatText))
	})
}

// GetLogger returns the default logger
func GetLogger() *Logger {
	return defaultLogger
}

// SetDefaultLogger sets the default logger
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// Emergency logs to the default logger at emergency level
func Emergency(format string, args ...interface{}) {
	defaultLogger.Emergency(format, args...)
}

// Alert logs to the default logger at alert level
func Alert(format string, args ...interface{}) {
	defaultLogger.Alert(format, args...)
}

// Critical logs to the default logger at critical level
func Critical(format string, args ...interface{}) {
	defaultLogger.Critical(format, args...)
}

// Error logs to the default logger at error level
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Warning logs to the default logger at warning level
func Warning(format string, args ...interface{}) {
	defaultLogger.Warning(format, args...)
}

// Notice logs to the default logger at notice level
func Notice(format string, args ...interface{}) {
	defaultLogger.Notice(format, args...)
}

// Info logs to the default logger at info level
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Debug logs to the default logger at debug level
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Verbose logs to the default logger at verbose level
func Verbose(format string, args ...interface{}) {
	defaultLogger.Verbose(format, args...)
}

// Trace logs to the default logger at trace level
func Trace(format string, args ...interface{}) {
	defaultLogger.Trace(format, args...)
}
