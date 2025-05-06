// Simulate API activity with various log levels
import { configureLogger, simulateAPIActivity, simulateNetworkActivity, simulateCacheActivity } from "./sample.mjs";

// Main function to run the example
async function main() {
    const logger = configureLogger();

    // Start activity simulators
    simulateNetworkActivity(logger);
    simulateCacheActivity(logger);
    simulateAPIActivity(logger);

    // Set up signal handling for graceful shutdown
    process.on('SIGINT', async () => {
        logger.notice('Received SIGINT, shutting down');

        // Flush logs before exit
        await logger.flush();
        logger.close();
        process.exit(0);
    });

    process.on('SIGTERM', async () => {
        logger.notice('Received SIGTERM, shutting down');

        // Flush logs before exit
        await logger.flush();
        logger.close();
        process.exit(0);
    });

    logger.info('All simulators started, press Ctrl+C to exit');
}

// Run the application
main().catch(err => {
    console.error('Application error:', err);
    process.exit(1);
});
