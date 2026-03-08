package installer

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/y0s3ph/gostrap/internal/models"
)

const sealedSecretsManifestURL = "https://github.com/bitnami-labs/sealed-secrets/releases/download/v%s/controller.yaml"

type SealedSecretsInstaller struct {
	repoPath       string
	clusterContext string
	version        string
}

func NewSealedSecrets(cfg *models.BootstrapConfig) *SealedSecretsInstaller {
	return &SealedSecretsInstaller{
		repoPath:       cfg.RepoPath,
		clusterContext: cfg.ClusterContext,
		version:        cfg.Secrets.Version,
	}
}

// Install deploys the Sealed Secrets controller onto the cluster,
// waits for it to become ready, and exports its public certificate
// so developers can seal secrets without direct cluster access.
func (ss *SealedSecretsInstaller) Install(progress ProgressFunc) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	manifestURL := fmt.Sprintf(sealedSecretsManifestURL, ss.version)
	if _, err := kubectl(ss.clusterContext, "apply", "-f", manifestURL); err != nil {
		return fmt.Errorf("applying Sealed Secrets controller: %w", err)
	}
	progress("Sealed Secrets v" + ss.version + " installed")

	if _, err := kubectl(ss.clusterContext,
		"-n", "kube-system",
		"rollout", "status",
		"deployment/sealed-secrets-controller",
		"--timeout=300s",
	); err != nil {
		return fmt.Errorf("waiting for sealed-secrets-controller: %w", err)
	}
	progress("sealed-secrets-controller ready")

	if err := ss.exportPublicCert(progress); err != nil {
		progress("Public cert export skipped: " + err.Error())
	}

	return nil
}

// exportPublicCert fetches the controller's public sealing key and writes
// it to bootstrap/sealed-secrets/pub-cert.pem in the generated repo.
// This cert is safe to commit — it allows offline sealing.
func (ss *SealedSecretsInstaller) exportPublicCert(progress ProgressFunc) error {
	var certB64 string
	var err error

	for attempt := 0; attempt < 15; attempt++ {
		certB64, err = kubectl(ss.clusterContext,
			"-n", "kube-system",
			"get", "secret",
			"-l", "sealedsecrets.bitnami.com/sealed-secrets-key=active",
			"-o", "jsonpath={.items[0].data.tls\\.crt}",
		)
		if err == nil && strings.TrimSpace(certB64) != "" {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("fetching public cert: %w", err)
	}

	certB64 = strings.TrimSpace(certB64)
	if certB64 == "" {
		return fmt.Errorf("sealed secrets key not found")
	}

	certPEM, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		return fmt.Errorf("decoding certificate: %w", err)
	}

	certPath := filepath.Join(ss.repoPath, "bootstrap", "sealed-secrets", "pub-cert.pem")
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return fmt.Errorf("creating cert directory: %w", err)
	}
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("writing pub-cert.pem: %w", err)
	}

	progress("Public cert exported to bootstrap/sealed-secrets/pub-cert.pem")
	return nil
}
