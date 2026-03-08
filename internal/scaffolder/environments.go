package scaffolder

import (
	"path/filepath"
)

type baseAppData struct {
	Name string
	Port int
}

type overlayData struct {
	AppName  string
	EnvName  string
	Replicas int
}

var defaultReplicas = map[string]int{
	"dev":        1,
	"staging":    2,
	"production": 3,
}

func (s *Scaffolder) scaffoldAppEnvironments(appName string, port int) error {
	baseData := baseAppData{Name: appName, Port: port}
	basePath := filepath.Join("environments", "base", appName)

	baseTemplates := []struct{ tmpl, out string }{
		{"environments/base/kustomization.yaml.tmpl", filepath.Join(basePath, "kustomization.yaml")},
		{"environments/base/deployment.yaml.tmpl", filepath.Join(basePath, "deployment.yaml")},
		{"environments/base/service.yaml.tmpl", filepath.Join(basePath, "service.yaml")},
	}

	for _, bt := range baseTemplates {
		if err := s.renderTemplateWithData(bt.tmpl, bt.out, baseData); err != nil {
			return err
		}
	}

	for _, env := range s.config.Environments {
		replicas := defaultReplicas[env.Name]
		if replicas == 0 {
			replicas = 1
		}

		data := overlayData{
			AppName:  appName,
			EnvName:  env.Name,
			Replicas: replicas,
		}

		outPath := filepath.Join("environments", env.Name, appName, "kustomization.yaml")
		if err := s.renderTemplateWithData("environments/overlay/kustomization.yaml.tmpl", outPath, data); err != nil {
			return err
		}
	}

	return nil
}
