package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/y0s3ph/gostrap/internal/models"
)

const fluxManifestURL = "https://github.com/fluxcd/flux2/releases/download/v%s/install.yaml"

type FluxInstaller struct {
	repoPath       string
	clusterContext string
	version        string
}

func NewFlux(cfg *models.BootstrapConfig) *FluxInstaller {
	return &FluxInstaller{
		repoPath:       cfg.RepoPath,
		clusterContext: cfg.ClusterContext,
		version:        cfg.Controller.Version,
	}
}

// Install deploys Flux controllers on the target cluster in two phases:
//  1. Core install: upstream Flux manifests (creates namespace, CRDs, controllers)
//  2. Sync setup: registers the GitRepository source and root Kustomization
func (f *FluxInstaller) Install(progress ProgressFunc) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	manifestURL := fmt.Sprintf(fluxManifestURL, f.version)
	if _, err := kubectl(f.clusterContext, "apply", "-f", manifestURL); err != nil {
		return fmt.Errorf("applying Flux install manifests: %w", err)
	}
	progress("Flux v" + f.version + " controllers installed")

	if err := f.waitForCRDs(progress); err != nil {
		return err
	}

	controllers := []string{
		"deployment/source-controller",
		"deployment/kustomize-controller",
		"deployment/helm-controller",
		"deployment/notification-controller",
	}
	for _, ctrl := range controllers {
		if _, err := kubectl(f.clusterContext,
			"-n", "flux-system",
			"rollout", "status",
			ctrl,
			"--timeout=300s",
		); err != nil {
			return fmt.Errorf("waiting for %s: %w", ctrl, err)
		}
		progress(ctrl + " ready")
	}

	rootPath := filepath.Join(f.repoPath, "apps", "_root.yaml")
	content, err := os.ReadFile(rootPath)
	if err != nil {
		return fmt.Errorf("reading root sync config: %w", err)
	}

	if strings.Contains(string(content), "YOUR_GIT_REPO_URL") {
		progress("Root sync skipped (update apps/_root.yaml with your Git repo URL first)")
	} else {
		if _, err := kubectl(f.clusterContext, "apply", "-f", rootPath); err != nil {
			return fmt.Errorf("applying root sync configuration: %w", err)
		}
		progress("GitRepository and root Kustomization registered")
	}

	return nil
}

func (f *FluxInstaller) waitForCRDs(progress ProgressFunc) error {
	crds := []string{
		"gitrepositories.source.toolkit.fluxcd.io",
		"kustomizations.kustomize.toolkit.fluxcd.io",
		"helmrepositories.source.toolkit.fluxcd.io",
		"helmreleases.helm.toolkit.fluxcd.io",
	}

	for _, crd := range crds {
		for attempt := 0; attempt < 30; attempt++ {
			if _, err := kubectl(f.clusterContext, "get", "crd", crd); err == nil {
				break
			}
			if attempt == 29 {
				return fmt.Errorf("timeout waiting for CRD %s", crd)
			}
			time.Sleep(2 * time.Second)
		}
	}
	progress("Flux CRDs available")
	return nil
}
