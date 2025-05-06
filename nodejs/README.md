# Advanced Node.js Logging System

This package provides a comprehensive logging system for Node.js applications based on best practices from high-performance systems. It's designed for applications that need both efficient production logging and detailed debugging capabilities.

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

- **Runtime level control**: Global and per-component log levels
- **Message sampling**: Rate-limiting for high-volume log messages

### Multiple Output Destinations

- **Console output**: Human-readable text with ANSI colors or machine-parseable JSON
- **File output**: With automatic rotation based on size and optional compression
- **Extensible**: Implement the `Output` interface for custom destinations

### Rich Context and Structured Data

- **Automatic context**: File, line number, timestamp, and instance ID
- **Component tagging**: Identify which part of the system generated the log
- **Default fields**: Add global context to all log messages
- **Custom fields**: Add per-message structured data

### Performance Optimizations

- **Asynchronous logging**: Non-blocking log calls with buffered queue
- **Level check before processing**: Skip processing for disabled levels
- **Rate limiting**: Control logging frequency for high-volume events

## Installation

```bash
npm install advanced-logger
```

## Usage Examples

### Basic Usage

```javascript
const logger = require('advanced-logger');

// Use the default logger
logger.info('Application starting up');

// Log at different levels
logger.debug('This is a debug message');
logger.error('Something went wrong', { error: 'Connection failed', code: 500 });

// Use formatted strings (use template literals for better performance)
logger.info(`User ${username} logged in from ${ipAddress}`);
```

### Creating a Custom Logger

```javascript
const { createLogger, ConsoleOutput, FileOutput, Format, Level } = require('advanced-logger');

// Create a new logger with custom configuration
const logger = createLogger({
  level: Level.DEBUG,
  defaultFields: {
    service: 'user-api',
    version: '1.2.3'
  }
});

// Add outputs
logger.addOutput(new ConsoleOutput(process.stdout, Format.TEXT));
logger.addOutput(new FileOutput('/var/log/app.log', Format.JSON, {
  maxSizeMB: 100,
  compress: true
}));

// Use the logger
logger.info('Custom logger initialized');
```

### Component-Specific Logging

```javascript
// Create a component-specific logger
const networkLogger = logger.with('network');

// These logs will be tagged with the component
networkLogger.info('Listening on port 8080');
networkLogger.debug('Accepted connection from 192.168.1.25');

// Set component-specific log level
logger.setComponentLevel('network', Level.VERBOSE);
```

### Structured Logging

```javascript
// Add fields to a specific log
logger.info('User authenticated', {
  user_id: 123,
  role: 'admin',
  ip: '192.168.1.1'
});

// Create a logger with default fields
const userLogger = logger.withFields({
  user_id: 123,
  session_id: 'abc-123'
});

// All logs from this logger will include the fields
userLogger.info('User performed action');
userLogger.error('Permission denied');

// Add a single field
const requestLogger = logger.withField('request_id', 'req-abc-123');
```

### Rate-Limited Logging

```javascript
// Only log 1 out of every 100 occurrences
for (let i = 0; i < 1000; i++) {
  logger.sampledInfo('high-volume-event', 100, 
    `Processing item ${i}`);
}

// Good for frequent errors that would flood logs
logger.sampledError('db-timeout', 10, 
  'Database timeout (showing 1 out of 10 occurrences)');
```

## API Reference

### Log Levels

```javascript
const { Level } = require('advanced-logger');

// Available levels
Level.EMERGENCY // 0
Level.ALERT     // 1
Level.CRITICAL  // 2
Level.ERROR     // 3
Level.WARNING   // 4
Level.NOTICE    // 5
Level.INFO      // 6
Level.DEBUG     // 7
Level.VERBOSE   // 8
Level.TRACE     // 9
```

### Formats

```javascript
const { Format } = require('advanced-logger');

// Available formats
Format.TEXT // Human-readable text with colors (good for development)
Format.JSON // Machine-parseable JSON (good for production)
```

### Logger Methods

All logging methods accept an optional fields object as the second parameter.

```javascript
logger.emergency(message, [fields]);
logger.alert(message, [fields]);
logger.critical(message, [fields]);
logger.error(message, [fields]);
logger.warning(message, [fields]);
logger.notice(message, [fields]);
logger.info(message, [fields]);
logger.debug(message, [fields]);
logger.verbose(message, [fields]);
logger.trace(message, [fields]);

// Sampled logging methods
logger.sampledInfo(key, rate, message, [fields]);
logger.sampledError(key, rate, message, [fields]);
logger.sampledWarning(key, rate, message, [fields]);
logger.sampledDebug(key, rate, message, [fields]);

// Create derived loggers
logger.with(component);
logger.withField(key, value);
logger.withFields(fields);

// Configure logger
logger.setLevel(level);
logger.setComponentLevel(component, level);
logger.addOutput(output);

// Clean up
await logger.flush();
logger.close();
```

### Outputs

#### Console Output

```javascript
const { ConsoleOutput, Format } = require('advanced-logger');

// Default to stdout and text format
const defaultConsole = new ConsoleOutput();

// Customize stream and format
const jsonConsole = new ConsoleOutput(process.stderr, Format.JSON);
```

#### File Output

```javascript
const { FileOutput, Format } = require('advanced-logger');

// Simple file output
const fileOutput = new FileOutput('/var/log/app.log');

// Full configuration
const rotatingFile = new FileOutput('/var/log/app.log', Format.JSON, {
  maxSizeMB: 50,       // Rotate at 50MB
  compress: true       // Compress rotated logs with gzip
});

// Set rotation callback
rotatingFile.setRotateCallback((rotatedPath) => {
  console.log(`Log rotated to ${rotatedPath}`);
  // Could upload to S3, notify monitoring, etc.
});
```

#### Custom Output

```javascript
const { Output, Format } = require('advanced-logger');

class DatabaseOutput extends Output {
  constructor(dbConnection) {
    super(Format.JSON);
    this.db = dbConnection;
  }
  
  write(entry) {
    // Insert log into database
    this.db.collection('logs').insertOne({
      timestamp: entry.timestamp,
      level: entry.level,
      message: entry.message,
      component: entry.component,
      fields: entry.fields || {}
    }).catch(err => {
      console.error('Failed to write log to database:', err);
    });
  }
  
  close() {
    // No cleanup needed
  }
}
```

## Best Practices

### When to Use Each Log Level

- **Emergency/Alert/Critical**: Severe issues that require immediate attention
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
   - Put variable data in fields, not message strings

3. **Avoid sensitive information**
   - No passwords, tokens, or personal data
   - Truncate or mask sensitive values

4. **Include relevant context**
   - Request IDs, user IDs, correlation IDs
   - Timing information
   - Key performance metrics

### Using Component Loggers

For larger applications, create separate loggers for major components:

```javascript
// In network module
const logger = require('./logger').with('network');

// In database module
const logger = require('./logger').with('database');
```

This allows setting different verbosity levels for each component:

```javascript
// Increase only network logging during troubleshooting
logger.setComponentLevel('network', Level.TRACE);
```

## Performance Considerations

- **Use Template Literals**: Faster than string concatenation for messages
- **Check Level Before Expensive Operations**: Avoid building complex log data when it won't be logged
- **Use Sampling for High-Volume Logs**: Avoid overwhelming your logging system
- **Buffer Appropriately**: Set queue sizes appropriate for peak load
- **Configure Rotation**: Prevent disk space issues with proper rotation settings

## License

MIT License