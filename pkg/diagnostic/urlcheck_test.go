package diagnostic

import (
	"strings"
	"testing"
)

func TestCheckEnvironmentURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantProblems int
		wantContains string // substring that should appear in the first problem message
		wantSuggURL  string // expected suggested URL (first problem)
	}{
		// --- Correct URLs (no problems) ---
		{
			name:         "correct SaaS apps URL",
			url:          "https://abc12345.apps.dynatrace.com",
			wantProblems: 0,
		},
		{
			name:         "correct dev apps URL",
			url:          "https://deve2e.dev.apps.dynatracelabs.com",
			wantProblems: 0,
		},
		{
			name:         "correct sprint apps URL",
			url:          "https://hard12345.sprint.apps.dynatracelabs.com",
			wantProblems: 0,
		},
		{
			name:         "empty URL",
			url:          "",
			wantProblems: 0,
		},
		{
			name:         "non-dynatrace URL",
			url:          "https://api.example.com",
			wantProblems: 0,
		},

		// --- Wrong URLs (should produce problems) ---
		{
			name:         "live.dynatrace.com instead of apps",
			url:          "https://abc12345.live.dynatrace.com",
			wantProblems: 1,
			wantContains: "live.dynatrace.com",
			wantSuggURL:  "https://abc12345.apps.dynatrace.com",
		},
		{
			name:         "live.dynatrace.com with path",
			url:          "https://abc12345.live.dynatrace.com/api/v2",
			wantProblems: 1,
			wantContains: "live.dynatrace.com",
			wantSuggURL:  "https://abc12345.apps.dynatrace.com/api/v2",
		},
		{
			name:         "dev without apps",
			url:          "https://deve2e.dev.dynatracelabs.com",
			wantProblems: 1,
			wantContains: "dev.dynatracelabs.com",
			wantSuggURL:  "https://deve2e.dev.apps.dynatracelabs.com",
		},
		{
			name:         "sprint without apps",
			url:          "https://hard12345.sprint.dynatracelabs.com",
			wantProblems: 1,
			wantContains: "sprint.dynatracelabs.com",
			wantSuggURL:  "https://hard12345.sprint.apps.dynatracelabs.com",
		},
		{
			name:         "bare dynatrace.com domain",
			url:          "https://abc12345.dynatrace.com",
			wantProblems: 1,
			wantContains: "bare",
			wantSuggURL:  "https://abc12345.apps.dynatrace.com",
		},
		{
			name:         "managed URL with /e/ path",
			url:          "https://myhost.example.com/e/abc12345",
			wantProblems: 1,
			wantContains: "Managed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := CheckEnvironmentURL(tt.url)

			if len(problems) != tt.wantProblems {
				t.Errorf("CheckEnvironmentURL(%q) returned %d problems, want %d",
					tt.url, len(problems), tt.wantProblems)
				for i, p := range problems {
					t.Logf("  problem[%d]: %s (suggested: %s)", i, p.Message, p.SuggestedURL)
				}
				return
			}

			if tt.wantProblems > 0 && tt.wantContains != "" {
				if !strings.Contains(problems[0].Message, tt.wantContains) {
					t.Errorf("problem message %q should contain %q", problems[0].Message, tt.wantContains)
				}
			}

			if tt.wantProblems > 0 && tt.wantSuggURL != "" {
				if problems[0].SuggestedURL != tt.wantSuggURL {
					t.Errorf("SuggestedURL = %q, want %q", problems[0].SuggestedURL, tt.wantSuggURL)
				}
			}
		})
	}
}

func TestURLSuggestions(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		wantSuggestions bool
		wantContains    []string
	}{
		{
			name:            "correct URL has no suggestions",
			url:             "https://abc12345.apps.dynatrace.com",
			wantSuggestions: false,
		},
		{
			name:            "wrong URL produces suggestions with correction",
			url:             "https://abc12345.live.dynatrace.com",
			wantSuggestions: true,
			wantContains:    []string{"Possible wrong environment URL", "Did you mean", "apps.dynatrace.com"},
		},
		{
			name:            "suggestions include fix command",
			url:             "https://deve2e.dev.dynatracelabs.com",
			wantSuggestions: true,
			wantContains:    []string{"dtctl ctx set", "dev.apps.dynatracelabs.com"},
		},
		{
			name:            "managed URL has no suggested fix URL",
			url:             "https://myhost.example.com/e/abc12345",
			wantSuggestions: true,
			wantContains:    []string{"Managed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := URLSuggestions(tt.url)

			if tt.wantSuggestions && len(suggestions) == 0 {
				t.Error("expected suggestions but got none")
			}
			if !tt.wantSuggestions && len(suggestions) > 0 {
				t.Errorf("expected no suggestions but got: %v", suggestions)
			}

			combined := strings.Join(suggestions, " ")
			for _, want := range tt.wantContains {
				if !strings.Contains(combined, want) {
					t.Errorf("suggestions should contain %q, got: %v", want, suggestions)
				}
			}
		})
	}
}

func TestFixDomain(t *testing.T) {
	tests := []struct {
		rawURL    string
		oldSuffix string
		newSuffix string
		want      string
	}{
		{
			rawURL:    "https://ABC12345.Live.Dynatrace.com/api",
			oldSuffix: ".live.dynatrace.com",
			newSuffix: ".apps.dynatrace.com",
			want:      "https://ABC12345.apps.dynatrace.com/api",
		},
		{
			rawURL:    "https://deve2e.DEV.dynatracelabs.com",
			oldSuffix: ".dev.dynatracelabs.com",
			newSuffix: ".dev.apps.dynatracelabs.com",
			want:      "https://deve2e.dev.apps.dynatracelabs.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.rawURL, func(t *testing.T) {
			got := fixDomain(tt.rawURL, tt.oldSuffix, tt.newSuffix)
			if got != tt.want {
				t.Errorf("fixDomain() = %q, want %q", got, tt.want)
			}
		})
	}
}
