package devr

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type CLI struct {
	Name     string
	Version  string
	Flags    []Flag
	Setup    func() error
	commands []Command
}

type Command struct {
	Name  string
	Usage string
	Args  string
	Run   func(ctx context.Context, args []string) error
	Flags []Flag
	Sub   []Command
}

type Flag struct {
	Name    string
	Short   string
	Usage   string
	Default string
	Value   *string
	Bool    *bool
}

func (cli *CLI) Add(cmds ...Command) {
	cli.commands = append(cli.commands, cmds...)
}

func (cli *CLI) Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	args = parseFlags(cli.Flags, args)

	if len(args) == 0 || isRootHelpToken(args[0]) {
		cli.printHelp(os.Stdout)
		return nil
	}

	if isVersionToken(args[0]) {
		fmt.Println(cli.Name, cli.Version)
		return nil
	}

	cmd, rest, ok := cli.lookupCommand(args)
	if !ok {
		cli.printHelp(os.Stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}

	rest = parseFlags(cmd.Flags, rest)

	if err := cli.runSetup(); err != nil {
		return err
	}

	return cli.runCommand(ctx, cmd, rest)
}

func parseFlags(flags []Flag, args []string) []string {
	fs := newFlagSet(flags)
	_ = fs.Parse(args)

	return fs.Args()
}

func (cli *CLI) runCommand(ctx context.Context, cmd *Command, args []string) error {
	if len(cmd.Sub) == 0 {
		return cmd.Run(ctx, args)
	}

	if len(args) == 0 || isCommandHelpToken(args[0]) {
		printCommandHelp(os.Stdout, cli.Name, cmd)
		return nil
	}

	sub, rest, ok := lookupSubcommand(cmd, args)
	if !ok {
		printCommandHelp(os.Stderr, cli.Name, cmd)
		return fmt.Errorf("unknown command: %s %s", cmd.Name, args[0])
	}

	if hasHelpFlag(rest) {
		printSubcommandHelp(os.Stdout, cli.Name, cmd.Name, sub)
		return nil
	}

	rest = parseFlags(sub.Flags, rest)

	return sub.Run(ctx, rest)
}

func printCommandHelp(w io.Writer, appName string, cmd *Command) {
	var b strings.Builder

	fmt.Fprintf(&b, "Usage: %s %s <command> [args]\n", appName, cmd.Name)
	fmt.Fprintf(&b, "\n%s\n", cmd.Usage)
	fmt.Fprintf(&b, "\nCommands:\n")
	writeCommandList(&b, cmd.Sub, false)

	_, _ = io.WriteString(w, b.String())
}

func printSubcommandHelp(w io.Writer, appName, parentName string, cmd *Command) {
	var b strings.Builder

	fmt.Fprintf(&b, "Usage: %s %s %s", appName, parentName, cmd.Name)

	if cmd.Args != "" {
		fmt.Fprintf(&b, " %s", cmd.Args)
	}

	fmt.Fprintf(&b, " [options]\n")
	fmt.Fprintf(&b, "\n%s\n", cmd.Usage)

	if len(cmd.Flags) > 0 {
		fmt.Fprintf(&b, "\nOptions:\n")

		writeFlagList(&b, cmd.Flags, true)
	}

	_, _ = io.WriteString(w, b.String())
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			return true
		}
	}

	return false
}

func (cli *CLI) printHelp(w io.Writer) {
	var b strings.Builder

	fmt.Fprintf(&b, "Usage: %s", cli.Name)

	if len(cli.Flags) > 0 {
		fmt.Fprintf(&b, " [options]")
	}

	fmt.Fprintf(&b, " <command> [args]\n")

	if len(cli.Flags) > 0 {
		fmt.Fprintf(&b, "\nOptions:\n")
		writeFlagList(&b, cli.Flags, false)
	}

	fmt.Fprintf(&b, "\nCommands:\n")
	writeCommandList(&b, cli.commands, true)

	if len(cli.commands) > 0 {
		fmt.Fprintf(&b, "\nRun '%s help' for more information.\n", cli.Name)
	}

	_, _ = io.WriteString(w, b.String())
}

func (cli *CLI) lookupCommand(args []string) (*Command, []string, bool) {
	name := args[0]
	rest := args[1:]

	for i := range cli.commands {
		if cli.commands[i].Name == name {
			return &cli.commands[i], rest, true
		}
	}

	return nil, nil, false
}

func (cli *CLI) runSetup() error {
	if cli.Setup == nil {
		return nil
	}

	return cli.Setup()
}

func lookupSubcommand(cmd *Command, args []string) (*Command, []string, bool) {
	name := args[0]
	rest := args[1:]

	for i := range cmd.Sub {
		if cmd.Sub[i].Name == name {
			return &cmd.Sub[i], rest, true
		}
	}

	return nil, nil, false
}

func newFlagSet(flags []Flag) *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	for i := range flags {
		bindFlag(fs, &flags[i])
	}

	return fs
}

func bindFlag(fs *flag.FlagSet, f *Flag) {
	if f.Bool != nil {
		fs.BoolVar(f.Bool, f.Name, false, f.Usage)

		if f.Short != "" {
			fs.BoolVar(f.Bool, f.Short, false, f.Usage)
		}

		return
	}

	if f.Value == nil {
		f.Value = new(string)
	}

	*f.Value = f.Default

	fs.StringVar(f.Value, f.Name, f.Default, f.Usage)

	if f.Short != "" {
		fs.StringVar(f.Value, f.Short, f.Default, f.Usage)
	}
}

func writeCommandList(w io.Writer, cmds []Command, includeBuiltins bool) {
	maxName := commandNameWidth(cmds, includeBuiltins)
	pad := fmt.Sprintf("%%-%ds", maxName+2)

	for _, cmd := range cmds {
		_, _ = fmt.Fprintf(w, "  "+pad+" %s\n", displayCommandName(cmd, includeBuiltins), cmd.Usage)
	}

	if includeBuiltins {
		_, _ = fmt.Fprintf(w, "\n  "+pad+" %s\n", "help", "Show this help")
		_, _ = fmt.Fprintf(w, "  "+pad+" %s\n", "version", "Show version")
	}
}

func commandNameWidth(cmds []Command, includeBuiltins bool) int {
	maxName := 0

	for _, cmd := range cmds {
		if n := len(displayCommandName(cmd, includeBuiltins)); n > maxName {
			maxName = n
		}
	}

	if includeBuiltins {
		for _, builtin := range []string{"help", "version"} {
			if len(builtin) > maxName {
				maxName = len(builtin)
			}
		}
	}

	return maxName
}

func displayCommandName(cmd Command, root bool) string {
	if root && len(cmd.Sub) > 0 {
		return cmd.Name
	}

	if cmd.Args == "" {
		return cmd.Name
	}

	return cmd.Name + " " + cmd.Args
}

func writeFlagList(w io.Writer, flags []Flag, includeDefaults bool) {
	for _, f := range flags {
		usage := f.Usage
		if includeDefaults && f.Default != "" {
			usage += fmt.Sprintf(" (default: %s)", f.Default)
		}

		_, _ = fmt.Fprintf(w, "  %-16s %s\n", formatFlagNames(f), usage)
	}
}

func formatFlagNames(f Flag) string {
	names := "--" + f.Name
	if f.Short == "" {
		return names
	}

	return "-" + f.Short + ", " + names
}

func isRootHelpToken(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func isCommandHelpToken(arg string) bool {
	return arg == "help" || arg == "-h"
}

func isVersionToken(arg string) bool {
	return arg == "version" || arg == "--version"
}
