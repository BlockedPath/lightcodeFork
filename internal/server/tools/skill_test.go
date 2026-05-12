package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testCtx = ToolContext{WorkingDirectory: ".", SessionID: "test-session"}

func TestSkill_MissingSkillName(t *testing.T) {
	resp, err := Skill(testCtx, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(resp.Content, "Error:") {
		t.Errorf("expected error message, got: %q", resp.Content)
	}
}

func TestSkill_SkillNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILL_PATH", dir)

	resp, err := Skill(testCtx, map[string]any{"skillName": "nonexistent"})
	if err == nil {
		t.Error("expected an error when SKILL.md is missing")
	}
	if resp.Content != "Skill not found" {
		t.Errorf("expected 'Skill not found', got: %q", resp.Content)
	}
}

func TestSkill_BasicSkill(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILL_PATH", dir)

	skillName := "mySKILL"
	skillDir := filepath.Join(dir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	skillMd := "---\nfrontmatter: true\n---\n# My Skill\nThis is the skill content."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644); err != nil {
		t.Fatal(err)
	}

	resp, err := Skill(testCtx, map[string]any{"skillName": skillName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp.Content, "<skill_content name=\""+skillName+"\"") {
		t.Errorf("expected skill_content tag with name, got: %q", resp.Content)
	}
	// The frontmatter block should be stripped from the leading skill_content section.
	// (It may still appear raw inside <skill_files> since that embeds the file as-is.)
	skillContentEnd := strings.Index(resp.Content, "<skill_files>")
	if skillContentEnd == -1 {
		skillContentEnd = len(resp.Content)
	}
	mainContent := resp.Content[:skillContentEnd]
	if strings.Contains(mainContent, "frontmatter: true") {
		t.Errorf("frontmatter block should be stripped from main skill content, got: %q", mainContent)
	}
	if !strings.Contains(resp.Content, "This is the skill content.") {
		t.Errorf("expected skill body text in output, got: %q", resp.Content)
	}
}

func TestSkill_IncludesAdditionalFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILL_PATH", dir)

	skillName := "withFiles"
	skillDir := filepath.Join(dir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "helper.py"), []byte("print('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	resp, err := Skill(testCtx, map[string]any{"skillName": skillName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp.Content, "<skill_files>") {
		t.Errorf("expected <skill_files> block in output, got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "helper.py") {
		t.Errorf("expected helper.py to appear in skill_files, got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "print('hello')") {
		t.Errorf("expected file content in skill_files, got: %q", resp.Content)
	}
}

func TestSkill_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILL_PATH", dir)

	skillName := "withSubdir"
	skillDir := filepath.Join(dir, skillName)
	subDir := filepath.Join(skillDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	resp, err := Skill(testCtx, map[string]any{"skillName": skillName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(resp.Content, "subdir") {
		t.Errorf("subdirectories should not appear in skill_files, got: %q", resp.Content)
	}
}
