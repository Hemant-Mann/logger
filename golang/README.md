# Advanced Go Logging System

This package provides a comprehensive logging system for Go applications that follows best practices inspired by high-performance proxy systems like Twemproxy. It's designed for applications that need both efficient production logging and detailed debug capabilities.

## Key Features

### Hierarchical Log Levels

The system implements 10 distinct log levels with increasing verbosity:

1. **Emergency (0)**: System is unusable
2. **Alert (1)**: Action must be taken immediately
3. **Critical (2)**: Critical conditions
4. **Error (3)**: Error conditions
5. **Warning (4)**: Warning conditions
6. **Notice (5)**: Normal but significant events
7. **Info (6)**: Informational messages
8. **Debug (7)**: Basic debugging information
9. **Verbose (8)**: Detailed debugging information
10. **Trace (9)**: Extremely detailed tracing

### Multi-Stage Filtering

- **Compile-time optimization**: Debug logging can be conditionally compiled
- **Runtime level control**: Global and per-component log levels
- **Message sampling**: Rate-limiting for high-volume log messages

### Multiple Output Destinations

- **Console output**: Human-readable text with ANSI colors or machine-parseable JSON
- **File output**: With automatic rotation based on size
- **Extensible**: Implement the `Output` interface for custom destinations

### Rich Context and Structured Data

- **Automatic context**: File, line number, timestamp, and instance ID
- **Component tagging**: Identify which part of the system generated the log
- **Default fields**: Add global context to all log messages
- **Custom fields**: Add per-message structured data

### Performance Optimizations

- **Asynchronous logging**: Non-blocking log calls with buffered channel
- **Level check before formatting**: Skip string formatting for disabled levels
- **Rate limiting**: Control logging frequency for high-volume events
- **Efficient memory usage**: Minimize allocations in hot paths

## Usage Examples

### Basic Usage

```go
import "github.com/hemant-mann/logger/golang"

func main() {
    // Use the default logger
    logger.Info("Application starting up")
    
    // Log at different levels
    logger.Debug("This is a debug message")
    logger.Error("Something went wrong: %v", err)
    
    // Use formatting
    logger.Info("User %s logged in from %s", username, ipAddress)
}
```

### Component-Specific Logging

```go
// Create a component-specific logger
netLogger := logger.GetLogger().With("network")

// These logs will be tagged with the component
netLogger.Info("Listening on port 8080")
netLogger.Debug("Accepted connection from %s", clientIP)

// Set component-specific log level
logger.GetLogger().SetComponentLevel("network", logger.LevelVerbose)
```

### Structured Logging

```go
// Add fields to a specific log
logger.Info("User authenticated", map[string]interface{}{
    "user_id": 123,
    "role": "admin",
    "ip": "192.168.1.1",
})

// Create a logger with default fields
userLogger := logger.WithFields(map[string]interface{}{
    "user_id": 123,
    "session_id": "abc-123",
})

// All logs from this logger will include the fields
userLogger.Info("User performed action")
userLogger.Error("Permission denied")
```

### Rate-Limited Logging

```go
// Only log 1 out of every 100 occurrences
for i := 0; i < 1000; i++ {
    logger.SampledInfo("high-volume-event", 100, 
        "Processing item %d", i)
}

// Good for frequent errors that would flood logs
logger.SampledError("db-timeout", 10, 
    "Database timeout (showing 1 out of 10 occurrences)")
```

### Custom Configuration

```go
// Create a new logger
loggerv1 := logger.NewLogger()

// Add outputs
loggerv1.AddOutput(logger.NewConsoleOutput(os.Stdout, logger.FormatText))

fileOutput, err := loggerv1.NewFileOutput("/var/log/app.log", logger.FormatJSON, 100)
if err == nil {
    loggerv1.AddOutput(fileOutput)
}

// Set as the default logger
logger.SetDefaultLogger(loggerv1)

// Set global level
logger.SetLevel(logger.LevelDebug)

// Set component-specific levels
logger.SetComponentLevel("network", logger.LevelVerbose)
logger.SetComponentLevel("database", logger.LevelInfo)
```

## Design Decisions and Best Practices

### When to Use Each Log Level

- **Emergency/Alert/Critical**: Severe issues that require immediate attention, likely to page someone
- **Error**: Problems that prevented an operation from completing successfully
- **Warning**: Issues that didn't prevent operation but indicate potential problems
- **Notice**: Important events that aren't errors but should be highlighted
- **Info**: General operational information, key lifecycle events
- **Debug**: Information useful when troubleshooting
- **Verbose**: Detailed internal operations, useful for deep debugging
- **Trace**: Extremely detailed tracing, typically voluminous

### Log Content Best Practices

1. **Be specific and actionable**
   - Include enough context to understand and resolve issues
   - Include IDs that can correlate logs across services

2. **Structure consistently**
   - Use verb phrases for actions ("User created", "Connection refused")
   - Put variable data in fields, not message strings when possible

3. **Avoid sensitive information**
   - No passwords, tokens, or personal data
   - Truncate or mask sensitive values

4. **Include relevant context**
   - Request IDs, user IDs, correlation IDs
   - Timing information
   - Key performance metrics

### Using Component Loggers

For larger applications, create separate loggers for major components:

```go
// In network package
var loggerv1 = logger.GetLogger().With("network")

// In database package
var loggerv1 = logger.GetLogger().With("database")
```

This allows setting different verbosity levels for each component:

```go
// Increase only network logging during troubleshooting
logger.GetLogger().SetComponentLevel("network", logger.LevelTrace)
```

## Performance Considerations

- **Test with production log volumes**: Benchmark your application with realistic logging rates
- **Buffer appropriately**: Set queue sizes appropriate for peak load
- **Watch for memory usage**: High-volume logging can consume significant memory
- **Consider log rotation**: Prevent disk space issues with proper rotation settings
- **Sample high-volume logs**: Use `SampledInfo` etc. for extremely frequent events

## License

[Your License Here]
