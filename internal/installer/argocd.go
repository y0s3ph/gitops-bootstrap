package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/y0s3ph/gostrap/internal/models"
)

// ProgressFunc is called after each installation step completes.
type ProgressFunc func(message string)

const argoCDManifestURL = "https://raw.githubusercontent.com/argoproj/argo-cd/v%s/manifests/install.yaml"

type ArgoCDInstaller struct {
	repoPath       string
	clusterContext string
	version        string
}

func NewArgoCD(cfg *models.BootstrapConfig) *ArgoCDInstaller {
	return &ArgoCDInstaller{
		repoPath:       cfg.RepoPath,
		clusterContext: cfg.ClusterContext,
		version:        cfg.Controller.Version,
	}
}

// Install bootstraps ArgoCD on the target cluster in three phases:
//  1. Core install: namespace + upstream ArgoCD manifests (creates CRDs)
//  2. Overlay: applies kustomization with config patches and AppProject
//  3. Root app: registers the App-of-Apps root Application
func (a *ArgoCDInstaller) Install(progress ProgressFunc) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	// Phase 1: create namespace and install core ArgoCD (creates CRDs)
	nsPath := filepath.Join(a.repoPath, "bootstrap", "argocd", "namespace.yaml")
	if _, err := kubectl(a.clusterContext, "apply", "-f", nsPath); err != nil {
		return fmt.Errorf("creating argocd namespace: %w", err)
	}

	manifestURL := fmt.Sprintf(argoCDManifestURL, a.version)
	if _, err := kubectl(a.clusterContext, "apply", "-n", "argocd", "-f", manifestURL); err != nil {
		return fmt.Errorf("applying ArgoCD upstream manifests: %w", err)
	}
	progress("ArgoCD v" + a.version + " core installed")

	// Wait for CRDs to be established before applying CRD-based resources
	if err := a.waitForCRDs(progress); err != nil {
		return err
	}

	// Phase 2: apply full kustomization (patches + AppProject)
	bootstrapPath := filepath.Join(a.repoPath, "bootstrap", "argocd")
	if _, err := kubectl(a.clusterContext, "apply", "-k", bootstrapPath); err != nil {
		return fmt.Errorf("applying ArgoCD kustomization: %w", err)
	}
	progress("RBAC and AppProject configured")

	// Wait for all workloads (deployments + statefulset)
	rollouts := []string{
		"deployment/argocd-server",
		"deployment/argocd-repo-server",
		"statefulset/argocd-application-controller",
	}
	for _, workload := range rollouts {
		if _, err := kubectl(a.clusterContext,
			"-n", "argocd",
			"rollout", "status",
			workload,
			"--timeout=300s",
		); err != nil {
			return fmt.Errorf("waiting for %s: %w", workload, err)
		}
		progress(workload + " ready")
	}

	// Phase 3: register root Application (skip if repoURL placeholder is still present)
	rootAppPath := filepath.Join(a.repoPath, "apps", "_root.yaml")
	content, err := os.ReadFile(rootAppPath)
	if err != nil {
		return fmt.Errorf("reading root Application: %w", err)
	}

	if strings.Contains(string(content), "YOUR_GIT_REPO_URL") {
		progress("Root Application skipped (update apps/_root.yaml with your Git repo URL first)")
	} else {
		if _, err := kubectl(a.clusterContext, "apply", "-f", rootAppPath); err != nil {
			return fmt.Errorf("applying root Application: %w", err)
		}
		progress("Root Application registered")
	}

	return nil
}

func (a *ArgoCDInstaller) waitForCRDs(progress ProgressFunc) error {
	crds := []string{
		"applications.argoproj.io",
		"appprojects.argoproj.io",
	}

	for _, crd := range crds {
		for attempt := 0; attempt < 30; attempt++ {
			if _, err := kubectl(a.clusterContext,
				"get", "crd", crd,
			); err == nil {
				break
			}
			if attempt == 29 {
				return fmt.Errorf("timeout waiting for CRD %s", crd)
			}
			time.Sleep(2 * time.Second)
		}
	}
	progress("ArgoCD CRDs available")
	return nil
}
