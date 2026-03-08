package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	"github.com/y0s3ph/gostrap/internal/models"
)

type SOPSInstaller struct {
	repoPath       string
	clusterContext string
	controllerType models.ControllerType
}

func NewSOPS(cfg *models.BootstrapConfig) *SOPSInstaller {
	return &SOPSInstaller{
		repoPath:       cfg.RepoPath,
		clusterContext: cfg.ClusterContext,
		controllerType: cfg.Controller.Type,
	}
}

// Install generates an age key pair, stores the private key as a
// Kubernetes Secret (for Flux decryption), and updates .sops.yaml
// with the generated public key.
func (s *SOPSInstaller) Install(progress ProgressFunc) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("generating age key pair: %w", err)
	}

	publicKey := identity.Recipient().String()
	privateKey := identity.String()
	progress("Age key pair generated")

	keyPath := filepath.Join(s.repoPath, "bootstrap", "sops", "age.agekey")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0750); err != nil {
		return fmt.Errorf("creating sops directory: %w", err)
	}
	keyContent := fmt.Sprintf("# created by gostrap — do NOT commit this file\n# public key: %s\n%s\n", publicKey, privateKey)
	if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
		return fmt.Errorf("writing age key: %w", err)
	}
	progress("Age private key saved to bootstrap/sops/age.agekey")

	if err := s.updateSOPSConfig(publicKey); err != nil {
		return fmt.Errorf("updating .sops.yaml: %w", err)
	}
	progress("Updated .sops.yaml with age public key")

	if err := s.createClusterSecret(privateKey, progress); err != nil {
		return err
	}

	return nil
}

// updateSOPSConfig replaces the placeholder in .sops.yaml with the real public key.
func (s *SOPSInstaller) updateSOPSConfig(publicKey string) error {
	sopsPath := filepath.Join(s.repoPath, ".sops.yaml")
	data, err := os.ReadFile(sopsPath)
	if err != nil {
		return fmt.Errorf("reading .sops.yaml: %w", err)
	}

	updated := strings.ReplaceAll(string(data), "AGE-PUBLIC-KEY-PLACEHOLDER", publicKey)
	return os.WriteFile(sopsPath, []byte(updated), 0600)
}

// createClusterSecret stores the age private key as a Kubernetes Secret.
// For Flux, this goes in flux-system so kustomize-controller can decrypt.
// For ArgoCD, this goes in argocd so KSOPS can access it.
func (s *SOPSInstaller) createClusterSecret(privateKey string, progress ProgressFunc) error {
	ns := "argocd"
	if s.controllerType == models.ControllerFlux {
		ns = "flux-system"
	}

	_, err := kubectl(s.clusterContext,
		"create", "secret", "generic", "sops-age",
		"--namespace", ns,
		"--from-literal=age.agekey="+privateKey,
	)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			progress("Secret sops-age already exists in " + ns)
			return nil
		}
		return fmt.Errorf("creating sops-age secret in %s: %w", ns, err)
	}

	progress(fmt.Sprintf("Secret sops-age created in %s namespace", ns))
	return nil
}

// GenerateAgeKeyForTest exposes key generation for testing without cluster access.
func GenerateAgeKeyForTest() (publicKey, privateKey string, err error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	return identity.Recipient().String(), identity.String(), nil
}
