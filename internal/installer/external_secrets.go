package installer

import (
	"fmt"
	"time"

	"github.com/y0s3ph/gostrap/internal/models"
)

const esoManifestURL = "https://github.com/external-secrets/external-secrets/releases/download/v%s/external-secrets.yaml"

type ESOInstaller struct {
	clusterContext string
	version        string
}

func NewESO(cfg *models.BootstrapConfig) *ESOInstaller {
	return &ESOInstaller{
		clusterContext: cfg.ClusterContext,
		version:        cfg.Secrets.Version,
	}
}

// Install deploys the External Secrets Operator onto the cluster,
// waits for CRDs to become available, and verifies all controller
// deployments reach a ready state.
//
// The upstream static manifest ships with namespace: default. We apply
// with --server-side to handle the large manifest and check rollout
// in the default namespace.
func (e *ESOInstaller) Install(progress ProgressFunc) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	manifestURL := fmt.Sprintf(esoManifestURL, e.version)
	if _, err := kubectl(e.clusterContext, "apply", "--server-side", "-f", manifestURL); err != nil {
		return fmt.Errorf("applying ESO manifests: %w", err)
	}
	progress("External Secrets Operator v" + e.version + " installed")

	if err := e.waitForCRDs(progress); err != nil {
		return err
	}

	controllers := []string{
		"deployment/external-secrets",
		"deployment/external-secrets-webhook",
		"deployment/external-secrets-cert-controller",
	}
	for _, ctrl := range controllers {
		if _, err := kubectl(e.clusterContext,
			"rollout", "status",
			ctrl,
			"--timeout=300s",
		); err != nil {
			return fmt.Errorf("waiting for %s: %w", ctrl, err)
		}
		progress(ctrl + " ready")
	}

	return nil
}

func (e *ESOInstaller) waitForCRDs(progress ProgressFunc) error {
	crds := []string{
		"externalsecrets.external-secrets.io",
		"secretstores.external-secrets.io",
		"clustersecretstores.external-secrets.io",
	}

	for _, crd := range crds {
		for attempt := 0; attempt < 30; attempt++ {
			if _, err := kubectl(e.clusterContext, "get", "crd", crd); err == nil {
				break
			}
			if attempt == 29 {
				return fmt.Errorf("timeout waiting for CRD %s", crd)
			}
			time.Sleep(2 * time.Second)
		}
	}
	progress("ESO CRDs available")
	return nil
}
