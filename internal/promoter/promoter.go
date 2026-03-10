package promoter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/y0s3ph/gostrap/internal/config"
	"github.com/y0s3ph/gostrap/internal/differ"
	"github.com/y0s3ph/gostrap/internal/models"
)

type PromoteResult struct {
	AppName   string
	SourceEnv string
	TargetEnv string
	Copied    []string
	Skipped   []string
}

type PromoteOptions struct {
	RepoPath  string
	AppName   string
	SourceEnv string
	TargetEnv string
	DryRun    bool
}

// Preview returns the diff between source and target environments for the
// given app, so the user can review before promoting.
func Preview(opts PromoteOptions) ([]differ.DiffResult, error) {
	return differ.Diff(opts.RepoPath, opts.SourceEnv, opts.TargetEnv, opts.AppName)
}

// Promote copies overlay files from source environment to target environment
// for the specified application(s). App definitions are not modified since
// they already point to the correct environment path.
func Promote(opts PromoteOptions) ([]PromoteResult, error) {
	cfg, err := config.Load(opts.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if !envExists(cfg, opts.SourceEnv) {
		return nil, fmt.Errorf("source environment %q not found in config", opts.SourceEnv)
	}
	if !envExists(cfg, opts.TargetEnv) {
		return nil, fmt.Errorf("target environment %q not found in config", opts.TargetEnv)
	}

	apps, err := discoverApps(opts.RepoPath)
	if err != nil {
		return nil, err
	}

	if opts.AppName != "" {
		found := false
		for _, a := range apps {
			if a == opts.AppName {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("application %q not found in environments/base/", opts.AppName)
		}
		apps = []string{opts.AppName}
	}

	var results []PromoteResult
	for _, app := range apps {
		result, err := promoteApp(opts.RepoPath, opts.SourceEnv, opts.TargetEnv, app, opts.DryRun)
		if err != nil {
			return nil, fmt.Errorf("promoting %s: %w", app, err)
		}
		results = append(results, result)
	}

	return results, nil
}

func promoteApp(repoPath, sourceEnv, targetEnv, app string, dryRun bool) (PromoteResult, error) {
	result := PromoteResult{
		AppName:   app,
		SourceEnv: sourceEnv,
		TargetEnv: targetEnv,
	}

	srcDir := filepath.Join(repoPath, "environments", sourceEnv, app)
	tgtDir := filepath.Join(repoPath, "environments", targetEnv, app)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return result, fmt.Errorf("reading source overlay %s: %w", srcDir, err)
	}

	if !dryRun {
		if err := os.MkdirAll(tgtDir, 0755); err != nil {
			return result, fmt.Errorf("creating target directory: %w", err)
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcFile := filepath.Join(srcDir, entry.Name())
		tgtFile := filepath.Join(tgtDir, entry.Name())
		relPath := filepath.Join("environments", targetEnv, app, entry.Name())

		srcContent, err := os.ReadFile(srcFile)
		if err != nil {
			return result, fmt.Errorf("reading %s: %w", srcFile, err)
		}

		if fileContentEqual(tgtFile, srcContent) {
			result.Skipped = append(result.Skipped, relPath)
			continue
		}

		if !dryRun {
			if err := os.WriteFile(tgtFile, srcContent, 0644); err != nil {
				return result, fmt.Errorf("writing %s: %w", tgtFile, err)
			}
		}
		result.Copied = append(result.Copied, relPath)
	}

	return result, nil
}

func fileContentEqual(path string, content []byte) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if len(existing) != len(content) {
		return false
	}
	for i := range existing {
		if existing[i] != content[i] {
			return false
		}
	}
	return true
}

func envExists(cfg *models.BootstrapConfig, name string) bool {
	for _, e := range cfg.Environments {
		if e.Name == name {
			return true
		}
	}
	return false
}

func discoverApps(repoPath string) ([]string, error) {
	baseDir := filepath.Join(repoPath, "environments", "base")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", baseDir, err)
	}

	var apps []string
	for _, e := range entries {
		if e.IsDir() {
			apps = append(apps, e.Name())
		}
	}
	return apps, nil
}

