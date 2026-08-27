package test

import (
	"strings"
	"testing"

	"github.com/PratSins/FrameVerse-Backend/pkg/gemini"
)

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name               string
		style              string
		expectedLead       string
		mustContainKeyword string
	}{
		{
			name:               "Anime style",
			style:              "anime",
			expectedLead:       "Redraw the video as a hand-drawn anime with clean line art, cel shading, and vibrant colors.",
			mustContainKeyword: "hand-drawn anime",
		},
		{
			name:               "Anime style default empty",
			style:              "",
			expectedLead:       "Redraw the video as a hand-drawn anime with clean line art, cel shading, and vibrant colors.",
			mustContainKeyword: "hand-drawn anime",
		},
		{
			name:               "3D style",
			style:              "3d",
			expectedLead:       "Transform the person into a 3D animated movie character (stylized CGI animation look, expressive big eyes, soft lighting).",
			mustContainKeyword: "3D animated movie character",
		},
		{
			name:               "3D animated style",
			style:              "3d_animated",
			expectedLead:       "Transform the person into a 3D animated movie character (stylized CGI animation look, expressive big eyes, soft lighting).",
			mustContainKeyword: "3D animated movie character",
		},
		{
			name:               "Pixar style",
			style:              "pixar",
			expectedLead:       "Transform the person into a 3D animated movie character (stylized CGI animation look, expressive big eyes, soft lighting).",
			mustContainKeyword: "3D animated movie character",
		},
		{
			name:               "Custom style",
			style:              "cyberpunk",
			expectedLead:       "Redraw the video in cyberpunk style.",
			mustContainKeyword: "cyberpunk",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prompt := gemini.BuildPrompt(tc.style)

			if !strings.HasPrefix(prompt, tc.expectedLead) {
				t.Errorf("expected prompt to start with %q, got %q", tc.expectedLead, prompt)
			}

			if !strings.Contains(prompt, tc.mustContainKeyword) {
				t.Errorf("expected prompt to contain %q", tc.mustContainKeyword)
			}

			if !strings.Contains(prompt, gemini.CommonPromptSuffix) {
				t.Errorf("expected prompt to contain CommonPromptSuffix")
			}

			if !strings.Contains(prompt, "strict pixel-aligned edit of the source video") {
				t.Errorf("expected prompt to contain pixel-aligned instruction")
			}

			if !strings.Contains(prompt, "Change only the visual style, nothing about the geometry, composition, or performance.") {
				t.Errorf("expected prompt to end with performance instruction")
			}
		})
	}
}
