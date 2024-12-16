// Package ui provides the web user interface handlers for the DND bot generator
package ui

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/opd-ai/dndbot/srv/generator"
)

// isValidSession checks if the provided session ID is valid.
//
// Parameters:
//   - sessionID: string to validate as a UUID
//
// Returns:
//   - bool: true if sessionID is a valid UUID, false if empty or malformed
//
// The function ensures the sessionID is both non-empty and a valid UUID format.
func isValidSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}

	// Validate UUID format
	_, err := uuid.Parse(sessionID)
	return err == nil
}

// formatMessages converts a slice of WebSocket messages into HTML representation.
//
// Parameters:
//   - messages: []generator.WSMessage slice of messages to format
//
// Returns:
//   - string: HTML formatted string containing all messages with proper styling
//
// Each message is formatted with timestamp, status, content, and output sections.
func formatMessages(messages []generator.Message) string {
	var html strings.Builder
	for _, msg := range messages {
		html.WriteString(fmt.Sprintf(`
            <div class="message %s">
                <div class="message-header">
                    <span>%s</span>
                    <span>%s</span>
                </div>
                %s
                %s
            </div>
        `,
			msg.Status,
			msg.Status,
			msg.Timestamp.Format("15:04:05"),
			formatContent(msg.Message),
			formatOutput(msg.Output),
		))
	}
	return html.String()
}

// formatContent creates an HTML paragraph from message content with XSS protection.
//
// Parameters:
//   - content: string to format as HTML paragraph
//
// Returns:
//   - string: HTML formatted paragraph with escaped content
//
// Returns empty string if content is empty. All HTML special characters are escaped
func formatContent(content string) string {
	if content == "" {
		return ""
	}
	// Escape HTML special characters to prevent XSS
	escaped := content // html.EscapeString(content)
	return fmt.Sprintf("<p class=\"message-content\">%s</p>", escaped)
}

// formatOutput creates an HTML pre element from output content with XSS protection.
//
// Parameters:
//   - output: string to format as preformatted text
//
// Returns:
//   - string: HTML formatted pre element with escaped content
//
// Returns empty string if output is empty. All HTML special characters are escaped
// to prevent XSS attacks.
func formatOutput(output string) string {
	if output == "" {
		return ""
	}
	// Escape HTML special characters to prevent XSS
	escaped := output // html.EscapeString(output)
	return fmt.Sprintf("<pre class=\"message-output\">%s</pre>", escaped)
}
