package gemini

import "net/http"

type GeminiClient struct {
	httpClient *http.Client
	projectID  string
	location   string
	model      string
}

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
