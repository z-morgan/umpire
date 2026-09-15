package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zmorgan/umpire/skills"
)

// runInstallSkillCmd drives the subcommand the way a user would, resetting the
// package-level flag vars first so one test's --force doesn't leak into the next.
func runInstallSkillCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	flagProject = false
	flagForce = false

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append([]string{"install-skill"}, args...))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err := rootCmd.Execute()
	return out.String(), err
}

func TestInstallSkillWritesToHomeByDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	out, err := runInstallSkillCmd(t)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".claude", "skills", "umpire", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != skills.UmpireSKILL {
		t.Error("installed file does not match the embedded skill")
	}
	if !strings.Contains(out, path) {
		t.Errorf("output %q should name the destination %q", out, path)
	}
}

func TestInstallSkillProjectFlagWritesToWorkingDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	t.Chdir(repo)

	if _, err := runInstallSkillCmd(t, "--project"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(repo, ".claude", "skills", "umpire", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); !os.IsNotExist(err) {
		t.Error("--project should not touch the home directory")
	}
}

func TestInstallSkillRefusesToClobberWithoutForce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".claude", "skills", "umpire", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runInstallSkillCmd(t); err == nil {
		t.Fatal("expected an error when SKILL.md already exists")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hand-edited\n" {
		t.Error("existing file should be left alone")
	}
}

func TestInstallSkillForceOverwrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".claude", "skills", "umpire", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runInstallSkillCmd(t, "--force"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != skills.UmpireSKILL {
		t.Error("--force should replace the file with the embedded skill")
	}
}
