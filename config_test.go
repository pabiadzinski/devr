package devr

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "cmd/*/main.go", cfg.Build.CmdPattern)
	assert.Equal(t, []string{"-race"}, cfg.Build.Flags)
	assert.Equal(t, "", cfg.Run.EnvFile)
	assert.Equal(t, []string{"."}, cfg.Watch.Dirs)
	assert.Equal(t, []string{".go"}, cfg.Watch.Extensions)
	assert.Equal(t, []string{"vendor", "node_modules"}, cfg.Watch.Exclude)
	assert.Equal(t, 500*time.Millisecond, cfg.Watch.Debounce)
	assert.Equal(t, "coverage.out", cfg.Test.CoverProfile)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		assert func(t *testing.T, cfg Config)
	}{
		{
			name: "partial override build flags",
			yaml: "build:\n  flags: [\"-trimpath\"]\n",
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, []string{"-trimpath"}, cfg.Build.Flags)
				assert.Equal(t, "cmd/*/main.go", cfg.Build.CmdPattern)
			},
		},
		{
			name: "override watch debounce",
			yaml: "watch:\n  debounce: 1s\n",
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, time.Second, cfg.Watch.Debounce)
				assert.Equal(t, []string{".go"}, cfg.Watch.Extensions)
			},
		},
		{
			name: "override env_file and test",
			yaml: "run:\n  env_file: .env.local\ntest:\n  cover_profile: cover.out\n",
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, ".env.local", cfg.Run.EnvFile)
				assert.Equal(t, "cover.out", cfg.Test.CoverProfile)
			},
		},
		{
			name: "override watch extensions and exclude",
			yaml: "watch:\n  extensions: [\".go\", \".templ\"]\n  exclude: [\"vendor\", \"tmp\"]\n",
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, []string{".go", ".templ"}, cfg.Watch.Extensions)
				assert.Equal(t, []string{"vendor", "tmp"}, cfg.Watch.Exclude)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, configFile), []byte(tt.yaml), 0644))

			cfg := LoadConfig(dir)
			tt.assert(t, cfg)
		})
	}
}

func TestLoadConfigMissing(t *testing.T) {
	cfg := LoadConfig(t.TempDir())
	assert.Equal(t, DefaultConfig(), cfg)
}
