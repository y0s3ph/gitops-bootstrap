package scaffolder

func (s *Scaffolder) renderDocs() error {
	docs := []struct{ tmpl, out string }{
		{"docs/ARCHITECTURE.md.tmpl", "docs/ARCHITECTURE.md"},
		{"docs/ADDING-AN-APP.md.tmpl", "docs/ADDING-AN-APP.md"},
		{"docs/SECRETS.md.tmpl", "docs/SECRETS.md"},
		{"docs/TROUBLESHOOTING.md.tmpl", "docs/TROUBLESHOOTING.md"},
	}

	for _, d := range docs {
		if err := s.renderTemplate(d.tmpl, d.out); err != nil {
			return err
		}
	}

	return nil
}
