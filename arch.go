// main.go
package main

import (
	"net/http"
)

// Configuration struct for API and other settings
type Config struct {
	APIKey     string
	OutputDir  string
	MaxRetries int
}

// Adventure represents the complete story structure
type Adventure struct {
	Title           string
	Episodes        []Episode
	TableOfContents string
	OriginalPrompt  string
}

// Episode represents a single adventure episode
type Episode struct {
	Title          string
	Summary        string
	Tagline        string
	Characters     []string
	Location       string
	OnePageDungeon string
	FullAdventure  string
	Illustrations  []IllustrationPrompt
}

// IllustrationPrompt represents a Stable Diffusion prompt
type IllustrationPrompt struct {
	Description string
	Style       string
	IsMap       bool
}

// ClaudeClient handles API communication
type ClaudeClient struct {
	apiKey     string
	httpClient *http.Client
}

// ClaudeRequest represents the API request structure
type ClaudeRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the API response structure
type ClaudeResponse struct {
	Content string `json:"content"`
}
