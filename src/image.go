package dndbot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/opd-ai/horde"
)

type ImageClient interface {
	ImageGenerate(prompt string, steps, width, height int, modelName string, progress progressor) ([]byte, error)
}

type LocalClient struct{}

// SDWebUIRequest represents the request structure for the Stable Diffusion WebUI API
// SDWebUIRequest represents the request structure for the Stable Diffusion WebUI API
type SDWebUIRequest struct {
	Prompt           string                 `json:"prompt"`
	NegativePrompt   string                 `json:"negative_prompt,omitempty"`
	Steps            int                    `json:"steps"`
	Width            int                    `json:"width"`
	Height           int                    `json:"height"`
	CFGScale         float64                `json:"cfg_scale,omitempty"`
	BatchSize        int                    `json:"batch_size,omitempty"`
	OverrideSettings map[string]interface{} `json:"override_settings,omitempty"`
}

// SDWebUIResponse represents the response structure from the Stable Diffusion WebUI API
type SDWebUIResponse struct {
	Images []string `json:"images"`
	Info   string   `json:"info"`
	Error  string   `json:"error,omitempty"`
}

func (l *LocalClient) ImageGenerate(prompt string, steps, width, height int, modelName string, progress progressor) ([]byte, error) {
	var pr progressor
	if progress != nil {
		pr = progress
	} else {
		pr = &nullProgressor{}
	}

	// Apply defaults and log them
	if steps == 0 {
		steps = horde.DefaultSteps
		pr.UpdateOutput(fmt.Sprintf("Using default steps: %d", steps))
	}
	if width == 0 {
		width = horde.DefaultWidth
		pr.UpdateOutput(fmt.Sprintf("Using default width: %d", width))
	}
	if height == 0 {
		height = horde.DefaultHeight
		pr.UpdateOutput(fmt.Sprintf("Using default height: %d", height))
	}

	pr.UpdateOutput(fmt.Sprintf("Starting image generation: prompt=%q, steps=%d, width=%d, height=%d",
		prompt, steps, width, height))

	sdWebUIURL := os.Getenv("SD_WEBUI_URL")
	if sdWebUIURL == "" {
		return nil, fmt.Errorf("SD_WEBUI_URL environment variable not set")
	}
	pr.UpdateOutput(fmt.Sprintf("Using local SD-WebUI URL: %s", sdWebUIURL))

	// Prepare the request payload
	requestData := SDWebUIRequest{
		Prompt:    prompt,
		Steps:     steps,
		Width:     width,
		Height:    height,
		CFGScale:  3.0,
		BatchSize: 1,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute, // SD generation can take a while
	}

	// Prepare the request
	req, err := http.NewRequest("POST", sdWebUIURL+"/sdapi/v1/txt2img", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	pr.UpdateOutput("Sending request to SD-WebUI...")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var sdResponse SDWebUIResponse
	if err := json.Unmarshal(body, &sdResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if we got any images
	if len(sdResponse.Images) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	// Convert base64 image to bytes
	imageBytes, err := base64.StdEncoding.DecodeString(sdResponse.Images[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	pr.UpdateOutput("Image generation completed successfully")
	return imageBytes, nil
}
