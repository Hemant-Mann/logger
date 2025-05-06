/**
   * Example application demonstrating the advanced logging system (ESM Version)
   */

import { fileURLToPath } from 'url';
import path from 'path';
import {
    Level,
    Format,
    Logger,
    ConsoleOutput,
    FileOutput,
    createLogger,
    LEVEL_NAMES
} from '../src/logger.js'; // Path to the logging library

// Get current file directory (ESM equivalent of __dirname)
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Parse command line arguments
const args = process.argv.slice(2).reduce((acc, arg) => {
    if (arg.startsWith('--')) {
        const [key, value] = arg.slice(2).split('=');
        acc[key] = value !== undefined ? value : true;
    }
    return acc;
}, {});

// Configure log level
function parseLogLevel(level) {
    const levelMap = {
        'emergency': Level.EMERGENCY,
        'emerg': Level.EMERGENCY,
        'alert': Level.ALERT,
        'critical': Level.CRITICAL,
        'crit': Level.CRITICAL,
        'error': Level.ERROR,
        'err': Level.ERROR,
        'warning': Level.WARNING,
        'warn': Level.WARNING,
        'notice': Level.NOTICE,
        'info': Level.INFO,
        'debug': Level.DEBUG,
        'verbose': Level.VERBOSE,
        'verb': Level.VERBOSE,
        'trace': Level.TRACE
    };

    return levelMap[level?.toLowerCase()] !== undefined
        ? levelMap[level.toLowerCase()]
        : Level.INFO;
}

// Component names
const COMPONENT = {
    NETWORK: 'network',
    CACHE: 'cache',
    API: 'api'
};

// Create and configure the logger
export function configureLogger() {
    const logLevel = parseLogLevel(args.logLevel || 'info');
    const netLogLevel = args.netLogLevel ? parseLogLevel(args.netLogLevel) : undefined;
    const logFormat = args.logFormat === 'json' ? Format.JSON : Format.TEXT;
    const enableTrace = args.trace === true;

    // Create the logger
    const logger = createLogger({
        level: logLevel,
        defaultFields: {
            app: 'example-service',
            version: '1.0.0'
        }
    });

    // Add console output
    logger.addOutput(new ConsoleOutput(process.stdout, logFormat));

    // Add file output if requested
    if (args.logFile) {
        const fileOutput = new FileOutput(args.logFile, logFormat, {
            maxSizeMB: args.logRotateSize || 100,
            compress: true
        });

        // Set up log rotation callback
        fileOutput.setRotateCallback((rotatedPath) => {
            console.log(`Log rotated to ${rotatedPath}`);
        });

        logger.addOutput(fileOutput);
    }

    // Set component-specific log level if provided
    if (netLogLevel !== undefined) {
        logger.setComponentLevel(COMPONENT.NETWORK, netLogLevel);
    }

    // Convert logLevel number to name for display
    let logLevelName = 'INFO';
    for (const [key, value] of Object.entries(Level)) {
        if (value === logLevel) {
            logLevelName = key;
            break;
        }
    }

    // Log startup information
    logger.info(`Starting example application with log level ${logLevelName}`);

    if (enableTrace) {
        logger.info('Trace logging enabled - this will generate a lot of output!');
    } else {
        logger.info('Trace logging disabled - use --trace to enable');
    }

    return logger;
}

export function simulateAPIActivity(logger) {
    const apiLogger = logger.with(COMPONENT.API);

    const endpoints = ['/api/users', '/api/products', '/api/orders', '/api/auth'];
    const methods = ['GET', 'POST', 'PUT', 'DELETE'];

    setInterval(() => {
        const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
        const method = methods[Math.floor(Math.random() * methods.length)];
        const requestID = Math.floor(Math.random() * 0xFFFFFFFF).toString(16).padStart(8, '0');
        let statusCode = 200;

        // Occasionally generate error status codes
        if (Math.random() < 0.1) {
            const statusCodes = [400, 401, 403, 404, 500, 503];
            statusCode = statusCodes[Math.floor(Math.random() * statusCodes.length)];
        }

        // Basic request logging (always at Info level)
        apiLogger.info(`API request: ${method} ${endpoint} (id=${requestID}, status=${statusCode})`);

        // Add context-specific fields for this request
        const requestLogger = apiLogger.withFields({
            request_id: requestID,
            method: method,
            endpoint: endpoint
        });

        // Log request details at debug level
        requestLogger.debug(`Request details: client_ip=192.168.1.${Math.floor(Math.random() * 255)}, user_agent=Mozilla/5.0...`);

        // Log timing information at verbose level
        requestLogger.verbose(`Request timing: auth=${Math.floor(Math.random() * 5) + 1}ms, db=${Math.floor(Math.random() * 20) + 5}ms, render=${Math.floor(Math.random() * 10) + 1}ms, total=${Math.floor(Math.random() * 50) + 10}ms`);

        // Log very detailed information at trace level
        requestLogger.trace(`Request trace: db_queries=${Math.floor(Math.random() * 10) + 1}, cache_hits=${Math.floor(Math.random() * 20)}, cache_misses=${Math.floor(Math.random() * 5)}`);

        // Handle errors with appropriate log levels
        if (statusCode >= 400) {
            if (statusCode >= 500) {
                requestLogger.error(`Server error: status=${statusCode}, error=Internal server error: database connection failed`, {
                    error_code: statusCode,
                    stack_trace: 'Error: Database connection failed\n    at Connection.connect (/app/db.js:42:11)\n    at ApiHandler.getUser (/app/handlers.js:157:23)'
                });
            } else {
                requestLogger.warning(`Client error: status=${statusCode}, error=Invalid input parameter 'id' must be numeric`, {
                    error_code: statusCode,
                    validation_errors: ['id must be numeric', 'id is required']
                });
            }
        }
    }, 300);
}

// Simulate network activity with various log levels
export function simulateNetworkActivity(logger) {
    const netLogger = logger.with(COMPONENT.NETWORK);

    setInterval(() => {
        // Simulate a connection
        const connId = Math.floor(Math.random() * 1000);

        netLogger.info(`New connection established (id=${connId})`);

        // Log detailed info at debug level
        netLogger.debug(`Connection details: local=127.0.0.1:8080, remote=192.168.1.${Math.floor(Math.random() * 255)}:45678`);

        // Log very detailed info at verbose level
        netLogger.verbose('Socket options set: TCP_NODELAY=1, SO_KEEPALIVE=1, SO_RCVBUF=65536');

        // Log ultra-detailed info at trace level
        netLogger.trace(`TCP handshake completed in ${Math.floor(Math.random() * 100)} ms, initial cwnd=${Math.floor(Math.random() * 10) + 1}`);

        // Occasionally simulate errors (1 in 10 chance)
        if (Math.random() < 0.1) {
            netLogger.error(`Connection error: timeout waiting for response from client (id=${connId})`, {
                connection_id: connId,
                error_type: 'timeout',
                duration_ms: 30000
            });

            // Rate-limited high-volume error (1 per 5 occurrences)
            netLogger.sampledError('conn-reset', 5, 'Connection reset by peer (sampled 1:5)', {
                connection_id: connId,
                socket_state: 'established'
            });
        }
    }, 500);
}

// Simulate cache activity with various log levels
export function simulateCacheActivity(logger) {
    const cacheLogger = logger.with(COMPONENT.CACHE).withFields({
        cache_type: 'lru',
        max_size: '1024MB'
    });

    const keys = ['user:123', 'product:456', 'session:789', 'settings:app'];

    setInterval(() => {
        const key = keys[Math.floor(Math.random() * keys.length)];
        const hitRate = 75.0 + Math.random() * 10.0; // 75-85% hit rate

        // Basic operational logging
        cacheLogger.info(`Cache stats: items=${Math.floor(Math.random() * 1000) + 5000}, hit_rate=${hitRate.toFixed(2)}%`);

        // Debug info about specific operations
        cacheLogger.debug(`Cache lookup for key=${key}`);

        // Verbose logging for cache internals
        if (Math.random() < 0.33) {
            cacheLogger.verbose(`Cache eviction: removed ${Math.floor(Math.random() * 10) + 1} items, oldest_key=${keys[Math.floor(Math.random() * keys.length)]}:expired`);
        }

        // Trace level for extremely detailed debugging
        cacheLogger.trace(`Cache key hash calculation: key=${key}, hash=0x${Math.floor(Math.random() * 0xFFFFFFFF).toString(16)}, bucket=${Math.floor(Math.random() * 64)}`);

        // Occasionally log warnings about cache performance
        if (Math.random() < 0.125) {
            // Rate-limited warning (log only 1 in every 3)
            cacheLogger.sampledWarning('cache-pressure', 3,
                `Cache pressure detected: load_factor=${(0.8 + Math.random() * 0.15).toFixed(2)}, eviction_rate=${Math.floor(Math.random() * 100) + 50}/sec`);
        }
    }, 800);
}