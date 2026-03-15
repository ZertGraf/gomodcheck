package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	cloneTimeout = 2 * time.Minute
	listTimeout  = 5 * time.Minute
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repoURL, err := parseArgs()
	if err != nil {
		return err
	}

	if err := ensureDeps(); err != nil {
		return err
	}

	dir, cleanup, err := cloneToTemp(repoURL)
	if err != nil {
		return err
	}
	defer cleanup()

	mod, err := readModuleInfo(dir)
	if err != nil {
		return err
	}

	fmt.Printf("module:     %s\n", mod.Module.Path)
	fmt.Printf("go version: %s\n", mod.Go.Version)
	fmt.Println()

	updates, err := findUpdates(dir)
	if err != nil {
		return err
	}

	printUpdates(updates)
	return nil
}

func parseArgs() (string, error) {
	if len(os.Args) < 2 {
		return "", fmt.Errorf("usage: gomodcheck <repo-url>")
	}
	return os.Args[1], nil
}

func ensureDeps() error {
	for _, bin := range []string{"git", "go"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not found in PATH", bin)
		}
	}
	return nil
}

// execOutput runs an external command with context and returns stdout.
// centralizes timeout detection and stderr extraction.
func execOutput(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}

	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("%s: timed out", name)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return nil, fmt.Errorf("%s: %s", name, strings.TrimSpace(string(exitErr.Stderr)))
	}

	return nil, fmt.Errorf("%s: %w", name, err)
}

func cloneToTemp(repoURL string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "gomodcheck-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: cleanup failed: %v\n", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloneTimeout)
	defer cancel()

	if _, err := execOutput(ctx, "", nil, "git", "clone", "--depth=1", "--single-branch", repoURL, tmpDir); err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

type goModInfo struct {
	Module struct {
		Path string `json:"Path"`
	} `json:"Module"`
	Go struct {
		Version string `json:"Version"`
	} `json:"Go"`
}

func readModuleInfo(dir string) (*goModInfo, error) {
	out, err := execOutput(context.Background(), dir, nil, "go", "mod", "edit", "-json")
	if err != nil {
		return nil, err
	}

	var info goModInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	if info.Module.Path == "" {
		return nil, fmt.Errorf("module directive not found in go.mod")
	}
	if info.Go.Version == "" {
		info.Go.Version = "unknown"
	}

	return &info, nil
}

type moduleEntry struct {
	Path    string      `json:"Path"`
	Version string      `json:"Version"`
	Main    bool        `json:"Main,omitempty"`
	Update  *updateInfo `json:"Update,omitempty"`
}

type updateInfo struct {
	Version string `json:"Version"`
}

type dependency struct {
	Path    string
	Current string
	Latest  string
}

func findUpdates(dir string) ([]dependency, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	out, err := execOutput(ctx, dir, append(os.Environ(), "GOFLAGS=-mod=mod"), "go", "list", "-m", "-u", "-json", "all")
	if err != nil {
		return nil, err
	}

	return parseGoListOutput(out)
}

func parseGoListOutput(data []byte) ([]dependency, error) {
	var deps []dependency

	dec := json.NewDecoder(strings.NewReader(string(data)))
	for dec.More() {
		var mod moduleEntry
		if err := dec.Decode(&mod); err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}

		if mod.Main || mod.Update == nil {
			continue
		}

		deps = append(deps, dependency{
			Path:    mod.Path,
			Current: mod.Version,
			Latest:  mod.Update.Version,
		})
	}

	return deps, nil
}

func printUpdates(updates []dependency) {
	if len(updates) == 0 {
		fmt.Println("all dependencies are up to date")
		return
	}

	maxPath := 0
	for _, u := range updates {
		if len(u.Path) > maxPath {
			maxPath = len(u.Path)
		}
	}

	fmt.Printf("updatable dependencies (%d):\n\n", len(updates))
	for _, u := range updates {
		fmt.Printf("  %-*s  %s -> %s\n", maxPath, u.Path, u.Current, u.Latest)
	}
}
