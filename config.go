package devr

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const configFile = ".devr.yaml"

type Config struct {
	Build  ConfigBuild `yaml:"build"`
	Run    ConfigRun   `yaml:"run"`
	Watch  ConfigWatch `yaml:"watch"`
	Test   ConfigTest  `yaml:"test"`
	Logs   ConfigLogs  `yaml:"logs"`
	Notify bool        `yaml:"notify"`
}

type ConfigBuild struct {
	CmdPattern string   `yaml:"cmd_pattern"`
	Flags      []string `yaml:"flags"`
}

type ConfigRun struct {
	EnvFile string `yaml:"env_file"`
}

type ConfigWatch struct {
	Dirs       []string      `yaml:"dirs"`
	Extensions []string      `yaml:"extensions"`
	Exclude    []string      `yaml:"exclude"`
	Debounce   time.Duration `yaml:"debounce"`
}

type ConfigTest struct {
	CoverProfile string `yaml:"cover_profile"`
}

type ConfigLogs struct {
	HighlightFields []string `yaml:"highlight_fields"`
}

func DefaultConfig() Config {
	return Config{
		Build: ConfigBuild{
			CmdPattern: "cmd/*/main.go",
			Flags:      []string{"-race"},
		},
		Watch: ConfigWatch{
			Dirs:       []string{"."},
			Extensions: []string{".go"},
			Exclude:    []string{"vendor", "node_modules"},
			Debounce:   500 * time.Millisecond,
		},
		Test: ConfigTest{
			CoverProfile: "coverage.out",
		},
		Logs: ConfigLogs{
			HighlightFields: []string{"msg"},
		},
	}
}

func LoadConfig(dir string) Config {
	cfg := DefaultConfig()

	path := filepath.Join(dir, configFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		Warn("invalid %s: %v", configFile, err)

		return DefaultConfig()
	}

	return cfg
}
