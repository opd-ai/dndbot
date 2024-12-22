package dndbot

import (
	"fmt"
	"os"

	"github.com/opd-ai/horde"
)

type HordeClient struct {
	*horde.Client
}

func NewHordeClient() *HordeClient {
	hc := &HordeClient{
		Client: horde.NewClient(os.Getenv("HORDE_API_KEY")),
	}
	return hc
}

func (c *HordeClient) ImageGenerate(prompt string, steps, width, height int, modelName string, progress progressor) ([]byte, error) {
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
	if modelName == "" {
		modelName = horde.DefaultModel
		pr.UpdateOutput(fmt.Sprintf("Using default model: %s", modelName))
	}
	pr.UpdateOutput(fmt.Sprintf("Starting image generation: prompt=%q, steps=%d, width=%d, height=%d",
		prompt, steps, width, height))

	// Create generation request
	req := horde.GenerationRequest{
		Prompt: prompt,
		Params: horde.Params{
			Steps:     steps,
			Width:     width,
			Height:    height,
			ModelName: modelName,
		},
	}

	// Request generation
	pr.UpdateOutput("Submitting generation request...")
	resp, err := c.RequestGeneration(req)
	if err != nil {
		return nil, fmt.Errorf("requesting generation: %w", err)
	}
	pr.UpdateOutput(fmt.Sprintf("Request accepted, got response: %v", resp))
	pr.UpdateOutput(fmt.Sprintf("Request accepted, got ID: %s", resp.ID))

	// Wait for completion
	pr.UpdateOutput("Waiting for generation to complete...")
	status, err := c.WaitForCompletion(resp.ID)
	if err != nil {
		return nil, fmt.Errorf("waiting for completion: %w", err)
	}

	pr.UpdateOutput(fmt.Sprintf("Status: %v", status.Generation[0].Image))
	// Verify we have results
	/*if len(status.Generation) != 0 {
		return nil, fmt.Errorf("no results returned")
	}*/

	// Download the image
	pr.UpdateOutput("Downloading generated image...")
	imageData, err := c.DownloadImage(status.Generation[0].Image)
	if err != nil {
		return nil, fmt.Errorf("downloading image: %w", err)
	}
	pr.UpdateOutput(fmt.Sprintf("Successfully downloaded image: %d bytes", len(imageData)))

	return imageData, nil
}
