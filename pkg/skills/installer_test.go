package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()
	expected := []string{"claude", "copilot", "cursor", "opencode"}
	if len(agents) != len(expected) {
		t.Fatalf("expected %d agents, got %d", len(expected), len(agents))
	}
	for i, name := range expected {
		if agents[i] != name {
			t.Errorf("agent[%d] = %q, want %q", i, agents[i], name)
		}
	}
}

func TestFindAgent(t *testing.T) {
	tests := []struct {
		name  string
		found bool
	}{
		{"claude", true},
		{"copilot", true},
		{"cursor", true},
		{"opencode", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, ok := FindAgent(tt.name)
			if ok != tt.found {
				t.Errorf("FindAgent(%q) found = %v, want %v", tt.name, ok, tt.found)
			}
			if ok && agent.Name != tt.name {
				t.Errorf("FindAgent(%q) returned agent.Name = %q", tt.name, agent.Name)
			}
		})
	}
}

func TestDetectAgent(t *testing.T) {
	// Each subtest clears ALL agent env vars to ensure full isolation.
	allEnvVars := []string{
		"CLAUDECODE", "CURSOR_AGENT", "GITHUB_COPILOT", "OPENCODE",
		"CODEIUM_AGENT", "TABNINE_AGENT", "AMAZON_Q", "AI_AGENT",
	}

	clearAllEnvVars := func(t *testing.T) {
		t.Helper()
		for _, env := range allEnvVars {
			t.Setenv(env, "")
		}
	}

	t.Run("no agent detected", func(t *testing.T) {
		clearAllEnvVars(t)
		_, detected := DetectAgent()
		if detected {
			t.Error("expected no agent detected")
		}
	})

	t.Run("detects claude", func(t *testing.T) {
		clearAllEnvVars(t)
		t.Setenv("CLAUDECODE", "1")
		agent, detected := DetectAgent()
		if !detected {
			t.Fatal("expected agent detected")
		}
		if agent.Name != "claude" {
			t.Errorf("expected claude, got %q", agent.Name)
		}
	})

	t.Run("detects opencode", func(t *testing.T) {
		clearAllEnvVars(t)
		t.Setenv("OPENCODE", "1")
		agent, detected := DetectAgent()
		if !detected {
			t.Fatal("expected agent detected")
		}
		if agent.Name != "opencode" {
			t.Errorf("expected opencode, got %q", agent.Name)
		}
	})

	t.Run("detects copilot", func(t *testing.T) {
		clearAllEnvVars(t)
		t.Setenv("GITHUB_COPILOT", "1")
		agent, detected := DetectAgent()
		if !detected {
			t.Fatal("expected agent detected")
		}
		if agent.Name != "copilot" {
			t.Errorf("expected copilot, got %q", agent.Name)
		}
	})

	t.Run("detects cursor", func(t *testing.T) {
		clearAllEnvVars(t)
		t.Setenv("CURSOR_AGENT", "1")
		agent, detected := DetectAgent()
		if !detected {
			t.Fatal("expected agent detected")
		}
		if agent.Name != "cursor" {
			t.Errorf("expected cursor, got %q", agent.Name)
		}
	})
}

func TestRender(t *testing.T) {
	for _, agent := range AllAgents() {
		t.Run(agent.Name, func(t *testing.T) {
			content, err := Render(agent)
			if err != nil {
				t.Fatalf("Render(%s) error: %v", agent.Name, err)
			}
			if content == "" {
				t.Errorf("Render(%s) returned empty content", agent.Name)
			}
			// Should NOT contain unrendered template syntax
			if strings.Contains(content, "{{.Version}}") {
				t.Errorf("Render(%s) contains unrendered template: {{.Version}}", agent.Name)
			}
		})
	}
}

func TestRenderWithData(t *testing.T) {
	agent, _ := FindAgent("claude")
	data := TemplateData{Version: "1.2.3"}

	content, err := RenderWithData(agent, data)
	if err != nil {
		t.Fatalf("RenderWithData error: %v", err)
	}

	if !strings.Contains(content, "1.2.3") {
		t.Error("rendered content should contain custom version 1.2.3")
	}
}

func TestRenderWithData_UnknownAgent(t *testing.T) {
	fakeAgent := Agent{
		Name:        "nonexistent",
		DisplayName: "Nonexistent",
		Template:    "hello {{.Version}}",
	}
	_, err := RenderWithData(fakeAgent, TemplateData{Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected error for unknown agent template")
	}
	if !strings.Contains(err.Error(), "no parsed template") {
		t.Errorf("expected 'no parsed template' error, got: %v", err)
	}
}

func TestInstall(t *testing.T) {
	t.Run("installs to project directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("claude")

		result, err := Install(agent, tmpDir, false, false)
		if err != nil {
			t.Fatalf("Install error: %v", err)
		}

		if result.Replaced {
			t.Error("expected Replaced=false for new install")
		}
		if result.Global {
			t.Error("expected Global=false")
		}

		expectedPath := filepath.Join(tmpDir, ".claude", "commands", "dtctl.md")
		if result.Path != expectedPath {
			t.Errorf("Path = %q, want %q", result.Path, expectedPath)
		}

		// Verify file exists and has content
		data, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("failed to read installed file: %v", err)
		}
		if len(data) == 0 {
			t.Error("installed file is empty")
		}
		if !strings.Contains(string(data), "dtctl") {
			t.Error("installed file should contain 'dtctl'")
		}
	})

	t.Run("refuses overwrite without --force", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("copilot")

		// First install
		_, err := Install(agent, tmpDir, false, false)
		if err != nil {
			t.Fatalf("first Install error: %v", err)
		}

		// Second install should fail
		_, err = Install(agent, tmpDir, false, false)
		if err == nil {
			t.Fatal("expected error on duplicate install")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error should mention 'already exists', got: %v", err)
		}
		if !strings.Contains(err.Error(), "--force") {
			t.Errorf("error should mention '--force', got: %v", err)
		}
	})

	t.Run("overwrites with force", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("cursor")

		// First install
		_, err := Install(agent, tmpDir, false, false)
		if err != nil {
			t.Fatalf("first Install error: %v", err)
		}

		// Second install with overwrite
		result, err := Install(agent, tmpDir, false, true)
		if err != nil {
			t.Fatalf("overwrite Install error: %v", err)
		}
		if !result.Replaced {
			t.Error("expected Replaced=true on overwrite")
		}
	})

	t.Run("global install unsupported agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("copilot")

		_, err := Install(agent, tmpDir, true, false)
		if err == nil {
			t.Fatal("expected error for unsupported global install")
		}
		if !strings.Contains(err.Error(), "does not support global") {
			t.Errorf("error should mention 'does not support global', got: %v", err)
		}
	})

	t.Run("installs all agents", func(t *testing.T) {
		tmpDir := t.TempDir()

		for _, agent := range AllAgents() {
			result, err := Install(agent, tmpDir, false, false)
			if err != nil {
				t.Fatalf("Install(%s) error: %v", agent.Name, err)
			}

			// Verify file exists
			if _, err := os.Stat(result.Path); err != nil {
				t.Errorf("Install(%s) file not found at %s", agent.Name, result.Path)
			}
		}
	})
}

func TestUninstall(t *testing.T) {
	t.Run("removes installed file", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("claude")

		// Install first
		result, _ := Install(agent, tmpDir, false, false)

		// Verify it exists
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("file should exist before uninstall")
		}

		// Uninstall
		removed, err := Uninstall(agent, tmpDir)
		if err != nil {
			t.Fatalf("Uninstall error: %v", err)
		}
		if len(removed) != 1 {
			t.Fatalf("expected 1 removed, got %d", len(removed))
		}

		// Verify it's gone
		if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
			t.Error("file should not exist after uninstall")
		}
	})

	t.Run("returns empty for non-installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("copilot")

		removed, err := Uninstall(agent, tmpDir)
		if err != nil {
			t.Fatalf("Uninstall error: %v", err)
		}
		if len(removed) != 0 {
			t.Errorf("expected 0 removed, got %d", len(removed))
		}
	})

	t.Run("returns removed paths alongside error on partial failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("claude")

		// Install project-local, then verify uninstall returns it
		Install(agent, tmpDir, false, false)
		removed, err := Uninstall(agent, tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(removed) != 1 {
			t.Errorf("expected 1 removed, got %d", len(removed))
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("installed project-local", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("cursor")

		// Install
		Install(agent, tmpDir, false, false)

		result := Status(agent, tmpDir)
		if !result.Installed {
			t.Error("expected Installed=true")
		}
		if result.Global {
			t.Error("expected Global=false")
		}
		if result.Path == "" {
			t.Error("expected non-empty Path")
		}
	})

	t.Run("not installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		agent, _ := FindAgent("opencode")

		result := Status(agent, tmpDir)
		if result.Installed {
			t.Error("expected Installed=false")
		}
	})
}

func TestStatusAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Install one agent
	agent, _ := FindAgent("claude")
	Install(agent, tmpDir, false, false)

	results := StatusAll(tmpDir)
	if len(results) != len(AllAgents()) {
		t.Fatalf("expected %d results, got %d", len(AllAgents()), len(results))
	}

	installedCount := 0
	for _, r := range results {
		if r.Installed {
			installedCount++
			if r.Agent.Name != "claude" {
				t.Errorf("unexpected installed agent: %s", r.Agent.Name)
			}
		}
	}
	if installedCount != 1 {
		t.Errorf("expected 1 installed, got %d", installedCount)
	}
}

func TestAgentPaths(t *testing.T) {
	// Verify each agent has the expected file path conventions
	tests := []struct {
		name     string
		pathPart string
	}{
		{"claude", ".claude/commands/dtctl.md"},
		{"copilot", ".github/instructions/dtctl.instructions.md"},
		{"cursor", ".cursor/rules/dtctl.mdc"},
		{"opencode", ".opencode/commands/dtctl.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, ok := FindAgent(tt.name)
			if !ok {
				t.Fatalf("agent %q not found", tt.name)
			}
			// Use filepath to compare (handles OS-specific separators)
			expectedPath := filepath.FromSlash(tt.pathPart)
			if agent.ProjectPath != expectedPath {
				t.Errorf("ProjectPath = %q, want %q", agent.ProjectPath, expectedPath)
			}
		})
	}
}

func TestTemplateContent(t *testing.T) {
	// Verify templates have reasonable content
	for _, agent := range AllAgents() {
		t.Run(agent.Name+" template not empty", func(t *testing.T) {
			if agent.Template == "" {
				t.Errorf("%s template is empty", agent.Name)
			}
		})

		t.Run(agent.Name+" template has version placeholder", func(t *testing.T) {
			if !strings.Contains(agent.Template, "{{.Version}}") {
				t.Errorf("%s template missing {{.Version}} placeholder", agent.Name)
			}
		})

		t.Run(agent.Name+" template mentions dtctl", func(t *testing.T) {
			if !strings.Contains(agent.Template, "dtctl") {
				t.Errorf("%s template should mention dtctl", agent.Name)
			}
		})

		t.Run(agent.Name+" template mentions --agent flag", func(t *testing.T) {
			if !strings.Contains(agent.Template, "--agent") {
				t.Errorf("%s template should mention --agent flag", agent.Name)
			}
		})
	}
}

func TestParsedTemplatesInitialized(t *testing.T) {
	// Verify that all templates were parsed at init time
	for _, agent := range AllAgents() {
		t.Run(agent.Name, func(t *testing.T) {
			if _, ok := parsedTemplates[agent.Name]; !ok {
				t.Errorf("template for %s not found in parsedTemplates", agent.Name)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	t.Run("project-local path", func(t *testing.T) {
		agent, _ := FindAgent("claude")
		path, err := resolvePath(agent, "/tmp/project", false)
		if err != nil {
			t.Fatalf("resolvePath error: %v", err)
		}
		expected := filepath.Join("/tmp/project", ".claude", "commands", "dtctl.md")
		if path != expected {
			t.Errorf("path = %q, want %q", path, expected)
		}
	})

	t.Run("global path for supported agent", func(t *testing.T) {
		agent, _ := FindAgent("claude")
		path, err := resolvePath(agent, "/tmp/project", true)
		if err != nil {
			t.Fatalf("resolvePath error: %v", err)
		}
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".claude", "commands", "dtctl.md")
		if path != expected {
			t.Errorf("path = %q, want %q", path, expected)
		}
	})

	t.Run("global path for unsupported agent", func(t *testing.T) {
		agent, _ := FindAgent("copilot")
		_, err := resolvePath(agent, "/tmp/project", true)
		if err == nil {
			t.Fatal("expected error for unsupported global path")
		}
		if !strings.Contains(err.Error(), "does not support global") {
			t.Errorf("error should mention 'does not support global', got: %v", err)
		}
	})
}
