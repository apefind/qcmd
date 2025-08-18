package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	itemlist "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	gloss "github.com/charmbracelet/lipgloss"
)

const listHeight = 25
const defaultWidth = 20

var (
	titleStyle        = gloss.NewStyle().MarginLeft(2)
	itemStyle         = gloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = gloss.NewStyle().PaddingLeft(2).Foreground(gloss.Color("170"))
	paginationStyle   = itemlist.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = itemlist.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = gloss.NewStyle().Margin(1, 0, 2, 4)
)

type cmdItem struct {
	label   string
	command string
}

type cmdItemDelegate struct {
}

type cmdItemModel struct {
	list     itemlist.Model
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

func (i cmdItem) FilterValue() string {
	return ""
}

func (d cmdItemDelegate) Height() int {
	return 1
}

func (d cmdItemDelegate) Spacing() int {
	return 0
}

func (d cmdItemDelegate) Update(_ tea.Msg, _ *itemlist.Model) tea.Cmd {
	return nil
}

func (d cmdItemDelegate) Render(w io.Writer, m itemlist.Model, k int, l itemlist.Item) {
	item, ok := l.(cmdItem)
	if !ok {
		return
	}
	label := fmt.Sprintf("%d. %s", k+1, item.label)
	if k == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("> "+label))
	} else {
		fmt.Fprint(w, itemStyle.Render(label))
	}
}

func (m cmdItemModel) Init() tea.Cmd {
	return nil
}

func (m cmdItemModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(cmdItem)
			if ok {
				m.choice = i.label
				m.command = i.command
			}
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m cmdItemModel) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(m.choice)
	}
	if m.quitting {
		return quitTextStyle.Render("")
	}
	return "\n" + m.list.View()
}

func getNthComand(items []itemlist.Item, n int) (string, error) {
	if n > len(items) {
		return "", errors.New("sdf")
	}
	item, ok := items[n-1].(cmdItem)
	if !ok {
		return "", errors.New("sdf")
	}
	return item.command, nil
}

func selectComand(items []itemlist.Item) (string, error) {
	cmdItemList := itemlist.New(items, cmdItemDelegate{}, defaultWidth, listHeight)
	cmdItemList.Title = "Select Command"
	cmdItemList.SetShowStatusBar(false)
	cmdItemList.SetFilteringEnabled(false)
	cmdItemList.Styles.Title = titleStyle
	cmdItemList.Styles.PaginationStyle = paginationStyle
	cmdItemList.Styles.HelpStyle = helpStyle
	prog := tea.NewProgram(cmdItemModel{list: cmdItemList})
	var m tea.Model
	var err error
	if m, err = prog.Run(); err != nil {
		return "", err
	}
	return m.(cmdItemModel).command, nil
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -f [.qcmd]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	var qFile string
	var qCmd int
	flag.StringVar(&qFile, "f", ".qcmd", ".qcmd filepath")
	flag.IntVar(&qCmd, "n", 0, "Execute the n-th command")
	flag.Parse()
	file, err := os.Open(qFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	if err := os.Chdir(filepath.Dir(qFile)); err != nil {
		log.Fatal(err)
	}
	var items []itemlist.Item
	scanner := bufio.NewScanner(file)
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
		items = append(items, cmdItem{label: label, command: cmd})
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
	if qCmd > 0 {
		command, err := getNthComand(items, qCmd)
		if err != nil {
			log.Print(err)
			os.Exit(-1)
		}
		fmt.Printf("\n%s\n\n", command)
		status, err := execCommand(command)
		if err != nil {
			log.Print(err)
		}
		os.Exit(status)
	}
	command, err := selectComand(items)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	if status, err := execCommand(command); err != nil {
		log.Print(err)
		os.Exit(status)
	}
	os.Exit(0)
}
