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

type omniRequest struct {
	Model            string               `json:"model"`
	Input            []omniInput          `json:"input"`
	ResponseFormat   []omniResponseFormat `json:"response_format,omitempty"`
	GenerationConfig *generationConfig    `json:"generation_config,omitempty"`
}

type omniInput struct {
	Type     string `json:"type"`
	URI      string `json:"uri,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
	Text     string `json:"text,omitempty"`
}

type omniResponseFormat struct {
	Type     string `json:"type"`
	Delivery string `json:"delivery"`
	GCSURI   string `json:"gcs_uri,omitempty"`
}

type generationConfig struct {
	VideoConfig *videoConfig `json:"video_config,omitempty"`
}

type videoConfig struct {
	Task string `json:"task"`
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
	httpClient *http.Client
	projectID  string
	location   string
	model      string
}

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

const CommonPromptSuffix = `This is a strict pixel-aligned edit of the source video: keep the same pose, motion, timing, clothing colors, and background.

The camera must not change — no zoom, no crop, no recentering, and no change to the field of view.

The person's face and body must stay at exactly the same position and size in the frame as the source.

Eyes, nose, and mouth must remain at the same screen coordinates in every frame.

Match the facial expression exactly, frame by frame.

Preserve the exact degree of mouth openness at every moment — if the mouth is slightly open and still, keep it slightly open and still; do not close it, and do not add talking or any mouth movement that is not in the source.

Mirror blinks, gaze direction, and eyebrow position at the same moments as the source.

Change only the visual style, nothing about the geometry, composition, or performance.`

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
