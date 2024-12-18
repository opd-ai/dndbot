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
        this.logger.debug('Examining stored session ID: ' + document.cookie + '/');
        this.logger.debug('Examining cookies: ' + document.cookie + '/');
        this.logger.debug('Retrieving stored session ID: '+ document.cookie + '/');
        const sessionId = document.cookie.split('; ')
            .find(row => row.startsWith('session_id='))
            ?.split('=')[1] || null;
        this.logger.debug('Session ID retrieved', { sessionId });
        return sessionId;
    }

    async generateAdventure(prompt, setting, style) {
        this.logger.info('Generating adventure', { prompt });
        try {
            const response = await fetch(`${this.baseUrl}generate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                    'X-Session-Id': this.sessionId
                },
                credentials: 'include',
                body: `prompt=${encodeURIComponent(prompt)}&setting=${encodeURIComponent(setting)}&style=${encodeURIComponent(style)}`
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
        this.pollingState = {
            interval: null,
            isPaused: false,
            emptyResponseCount: 0,
            maxEmptyResponses: 3 // Number of empty responses before pausing
        };
        this.initializeUI();
    }

    initializeUI() {
        this.logger.debug('Initializing UI elements');
        this.elements = {
            form: document.getElementById('generator-form'),
            prompt: document.getElementById('prompt-input'),
            setting: document.getElementById('setting-input'),
            style: document.getElementById('style-input'),
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
        this.startPolling();
    }

    /**
     * Handles form submission with polling management
     * @param {Event} event Form submit event
     */
    async handleSubmit(event) {
        event.preventDefault();
        const prompt = this.elements.prompt.value.trim();
        const setting = this.elements.setting.value.trim();
        const style = this.elements.style.value.trim();
        this.logger.info('Form submitted', { promptLength: prompt.length });

        if (!prompt) {
            this.logger.warn('Empty prompt submitted');
            this.showError('Please enter a prompt');
            return;
        }

        try {
            this.setLoading(true);
            // Stop any existing polling
            this.stopPolling();
            
            this.logger.debug('Starting adventure generation');
            const result = await this.apiClient.generateAdventure(prompt, setting, style);
            this.updateOutput(result);
            
            // Reset polling state and start fresh
            this.pollingState.emptyResponseCount = 0;
            this.pollingState.isPaused = false;

            
            this.startPolling();
            
            this.logger.info('Adventure generation completed');
        } catch (error) {
            this.logger.error('Form submission failed', error);
            this.handleError(error);
        } finally {
            this.setLoading(false);
        }
    }

    handleError(error){
        this.logger.error("Error caught:", error)
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

    /**
     * Manages message polling with intelligent pause/resume
     * @returns {void}
     */
    async startPolling() {
        this.logger.info('Starting message polling');
        let pollCount = 0;
        
        // Clear any existing polling interval
        if (this.pollingState.interval) {
            clearInterval(this.pollingState.interval);
        }
        //TODO: figure out how to make these exectable
        nodeScriptReplace(document.getElementById("qr"));
        nodeScriptReplace(document.getElementById("btcqr"));

        const poll = async () => {
            try {
                pollCount++;
                this.logger.debug('Polling for updates', { 
                    pollCount,
                    isPaused: this.pollingState.isPaused,
                    emptyResponseCount: this.pollingState.emptyResponseCount
                });

                const history = await this.apiClient.getMessageHistory();
                
                // Check response content
                if (!history || history.length === 0) {
                    this.pollingState.emptyResponseCount++;
                    this.logger.debug('Empty response received', {
                        emptyResponseCount: this.pollingState.emptyResponseCount
                    });

                    // Pause polling if we've received too many empty responses
                    if (this.pollingState.emptyResponseCount >= this.pollingState.maxEmptyResponses) {
                        this.pausePolling();
                    }
                } else {
                    // Reset empty response counter and ensure polling is active
                    this.pollingState.emptyResponseCount = 0;
                    if (this.pollingState.isPaused) {
                        this.resumePolling();
                    }
                    this.updateOutput(history);
                }

            } catch (error) {
                this.logger.error('Polling failed', error, { pollCount });
                this.stopPolling();
            }
            window.scrollTo(0, document.body.scrollHeight);
        };

        // Initial poll
        await poll();

        // Set up polling interval
        this.pollingState.interval = setInterval(poll, 2000);

        // Set up polling timeout
        setTimeout(() => {
            this.stopPolling();
            this.logger.info('Polling stopped after timeout', { totalPolls: pollCount });
        }, 300000); // 5 minutes
    }

    /**
     * Pauses the polling loop
     */
    pausePolling() {
        if (!this.pollingState.isPaused) {
            this.logger.info('Pausing polling due to empty responses');
            this.pollingState.isPaused = true;
            clearInterval(this.pollingState.interval);
            this.pollingState.interval = null;
            
            // Update UI to show paused state
            this.elements.status.textContent = 'Generation paused...';
            this.elements.status.className = 'status-paused';
        }
    }

    /**
     * Resumes the polling loop
     */
    resumePolling() {
        if (this.pollingState.isPaused) {
            this.logger.info('Resuming polling');
            this.pollingState.isPaused = false;
            this.pollingState.emptyResponseCount = 0;
            this.startPolling(); // Restart polling loop
            
            // Update UI to show active state
            this.elements.status.textContent = 'Generating content...';
            this.elements.status.className = 'status-active';
        }
    }

    /**
     * Completely stops the polling loop
     */
    stopPolling() {
        this.logger.info('Stopping polling completely');
        if (this.pollingState.interval) {
            clearInterval(this.pollingState.interval);
            this.pollingState.interval = null;
        }
        this.pollingState.isPaused = false;
        this.pollingState.emptyResponseCount = 0;
        
        // Update UI to show completed state
        this.elements.status.textContent = 'Generation complete';
        this.elements.status.className = 'status-complete';
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

function nodeScriptReplace(node) {
    if (node === null) {
        return
    }
    if ( nodeScriptIs(node) === true ) {
            node.parentNode.replaceChild( nodeScriptClone(node) , node );
    }
    else {
            var i = -1, children = node.childNodes;
            while ( ++i < children.length ) {
                  nodeScriptReplace( children[i] );
            }
    }

    return node;
}
function nodeScriptClone(node){
    var script  = document.createElement("script");
    script.text = node.innerHTML;

    var i = -1, attrs = node.attributes, attr;
    while ( ++i < attrs.length ) {                                    
          script.setAttribute( (attr = attrs[i]).name, attr.value );
    }
    return script;
}

function nodeScriptIs(node) {
    return node.tagName === 'SCRIPT';
}
