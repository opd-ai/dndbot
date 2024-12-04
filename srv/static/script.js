let ws = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;
const reconnectDelay = 1000; // 1 second

function connectWebSocket(sessionId) {
    if (ws !== null) {
        ws.close();
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/${sessionId}`;

    console.log(`Connecting to WebSocket at ${wsUrl}`);
    
    ws = new WebSocket(wsUrl);

    ws.onopen = function() {
        console.log('WebSocket connected');
        reconnectAttempts = 0;
    };

    ws.onmessage = function(event) {
        const message = event.data;
        appendMessage(message);
        
        if (message.includes('complete') || message.includes('error')) {
            enableGenerateButton();
        }
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        if (reconnectAttempts < maxReconnectAttempts) {
            setTimeout(() => {
                reconnectAttempts++;
                console.log(`Attempting to reconnect (${reconnectAttempts}/${maxReconnectAttempts})`);
                connectWebSocket(sessionId);
            }, reconnectDelay);
        } else {
            appendMessage('❌ Connection failed. Please try again.');
            enableGenerateButton();
        }
    };

    ws.onclose = function() {
        console.log('WebSocket closed');
        if (reconnectAttempts < maxReconnectAttempts) {
            setTimeout(() => {
                reconnectAttempts++;
                console.log(`Attempting to reconnect (${reconnectAttempts}/${maxReconnectAttempts})`);
                connectWebSocket(sessionId);
            }, reconnectDelay);
        }
    };
}

async function startGeneration() {
    const generateButton = document.getElementById('generate-btn');
    const prompt = document.getElementById('prompt-input').value;

    if (!prompt) {
        alert('Please enter a prompt');
        return;
    }

    generateButton.disabled = true;
    clearMessages();
    appendMessage('🚀 Starting generation...');

    try {
        const response = await fetch('/generate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `prompt=${encodeURIComponent(prompt)}`
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        connectWebSocket(data.sessionId);
    } catch (error) {
        console.error('Error:', error);
        appendMessage('❌ Error starting generation');
        enableGenerateButton();
    }
}

function appendMessage(message) {
    const output = document.getElementById('progress-log');
    const messageDiv = document.createElement('div');
    messageDiv.textContent = message;
    output.appendChild(messageDiv);
    output.scrollTop = output.scrollHeight;
}

function enableGenerateButton() {
    const generateButton = document.getElementById('generate-btn');
    generateButton.disabled = false;
}

function clearMessages() {
    const output = document.getElementById('progress-log');
    output.innerHTML = '';
}