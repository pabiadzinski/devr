package devr

import (
	"context"
	"fmt"
	"strings"
)

func (a *CLI) cmdCompletion() Command {
	return Command{
		Name: "completion", Usage: "Generate shell completions", Args: "<fish|bash|zsh>",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: %s completion <fish|bash|zsh>", a.Name)
			}

			switch args[0] {
			case "fish":
				fmt.Print(a.completionFish())
			case "bash":
				fmt.Print(a.completionBash())
			case "zsh":
				fmt.Print(a.completionZsh())
			default:
				return fmt.Errorf("unsupported shell: %s (fish, bash, zsh)", args[0])
			}

			return nil
		},
	}
}

func (a *CLI) completionFish() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s fish completions\n", a.Name)
	fmt.Fprintf(&b, "# eval (%s completion fish)\n\n", a.Name)

	// Disable file completions by default
	fmt.Fprintf(&b, "complete -c %s -f\n", a.Name)

	for _, cmd := range a.commands {
		fmt.Fprintf(&b, "complete -c %s -n '__fish_use_subcommand' -a %s -d %q\n",
			a.Name, cmd.Name, cmd.Usage)
	}

	fmt.Fprintf(&b, "complete -c %s -n '__fish_use_subcommand' -a help -d 'Show help'\n", a.Name)
	fmt.Fprintf(&b, "complete -c %s -n '__fish_use_subcommand' -a version -d 'Show version'\n", a.Name)

	// Subcommands
	for _, cmd := range a.commands {
		if len(cmd.Sub) == 0 {
			a.writeFishFlags(&b, cmd.Name, "", cmd.Flags)

			continue
		}

		cond := fmt.Sprintf("__fish_seen_subcommand_from %s; and not __fish_seen_subcommand_from", cmd.Name)
		for _, sc := range cmd.Sub {
			cond += " " + sc.Name
		}

		for _, sc := range cmd.Sub {
			fmt.Fprintf(&b, "complete -c %s -n '%s' -a %s -d %q\n",
				a.Name, cond, sc.Name, sc.Usage)
		}

		for _, sc := range cmd.Sub {
			a.writeFishFlags(&b, cmd.Name, sc.Name, sc.Flags)
		}
	}

	// Completion subcommand completions
	fmt.Fprintf(&b, "complete -c %s -n '__fish_seen_subcommand_from completion' -a 'fish bash zsh'\n", a.Name)

	return b.String()
}

func (a *CLI) writeFishFlags(b *strings.Builder, parent, sub string, flags []Flag) {
	if len(flags) == 0 {
		return
	}

	cond := fmt.Sprintf("__fish_seen_subcommand_from %s", parent)
	if sub != "" {
		cond = fmt.Sprintf("__fish_seen_subcommand_from %s; and __fish_seen_subcommand_from %s", parent, sub)
	}

	for _, f := range flags {
		short := ""
		if f.Short != "" {
			short = fmt.Sprintf(" -s %s", f.Short)
		}

		if f.Bool != nil {
			fmt.Fprintf(b, "complete -c %s -n '%s' -l %s%s -d %q\n",
				a.Name, cond, f.Name, short, f.Usage)
		} else {
			fmt.Fprintf(b, "complete -c %s -n '%s' -l %s%s -r -d %q\n",
				a.Name, cond, f.Name, short, f.Usage)
		}
	}
}

func (a *CLI) completionBash() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s bash completions\n", a.Name)
	fmt.Fprintf(&b, "# eval \"$(%s completion bash)\"\n\n", a.Name)

	fmt.Fprintf(&b, "_%s() {\n", a.Name)
	fmt.Fprintf(&b, "  local cur prev words cword\n")
	fmt.Fprintf(&b, "  _init_completion || return\n\n")

	tops := []string{"help", "version"}
	for _, cmd := range a.commands {
		tops = append(tops, cmd.Name)
	}

	fmt.Fprintf(&b, "  case \"${words[1]}\" in\n")

	for _, cmd := range a.commands {
		if len(cmd.Sub) == 0 {
			continue
		}

		var subs []string
		for _, sc := range cmd.Sub {
			subs = append(subs, sc.Name)
		}

		fmt.Fprintf(&b, "    %s)\n", cmd.Name)
		fmt.Fprintf(&b, "      COMPREPLY=($(compgen -W %q -- \"$cur\"))\n", strings.Join(subs, " "))
		fmt.Fprintf(&b, "      return ;;\n")
	}

	fmt.Fprintf(&b, "    completion)\n")
	fmt.Fprintf(&b, "      COMPREPLY=($(compgen -W 'fish bash zsh' -- \"$cur\"))\n")
	fmt.Fprintf(&b, "      return ;;\n")

	fmt.Fprintf(&b, "  esac\n\n")

	fmt.Fprintf(&b, "  COMPREPLY=($(compgen -W %q -- \"$cur\"))\n", strings.Join(tops, " "))
	fmt.Fprintf(&b, "}\n\n")

	fmt.Fprintf(&b, "complete -F _%s %s\n", a.Name, a.Name)

	return b.String()
}

func (a *CLI) completionZsh() string {
	var b strings.Builder

	fmt.Fprintf(&b, "#compdef %s\n", a.Name)
	fmt.Fprintf(&b, "# %s zsh completions\n", a.Name)
	fmt.Fprintf(&b, "# eval \"$(%s completion zsh)\"\n\n", a.Name)

	fmt.Fprintf(&b, "_%s() {\n", a.Name)
	fmt.Fprintf(&b, "  local -a commands\n\n")

	fmt.Fprintf(&b, "  commands=(\n")

	for _, cmd := range a.commands {
		fmt.Fprintf(&b, "    '%s:%s'\n", cmd.Name, cmd.Usage)
	}

	fmt.Fprintf(&b, "    'help:Show help'\n")
	fmt.Fprintf(&b, "    'version:Show version'\n")
	fmt.Fprintf(&b, "  )\n\n")

	fmt.Fprintf(&b, "  if (( CURRENT == 2 )); then\n")
	fmt.Fprintf(&b, "    _describe 'command' commands\n")
	fmt.Fprintf(&b, "    return\n")
	fmt.Fprintf(&b, "  fi\n\n")

	fmt.Fprintf(&b, "  case \"$words[2]\" in\n")

	for _, cmd := range a.commands {
		if len(cmd.Sub) == 0 {
			continue
		}

		fmt.Fprintf(&b, "    %s)\n", cmd.Name)
		fmt.Fprintf(&b, "      local -a subcmds\n")
		fmt.Fprintf(&b, "      subcmds=(\n")

		for _, sc := range cmd.Sub {
			fmt.Fprintf(&b, "        '%s:%s'\n", sc.Name, sc.Usage)
		}

		fmt.Fprintf(&b, "      )\n")
		fmt.Fprintf(&b, "      _describe 'subcommand' subcmds\n")
		fmt.Fprintf(&b, "      ;;\n")
	}

	fmt.Fprintf(&b, "    completion)\n")
	fmt.Fprintf(&b, "      _values 'shell' fish bash zsh\n")
	fmt.Fprintf(&b, "      ;;\n")

	fmt.Fprintf(&b, "  esac\n")
	fmt.Fprintf(&b, "}\n\n")

	fmt.Fprintf(&b, "_%s\n", a.Name)

	return b.String()
}
