package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 25
const defaultWidth = 20

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item struct {
	label string
	cmd   string
}

type itemDelegate struct {
}

type model struct {
	list     list.Model
	choice   string
	command  string
	quitting bool
}

func execCommand(command string) (int, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if status.Exited() {
				return status.ExitStatus(), err
			}
			if status.Signaled() {
				return -int(status.Signal()), err
			}
		}
		return -1, err
	}
	return 0, nil
}

func (i item) FilterValue() string {
	return ""
}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, k int, l list.Item) {
	item, ok := l.(item)
	if !ok {
		return
	}
	label := fmt.Sprintf("%d. %s", k+1, item.label)
	fn := itemStyle.Render
	if k == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}
	fmt.Fprint(w, fn(label))
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.label
				m.command = i.cmd
			}
			// fmt.Println("CHOICE:", m.choice)
			// fmt.Println("CMD:", m.command)
			// fmt.Print("PRESS ENTER")
			// var name string
			// fmt.Scanln(&name)
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(m.choice)
	}
	if m.quitting {
		return quitTextStyle.Render("")
	}
	return "\n" + m.list.View()
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -f [.qcmd]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	var qfile string
	flag.StringVar(&qfile, "f", ".qcmd", ".qcmd filepath")
	flag.Parse()

	file, err := os.Open(qfile)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	var items []list.Item
	for scanner.Scan() {
		var cmd, label string
		ln := scanner.Text()
		i := strings.Index(ln, "#")
		if i >= 0 {
			ln = ln[:i]
		}
		s := strings.Split(ln, ":")
		if len(s) == 1 {
			label = strings.TrimSpace(s[0])
			cmd = strings.TrimSpace(s[0])
		} else {
			label = strings.TrimSpace(s[0])
			cmd = strings.TrimSpace(s[1])
		}
		if cmd == "" {
			continue
		}
		items = append(items, item{label: label, cmd: cmd})
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("%v", err)
	}
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select Command"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	prog := tea.NewProgram(model{list: l})
	var m tea.Model
	if m, err = prog.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
	execCommand(m.(model).command)
}
