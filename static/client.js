/**
 * Logger class for structured application logging
 */
class Logger {
    static LOG_LEVELS = {
        DEBUG: 0,
        INFO: 1,
        WARN: 2,
        ERROR: 3
    };

    constructor(context, minLevel = 'DEBUG') {
        this.context = context;
        this.minLevel = Logger.LOG_LEVELS[minLevel];
        this.startTime = Date.now();
    }

    /**
     * Creates timestamp for log entries
     * @returns {string} Formatted timestamp
     */
    getTimestamp() {
        return new Date().toISOString();
    }

    /**
     * Formats log message with context
     * @param {string} level Log level
     * @param {string} message Log message
     * @param {Object} data Additional data to log
     * @returns {Object} Formatted log entry
     */
    formatLog(level, message, data = {}) {
        return {
            timestamp: this.getTimestamp(),
            level,
            context: this.context,
            message,
            data,
            timeElapsed: `${Date.now() - this.startTime}ms`
        };
    }

    debug(message, data) {
        if (this.minLevel <= Logger.LOG_LEVELS.DEBUG) {
            console.debug(this.formatLog('DEBUG', message, data));
        }
    }

    info(message, data) {
        if (this.minLevel <= Logger.LOG_LEVELS.INFO) {
            console.info(this.formatLog('INFO', message, data));
        }
    }

    warn(message, data) {
        if (this.minLevel <= Logger.LOG_LEVELS.WARN) {
            console.warn(this.formatLog('WARN', message, data));
        }
    }

    error(message, error, data = {}) {
        if (this.minLevel <= Logger.LOG_LEVELS.ERROR) {
            console.error(this.formatLog('ERROR', message, {
                ...data,
                error: {
                    message: error.message,
                    stack: error.stack,
                    name: error.name
                }
            }));
        }
    }
}

/**
 * Core API client for D&D Adventure Generator
 * Handles all HTTP interactions with the server
 */
class DndApiClient {
    constructor(baseUrl = '/') {
        this.baseUrl = baseUrl;
        this.logger = new Logger('DndApiClient');
        this.sessionId = this.getStoredSessionId();
        this.logger.info('API Client initialized', { baseUrl, sessionId: this.sessionId });
    }

    getStoredSessionId() {
        this.logger.debug('Retrieving stored session ID');
        const sessionId = document.cookie.split('; ')
            .find(row => row.startsWith('session_id='))
            ?.split('=')[1] || null;
        this.logger.debug('Session ID retrieved', { sessionId });
        return sessionId;
    }

    async generateAdventure(prompt) {
        this.logger.info('Generating adventure', { prompt });
        try {
            const response = await fetch(`${this.baseUrl}generate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                    'X-Session-Id': this.sessionId
                },
                credentials: 'include',
                body: `prompt=${encodeURIComponent(prompt)}`
            });

            this.logger.debug('Generation response received', {
                status: response.status,
                headers: Object.fromEntries(response.headers)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            this.sessionId = response.headers.get('X-Session-Id');
            const result = await response.text();
            this.logger.info('Adventure generated successfully', {
                newSessionId: this.sessionId,
                responseLength: result.length
            });
            return result;
        } catch (error) {
            this.logger.error('Generation failed', error, { prompt });
            throw error;
        }
    }

    async getMessageHistory(sessionId = this.sessionId) {
        this.logger.debug('Fetching message history', { sessionId });
        try {
            const response = await fetch(`${this.baseUrl}api/messages/${sessionId}`, {
                credentials: 'include'
            });
            const result = await response.text();
            this.logger.debug('Message history retrieved', {
                sessionId,
                responseLength: result.length
            });
            return result;
        } catch (error) {
            this.logger.error('Failed to fetch message history', error, { sessionId });
            throw error;
        }
    }
}

/**
 * UI Manager for D&D Adventure Generator
 */
class DndGeneratorUI {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.logger = new Logger('DndGeneratorUI');
        this.logger.info('UI Manager initialized');
        this.initializeUI();
    }

    initializeUI() {
        this.logger.debug('Initializing UI elements');
        this.elements = {
            form: document.getElementById('generator-form'),
            prompt: document.getElementById('prompt-input'),
            output: document.getElementById('output-area'),
            status: document.getElementById('status-message')
        };

        // Log if any elements are missing
        Object.entries(this.elements).forEach(([key, element]) => {
            if (!element) {
                this.logger.warn(`UI element not found: ${key}`);
            }
        });

        this.elements.form.addEventListener('submit', (e) => this.handleSubmit(e));
        this.logger.info('UI initialization complete');
    }

    async handleSubmit(event) {
        event.preventDefault();
        const prompt = this.elements.prompt.value.trim();
        this.logger.info('Form submitted', { promptLength: prompt.length });

        if (!prompt) {
            this.logger.warn('Empty prompt submitted');
            this.showError('Please enter a prompt');
            return;
        }

        try {
            this.setLoading(true);
            this.logger.debug('Starting adventure generation');
            const result = await this.apiClient.generateAdventure(prompt);
            this.updateOutput(result);
            this.startPolling();
            this.logger.info('Adventure generation completed');
        } catch (error) {
            this.logger.error('Form submission failed', error);
            this.handleError(error);
        } finally {
            this.setLoading(false);
        }
    }

    updateOutput(content) {
        this.logger.debug('Updating output', { contentLength: content.length });
        this.elements.output.innerHTML = content;
    }

    showError(message) {
        this.logger.warn('Showing error message', { message });
        this.elements.status.textContent = message;
        this.elements.status.className = 'error';
    }

    setLoading(isLoading) {
        this.logger.debug('Setting loading state', { isLoading });
        this.elements.form.classList.toggle('loading', isLoading);
        this.elements.prompt.disabled = isLoading;
    }

    async startPolling() {
        this.logger.info('Starting message polling');
        let pollCount = 0;
        const pollInterval = setInterval(async () => {
            try {
                pollCount++;
                this.logger.debug('Polling for updates', { pollCount });
                const history = await this.apiClient.getMessageHistory();
                this.updateOutput(history);
            } catch (error) {
                this.logger.error('Polling failed', error, { pollCount });
                clearInterval(pollInterval);
            }
        }, 2000);

        setTimeout(() => {
            clearInterval(pollInterval);
            this.logger.info('Polling stopped after timeout', { totalPolls: pollCount });
        }, 300000);
    }
}

// Initialize with logging
document.addEventListener('DOMContentLoaded', () => {
    const logger = new Logger('Main');
    logger.info('Application starting');
    
    try {
        const apiClient = new DndApiClient();
        const ui = new DndGeneratorUI(apiClient);
        logger.info('Application initialized successfully');
    } catch (error) {
        logger.error('Failed to initialize application', error);
    }
});