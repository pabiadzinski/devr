package devr

func (cfg *Config) BuildCLIFlags() []Flag {
	return []Flag{
		{Name: "race", Usage: "Enable race detector", Bool: &cfg.Build.Race},
	}
}

func (cfg *Config) RunCLIFlags() []Flag {
	return []Flag{
		{Name: "no-env", Usage: "Skip loading .env file", Bool: &cfg.Run.NoEnv},
		{Name: "env-file", Usage: "Env file path", Value: &cfg.Run.EnvFile},
	}
}

func joinFlags(groups ...[]Flag) []Flag {
	var out []Flag

	for _, g := range groups {
		out = append(out, g...)
	}

	return out
}
