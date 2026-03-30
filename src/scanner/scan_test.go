package scanner

import (
	"os"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	defaultConfig := `
scandirs:
  include:
    - /default/path
`

	t.Run("valid config file", func(t *testing.T) {
		f, err := os.CreateTemp("", "drs-test-*.yml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		content := `
scandirs:
  include:
    - /home/user/projects
    - /home/user/work
  exclude:
    - /home/user/projects/vendor
`
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
		f.Close()

		config, err := ParseConfigFile(f.Name(), defaultConfig)
		if err != nil {
			t.Fatalf("ParseConfigFile: %v", err)
		}
		if len(config.ScanDirs.Include) != 2 {
			t.Errorf("got %d include dirs, want 2", len(config.ScanDirs.Include))
		}
		if config.ScanDirs.Include[0] != "/home/user/projects" {
			t.Errorf("include[0] = %q, want /home/user/projects", config.ScanDirs.Include[0])
		}
		if len(config.ScanDirs.Exclude) != 1 {
			t.Errorf("got %d exclude dirs, want 1", len(config.ScanDirs.Exclude))
		}
	})

	t.Run("file does not exist falls back to default", func(t *testing.T) {
		config, err := ParseConfigFile("/nonexistent/path/config.yml", defaultConfig)
		if err != nil {
			t.Fatalf("ParseConfigFile: %v", err)
		}
		if len(config.ScanDirs.Include) != 1 {
			t.Errorf("got %d include dirs, want 1", len(config.ScanDirs.Include))
		}
		if config.ScanDirs.Include[0] != "/default/path" {
			t.Errorf("include[0] = %q, want /default/path", config.ScanDirs.Include[0])
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		f, err := os.CreateTemp("", "drs-test-*.yml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		if _, err := f.WriteString("{{invalid yaml"); err != nil {
			t.Fatal(err)
		}
		f.Close()

		_, err = ParseConfigFile(f.Name(), defaultConfig)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}
