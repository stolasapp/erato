package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "valid config",
			yaml:    `root_uri: "https://example.com"`,
			wantErr: "",
		},
		{
			name:    "missing root_uri fails validation",
			yaml:    `log_level: INFO`,
			wantErr: "config validation failed",
		},
		{
			name:    "empty root_uri fails validation",
			yaml:    `root_uri: ""`,
			wantErr: "config validation failed",
		},
		{
			name:    "invalid yaml syntax",
			yaml:    `invalid: [yaml: content`,
			wantErr: "failed to unmarshal config file",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := writeTestConfig(t, test.yaml)
			cfg, err := Load(path)

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				assert.Nil(t, cfg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	cfg, err := Load("/nonexistent/path/config.yaml")
	require.ErrorContains(t, err, "failed to read config file")
	assert.Nil(t, cfg)
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)
	return path
}
