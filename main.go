package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	modName, goVer, err := readModuleInfo(dir)
	if err != nil {
		return err
	}

	fmt.Printf("module:     %s\n", modName)
	fmt.Printf("go version: %s\n", goVer)
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

	cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", repoURL, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git clone: %s", strings.TrimSpace(string(out)))
	}

	return tmpDir, cleanup, nil
}

func readModuleInfo(dir string) (string, string, error) {
	modPath := filepath.Join(dir, "go.mod")

	if _, err := os.Stat(modPath); err != nil {
		return "", "", fmt.Errorf("go.mod not found in repository root")
	}

	return parseGoMod(modPath)
}

func parseGoMod(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("open go.mod: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: close go.mod: %v\n", err)
		}
	}()

	var modName, goVer string

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		if strings.HasPrefix(line, "module ") {
			modName = strings.Trim(strings.TrimPrefix(line, "module "), `"`)
		}
		if strings.HasPrefix(line, "go ") {
			goVer = strings.TrimPrefix(line, "go ")
		}
	}

	if err := sc.Err(); err != nil {
		return "", "", fmt.Errorf("read go.mod: %w", err)
	}
	if modName == "" {
		return "", "", fmt.Errorf("module directive not found in go.mod")
	}
	if goVer == "" {
		goVer = "unknown"
	}

	return modName, goVer, nil
}

type dependency struct {
	Path    string
	Current string
	Latest  string
}

func findUpdates(dir string) ([]dependency, error) {
	cmd := exec.Command("go", "list", "-m", "-u", "all")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("go list: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("go list: %w", err)
	}

	return parseGoListOutput(out)
}

func parseGoListOutput(data []byte) ([]dependency, error) {
	var deps []dependency

	sc := bufio.NewScanner(strings.NewReader(string(data)))
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}

		line := sc.Text()
		if !strings.Contains(line, "[") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		deps = append(deps, dependency{
			Path:    fields[0],
			Current: fields[1],
			Latest:  strings.Trim(fields[2], "[]"),
		})
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("parse output: %w", err)
	}

	return deps, nil
}

func printUpdates(updates []dependency) {
	if len(updates) == 0 {
		fmt.Println("all dependencies are up to date")
		return
	}

	fmt.Printf("updatable dependencies (%d):\n\n", len(updates))
	for _, u := range updates {
		fmt.Printf("  %-50s %s -> %s\n", u.Path, u.Current, u.Latest)
	}
}
