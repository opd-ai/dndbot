/**
 * Configuration for WebSocket connection and UI elements
 */
const CONFIG = {
    WEBSOCKET: {
        MAX_RETRIES: 5,
        RETRY_DELAY_MS: 1000,
        COMPLETION_MARKERS: ['complete', 'error'],
        CONNECTION_TIMEOUT: 10000 // 10 seconds
    },
    MESSAGES: {
        START: '🎲 Starting adventure generation...',
        ERROR: '❌ Error generating adventure',
        CONNECTION_FAILED: '❌ Connection failed. Please try again.',
        PROMPT_REQUIRED: 'Please enter details for your adventure',
        TIMEOUT: '⏱️ Connection timeout. Please try again.'
    }
};

/**
 * Handles UI-related operations
 */
const UI = {
    appendMessage(message) {
        const log = document.getElementById('progress-log');
        const messageElement = document.createElement('div');
        messageElement.textContent = message;
        log.appendChild(messageElement);
        log.scrollTop = log.scrollHeight;
    },

    enableGenerateButton() {
        document.getElementById('generate-btn').disabled = false;
    },

    clearMessages() {
        document.getElementById('progress-log').innerHTML = '';
    }
};

/**
 * Debounce function to prevent multiple rapid submissions
 */
function debounce(func, wait, options = {}) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            timeout = null;
            if (options.trailing) func.apply(this, args);
        };
        const callNow = options.leading && !timeout;
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
        if (callNow) func.apply(this, args);
    };
}

/**
 * Manages D&D adventure generation WebSocket communications
 */
class AdventureGenerator {
    #ws = null;
    #retryCount = 0;
    #sessionId = null;
    #connectionTimeout = null;

    /**
     * Initializes WebSocket connection for adventure generation
     * @param {string} sessionId - Unique session identifier
     */
    connect(sessionId) {
        if (!sessionId) {
            console.error('Invalid session ID');
            UI.appendMessage(CONFIG.MESSAGES.ERROR);
            UI.enableGenerateButton();
            return;
        }

        this.#sessionId = sessionId;
        this.#closeExistingConnection();

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/${sessionId}`;

        console.log(`Connecting to WebSocket with session ID: ${sessionId}`);

        this.#ws = new WebSocket(wsUrl);
        this.#setupEventHandlers();
        this.#setConnectionTimeout();
    }

    /**
     * Sets up WebSocket event handlers
     * @private
     */
    #setupEventHandlers() {
        this.#ws.onopen = () => {
            console.log('Adventure generator connected');
            this.#retryCount = 0;
            this.#clearConnectionTimeout();
        };

        this.#ws.onmessage = ({ data }) => {
            UI.appendMessage(data);

            if (CONFIG.WEBSOCKET.COMPLETION_MARKERS.some(marker => data.includes(marker))) {
                UI.enableGenerateButton();
                this.#closeExistingConnection();
            }
        };

        this.#ws.onerror = (error) => {
            console.error('Connection error:', error);
            this.#attemptReconnection();
        };

        this.#ws.onclose = () => {
            console.log('Connection closed');
            this.#attemptReconnection();
        };
    }

    /**
     * Sets a timeout for the WebSocket connection
     * @private
     */
    #setConnectionTimeout() {
        this.#clearConnectionTimeout();
        this.#connectionTimeout = setTimeout(() => {
            console.log('Connection timeout');
            UI.appendMessage(CONFIG.MESSAGES.TIMEOUT);
            UI.enableGenerateButton();
            this.#closeExistingConnection();
        }, CONFIG.WEBSOCKET.CONNECTION_TIMEOUT);
    }

    /**
     * Clears the connection timeout
     * @private
     */
    #clearConnectionTimeout() {
        if (this.#connectionTimeout) {
            clearTimeout(this.#connectionTimeout);
            this.#connectionTimeout = null;
        }
    }

    /**
     * Attempts to reconnect to the WebSocket server
     * @private
     */
    #attemptReconnection() {
        if (this.#retryCount < CONFIG.WEBSOCKET.MAX_RETRIES) {
            setTimeout(() => {
                this.#retryCount++;
                console.log(`Retry attempt ${this.#retryCount}/${CONFIG.WEBSOCKET.MAX_RETRIES}`);
                this.connect(this.#sessionId);
            }, CONFIG.WEBSOCKET.RETRY_DELAY_MS);
        } else {
            UI.appendMessage(CONFIG.MESSAGES.CONNECTION_FAILED);
            UI.enableGenerateButton();
            this.#clearConnectionTimeout();
        }
    }

    /**
     * Closes any existing WebSocket connection
     * @private
     */
    #closeExistingConnection() {
        if (this.#ws) {
            this.#ws.close();
            this.#ws = null;
        }
        this.#clearConnectionTimeout();
    }
}

// Initialize the adventure generator
const adventureGenerator = new AdventureGenerator();

/**
 * Starts the adventure generation process
 * @async
 */
async function startGeneration() {
    const generateButton = document.getElementById('generate-btn');
    const promptInput = document.getElementById('prompt-input');

    if (!promptInput.value.trim()) {
        alert(CONFIG.MESSAGES.PROMPT_REQUIRED);
        return;
    }

    // Prevent multiple submissions
    if (generateButton.disabled) {
        return;
    }

    generateButton.disabled = true;
    UI.clearMessages();
    UI.appendMessage(CONFIG.MESSAGES.START);

    try {
        const response = await fetch('/generate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `prompt=${encodeURIComponent(promptInput.value.trim())}`
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        if (!data.sessionId) {
            throw new Error('No session ID received from server');
        }

        adventureGenerator.connect(data.sessionId);
    } catch (error) {
        console.error('Generation error:', error);
        UI.appendMessage(CONFIG.MESSAGES.ERROR);
        UI.enableGenerateButton();
    }
}

// Wait for DOM to be fully loaded before adding event listeners
document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('generate-btn').addEventListener('click',
        debounce(startGeneration, 1000, { leading: true, trailing: false })
    );
});