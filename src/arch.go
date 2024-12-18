package dndbot

import "strings"

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
	Covers          []IllustrationPrompt
	Setting         string "PROMPT.md"
	Style           string "STYLE.md"
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

func (e *Episode) Text() string {
	val := "## " + e.Title + "\n"
	val += "Summary" + e.Summary + "\n"
	val += "Tagline: " + e.Tagline + "\n"
	val += "Location: " + e.Location + "\n"
	val += "Characters: " + strings.Join(e.Characters, ",") + "\n"
	return val
}

// IllustrationPrompt represents a Stable Diffusion prompt
type IllustrationPrompt struct {
	Description string
	Style       string
	IsMap       bool
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
