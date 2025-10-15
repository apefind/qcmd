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
	"runtime"
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

func (item cmdItem) FilterValue() string {
	return ""
}

func (item cmdItem) Label(k int) string {
	return fmt.Sprintf("%d. %s", k+1, item.label)
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

func (d cmdItemDelegate) Render(w io.Writer, m itemlist.Model, k int, itm itemlist.Item) {
	item, ok := itm.(cmdItem)
	if !ok {
		return
	}
	if k == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("> "+item.Label(k)))
	} else {
		fmt.Fprint(w, itemStyle.Render(item.Label(k)))
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

func getShellCommand(command string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("pwsh", "-command", command)

	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd
}

func execShellCommand(command string) (int, error) {
	cmd := getShellCommand(command)
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

func readCommandItems(fp string) ([]itemlist.Item, error) {
	var items []itemlist.Item
	file, err := os.Open(fp)
	if err != nil {
		return items, err
	}
	defer file.Close()
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
	return items, scanner.Err()
}

func printCommandItems(items []itemlist.Item) {
	fmt.Println("")
	for k, itm := range items {
		item, ok := itm.(cmdItem)
		if !ok {
			continue
		}
		fmt.Printf("    %s\n", item.Label(k))
	}
	fmt.Println("")
}

func getCommand(items []itemlist.Item, k int) (string, error) {

	getNthComand := func(items []itemlist.Item, n int) (string, error) {
		if n > len(items) {
			return "", errors.New("item not found")
		}
		item, ok := items[n-1].(cmdItem)
		if !ok {
			return "", errors.New("not a cmdItem")
		}
		return item.command, nil
	}

	selectComand := func(items []itemlist.Item) (string, error) {
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

	if k > 0 {
		command, err := getNthComand(items, k)
		fmt.Printf("\n%s\n\n", command)
		return command, err
	}
	return selectComand(items)
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-f .qcmd] [-n cmd] [-l]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	var qCmdFile string
	var qNthCmd int
	var qListCmds bool
	flag.StringVar(&qCmdFile, "f", ".qcmd", ".qcmd filepath")
	flag.IntVar(&qNthCmd, "n", 0, "Execute the n-th command")
	flag.BoolVar(&qListCmds, "l", false, "List available commands")
	flag.Parse()
	items, err := readCommandItems(qCmdFile)
	if err != nil {
		log.Fatal(err)
	}
	if qListCmds {
		printCommandItems(items)
		os.Exit(0)
	}
	command, err := getCommand(items, qNthCmd)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chdir(filepath.Dir(qCmdFile)); err != nil {
		log.Fatal(err)
	}
	status, err := execShellCommand(command)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(status)
}
