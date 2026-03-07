package devr

func Register(c *CLI, a *App) {
	c.Add(
		cmdApp(a),
		cmdTest(a),
		cmdInit(a),
		c.cmdCompletion(),
	)
}

func pkgArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return ""
}
