package ui

import (
	"fmt"
	"html"
	"strings"

	"github.com/google/uuid"
	"github.com/opd-ai/dndbot/srv/generator"
)

// Add this helper function
func isValidSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}

	// Validate UUID format
	_, err := uuid.Parse(sessionID)
	return err == nil
}

// formatMessages formats a slice of WebSocket messages into HTML
func formatMessages(messages []generator.WSMessage) string {
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

// formatContent formats the message content with proper HTML escaping
func formatContent(content string) string {
	if content == "" {
		return ""
	}
	// Escape HTML special characters to prevent XSS
	escaped := html.EscapeString(content)
	return fmt.Sprintf("<p class=\"message-content\">%s</p>", escaped)
}

// formatOutput formats the output content with proper HTML escaping
func formatOutput(output string) string {
	if output == "" {
		return ""
	}
	// Escape HTML special characters to prevent XSS
	escaped := html.EscapeString(output)
	return fmt.Sprintf("<pre class=\"message-output\">%s</pre>", escaped)
}
