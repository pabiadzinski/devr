package devr

import (
	"os"
	"strings"
)

func loadEnvFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var env []string

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		v = strings.Trim(v, `"'`)
		env = append(env, k+"="+v)
	}

	return env, nil
}
