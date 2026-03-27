package devr

import (
	"os"
	"path/filepath"
	"strings"
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
	Race       bool     `yaml:"race"`
}

type ConfigRun struct {
	EnvFile string `yaml:"env_file"`
	NoEnv   bool   `yaml:"no_env"`
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
	Format          string          `yaml:"format"`
	LevelField      string          `yaml:"level_field"`
	LevelValues     ConfigLogLevels `yaml:"level_values"`
	HighlightFields []string        `yaml:"highlight_fields"`
}

type ConfigLogLevels struct {
	Error []string `yaml:"error"`
	Warn  []string `yaml:"warn"`
	Info  []string `yaml:"info"`
	Debug []string `yaml:"debug"`
}

func DefaultConfig() Config {
	return Config{
		Build: ConfigBuild{
			CmdPattern: "cmd/*/main.go",
			Race:       true,
		},
		Run: ConfigRun{
			EnvFile: ".env",
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
			Format:     "auto",
			LevelField: "level",
			LevelValues: ConfigLogLevels{
				Error: []string{"error", "err", "fatal"},
				Warn:  []string{"warn", "warning"},
				Info:  []string{"info"},
				Debug: []string{"debug", "trace"},
			},
			HighlightFields: []string{"msg"},
		},
	}
}

func (b ConfigBuild) GoFlags() []string {
	var flags []string

	if b.Race {
		flags = append(flags, "-race")
	}

	flags = append(flags, b.Flags...)

	return flags
}

func (b ConfigBuild) Label() string {
	flags := b.GoFlags()
	if len(flags) == 0 {
		return ""
	}

	return strings.Join(flags, " ")
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
