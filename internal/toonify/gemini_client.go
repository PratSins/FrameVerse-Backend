package toonify

import (
	"context"
	"fmt"
	"os"
	"time"

	"bytes"
	"encoding/json"
	"google.golang.org/genai"
	"net/http"
)

type omniRequest struct {
	Model string `json:"model"`

	Input []omniInput `json:"input"`

	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`
}

type omniInput struct {
	Type     string `json:"type"`
	URI      string `json:"uri,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
	Text     string `json:"text,omitempty"`
}

type omniResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`

	Steps []struct {
		Type    string `json:"type"`
		Content []struct {
			Type     string `json:"type"`
			MIMEType string `json:"mime_type"`
			Data     string `json:"data"`
			URI      string `json:"uri"`
		} `json:"content"`
	} `json:"steps"`
}

type GeminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(
	ctx context.Context,
	apiKey string,
	model string,
) (*GeminiClient, error) {

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		return nil, fmt.Errorf("create Gemini client: %w", err)
	}

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

func (g *GeminiClient) Toonify(
	ctx context.Context,
	videoPath string,
	style string,
) (string, error) {

	// Upload the source video to Gemini Files API.
	videoFile, err := g.client.Files.UploadFromPath(
		ctx,
		videoPath,
		&genai.UploadFileConfig{
			MIMEType: "video/mp4",
		},
	)

	if err != nil {
		return "", fmt.Errorf("upload video to Gemini: %w", err)
	}

	// Wait for Gemini to process the uploaded video.
	for {
		if videoFile.State == "ACTIVE" {
			break
		}

		if videoFile.State == "FAILED" {
			return "", fmt.Errorf("Gemini failed to process input video")
		}

		time.Sleep(5 * time.Second)

		videoFile, err = g.client.Files.Get(
			ctx,
			videoFile.Name,
			nil,
		)

		if err != nil {
			return "", fmt.Errorf("check Gemini file status: %w", err)
		}
	}

	prompt := fmt.Sprintf(
		"Transform this video into %s style. "+
			"Preserve the original subject, actions, camera movement, "+
			"timing and composition as much as possible. "+
			"Keep everything else the same.",
		style,
	)

	reqBody := omniRequest{
		Model: g.model,

		Input: []omniInput{
			{
				Type:     "document",
				URI:      videoFile.URI,
				MIMEType: videoFile.MIMEType,
			},
			{
				Type: "text",
				Text: prompt,
			},
		},

		ResponseFormat: map[string]interface{}{
			"type":     "video",
			"delivery": "uri",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal Gemini request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://generativelanguage.googleapis.com/v1beta/interactions",
		bytes.NewReader(body),
	)

	if err != nil {
		return "", fmt.Errorf("create Gemini request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", os.Getenv("GEMINI_API_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf(
			"Gemini returned HTTP %d",
			resp.StatusCode,
		)
	}

	var result omniResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf(
			"decode Gemini response: %w",
			err,
		)
	}

	for _, step := range result.Steps {
		if step.Type != "model_output" {
			continue
		}

		for _, content := range step.Content {
			if content.Type == "video" {
				if content.URI != "" {
					return content.URI, nil
				}

				return content.Data, nil
			}
		}
	}

	return "", fmt.Errorf("Gemini response did not contain video output")
}
