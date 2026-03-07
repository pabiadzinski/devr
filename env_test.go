package devr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "simple key=value",
			content: "FOO=bar\nBAZ=qux\n",
			want:    []string{"FOO=bar", "BAZ=qux"},
		},
		{
			name:    "double quotes stripped",
			content: `DB_URL="postgres://localhost/db"`,
			want:    []string{"DB_URL=postgres://localhost/db"},
		},
		{
			name:    "single quotes stripped",
			content: `SECRET='s3cret'`,
			want:    []string{"SECRET=s3cret"},
		},
		{
			name:    "export prefix stripped",
			content: "export FOO=bar\nexport BAZ=\"qux\"\n",
			want:    []string{"FOO=bar", "BAZ=qux"},
		},
		{
			name:    "comments and empty lines skipped",
			content: "# comment\n\nFOO=bar\n  # indented comment\n",
			want:    []string{"FOO=bar"},
		},
		{
			name:    "lines without = skipped",
			content: "FOO=bar\nINVALID\nBAZ=qux\n",
			want:    []string{"FOO=bar", "BAZ=qux"},
		},
		{
			name:    "empty value",
			content: "FOO=\n",
			want:    []string{"FOO="},
		},
		{
			name:    "value with equals sign",
			content: "FOO=bar=baz\n",
			want:    []string{"FOO=bar=baz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".env")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0644))

			got, err := loadEnvFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadEnvFileMissing(t *testing.T) {
	_, err := loadEnvFile("/nonexistent/.env")
	assert.Error(t, err)
}
