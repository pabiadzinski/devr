package devr

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type LogViewOptions struct {
	LogPath         string
	PID             int
	OnStop          func()
	OnExit          func()
	Title           string
	ExitCh          <-chan error
	HighlightFields []string
}

func RunLogView(opts LogViewOptions) error {
	f, err := os.Open(opts.LogPath)
	if err != nil {
		return fmt.Errorf("no log file at %s", opts.LogPath)
	}

	m := newModel()
	m.title = opts.Title
	m.highlightFields = opts.HighlightFields

	m.lines, m.filtered = loadLines(f)
	if len(m.filtered) > 0 {
		m.cursor = len(m.filtered) - 1
	}

	live := opts.PID > 0
	m.follow = live

	p := tea.NewProgram(m, tea.WithAltScreen())

	if live {
		go tailFile(f, p, m.done)
	} else {
		_ = f.Close()
	}

	if opts.ExitCh != nil {
		go func() {
			select {
			case <-opts.ExitCh:
				if opts.OnExit != nil {
					opts.OnExit()
				}

				p.Send(processExitMsg{})
			case <-m.done:
			}
		}()
	}

	finalModel, err := p.Run()

	close(m.done)

	if live {
		_ = f.Close()
	}

	if fm, ok := finalModel.(logViewModel); ok && fm.exit == exitStop && opts.OnStop != nil {
		opts.OnStop()
	}

	return err
}

func loadLines(f *os.File) ([]logEntry, []int) {
	var (
		lines    []logEntry
		filtered []int
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		entry := parseLine(scanner.Text())
		lines = append(lines, entry)
		filtered = append(filtered, len(lines)-1)
	}

	return lines, filtered
}

func tailFile(f *os.File, p *tea.Program, done chan struct{}) {
	reader := bufio.NewReader(f)

	for {
		select {
		case <-done:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		line = strings.TrimRight(line, "\n")
		if line != "" {
			p.Send(lineMsg(line))
		}
	}
}
