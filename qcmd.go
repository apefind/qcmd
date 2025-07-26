package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 25

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string
type itemDelegate struct{}

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

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	str := fmt.Sprintf("%d. %s", index+1, i)
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}
	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
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
				m.choice = string(i)
			}
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
		if strings.HasPrefix(ln, "#") {
			continue
		}
		s := strings.Split(ln, ":")
		if len(s) == 1 {
			cmd = strings.TrimSpace(s[0])
			label = ""
		} else if len(s) == 2 {
			cmd = strings.TrimSpace(s[0])
			label = strings.TrimSpace(s[1])
		} else {
			continue
		}
		if strings.HasPrefix(cmd, "#") {
			continue
		}
		if cmd == "" && label == "" {
			continue
		}
		if label != "" {
			items = append(items, item(fmt.Sprintf("%s: %s", label, cmd)))
		} else {
			items = append(items, item(cmd))

		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("%v", err)
	}
	const defaultWidth = 20
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	m := model{list: l}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
