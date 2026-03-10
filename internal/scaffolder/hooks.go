package scaffolder

func (s *Scaffolder) renderPreCommitConfig() error {
	return s.renderTemplate(
		"hooks/pre-commit-config.yaml.tmpl",
		".pre-commit-config.yaml",
	)
}
