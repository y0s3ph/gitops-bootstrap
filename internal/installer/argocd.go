package installer

import (
	"fmt"
	"path/filepath"

	"github.com/y0s3ph/gitops-bootstrap/internal/models"
)

// ProgressFunc is called after each installation step completes.
type ProgressFunc func(message string)

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

// Install applies the generated ArgoCD bootstrap manifests to the cluster
// and waits for all core components to become ready.
func (a *ArgoCDInstaller) Install(progress ProgressFunc) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	bootstrapPath := filepath.Join(a.repoPath, "bootstrap", "argocd")
	if _, err := kubectl(a.clusterContext, "apply", "-k", bootstrapPath); err != nil {
		return fmt.Errorf("applying ArgoCD manifests: %w", err)
	}
	progress("Bootstrap manifests applied")

	deployments := []string{
		"argocd-server",
		"argocd-repo-server",
		"argocd-application-controller",
	}

	for _, deploy := range deployments {
		if _, err := kubectl(a.clusterContext,
			"-n", "argocd",
			"rollout", "status",
			"deployment/"+deploy,
			"--timeout=300s",
		); err != nil {
			return fmt.Errorf("waiting for %s: %w", deploy, err)
		}
		progress(deploy + " ready")
	}

	rootAppPath := filepath.Join(a.repoPath, "apps", "_root.yaml")
	if _, err := kubectl(a.clusterContext, "apply", "-f", rootAppPath); err != nil {
		return fmt.Errorf("applying root Application: %w", err)
	}
	progress("Root Application registered")

	return nil
}
