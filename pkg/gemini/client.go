package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func NewGeminiClient(
	ctx context.Context,
	projectID string,
	location string,
	model string,
) (*GeminiClient, error) {
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("create google token source: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, ts)

	if projectID == "" {
		projectID = "swift-delight-441118-c4"
	}
	if location == "" {
		location = "global"
	}
	if model == "" {
		model = "gemini-omni-flash-preview"
	}

	return &GeminiClient{
		httpClient: httpClient,
		projectID:  projectID,
		location:   location,
		model:      model,
	}, nil
}

func BuildPrompt(style string) string {
	var stylePrompt string
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "3d", "3d_animated", "3d_animation", "3d_movie", "pixar":
		stylePrompt = "Transform the person into a 3D animated movie character (stylized CGI animation look, expressive big eyes, soft lighting)."
	case "anime", "":
		stylePrompt = "Redraw the video as a hand-drawn anime with clean line art, cel shading, and vibrant colors."
	default:
		stylePrompt = fmt.Sprintf("Redraw the video in %s style.", style)
	}

	return fmt.Sprintf("%s\n\n%s", stylePrompt, CommonPromptSuffix)
}

func (g *GeminiClient) Toonify(
	ctx context.Context,
	inputGCSURI string,
	outputGCSURI string,
	style string,
) (string, error) {
	prompt := BuildPrompt(style)

	reqBody := omniRequest{
		Model: g.model,
		Input: []omniInput{
			{
				Type:     "video",
				URI:      inputGCSURI,
				MIMEType: "video/mp4",
			},
			{
				Type: "text",
				Text: prompt,
			},
		},
		ResponseFormat: []omniResponseFormat{
			{
				Type:     "video",
				Delivery: "uri",
				GCSURI:   outputGCSURI,
			},
		},
		GenerationConfig: &generationConfig{
			VideoConfig: &videoConfig{
				Task: "edit",
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal Vertex AI request: %w", err)
	}

	endpoint := fmt.Sprintf(
		"https://aiplatform.googleapis.com/v1beta1/projects/%s/locations/%s/interactions",
		g.projectID,
		g.location,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create Vertex AI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Vertex AI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf(
			"Vertex AI returned HTTP %d: %s",
			resp.StatusCode,
			string(errBody),
		)
	}

	var result omniResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Vertex AI response: %w", err)
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

	return "", fmt.Errorf("Vertex AI response did not contain video output")
}
