package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/huh"
)

type CmdEntry struct {
	Label     string
	Command   string
	Exit      bool
	Separator bool
	Entries   []*CmdEntry
}

func breadcrumb(path []*CmdEntry) string {
	p := make([]string, len(path))
	for i, e := range path {
		p[i] = e.Label
	}
	return strings.Join(p, " › ")
}

func indentWidth(line string, tabSize int) int {
	w := 0
	for _, r := range line {
		switch r {
		case ' ':
			w++
		case '\t':
			w += tabSize
		default:
			return w
		}
	}
	return w
}

func skip(ln string) bool {
	ln = strings.TrimSpace(ln)
	return ln == "" || strings.HasPrefix(ln, "#") && !strings.HasPrefix(ln, "#tab=")
}

func getCmdEntry(ln string) *CmdEntry {
	ln = strings.TrimSpace(ln)

	if ln == "---" {
		return &CmdEntry{Separator: true}
	}

	exit := true
	if strings.HasSuffix(ln, "␍") {
		exit = false
		ln = strings.TrimSpace(strings.TrimSuffix(ln, "␍"))
	}

	if strings.HasSuffix(ln, ":") {
		return &CmdEntry{Label: ln[:len(ln)-1], Exit: exit}
	}

	s := strings.SplitN(ln, ": ", 2)
	label := strings.TrimSpace(s[0])

	if len(s) == 1 {
		return &CmdEntry{Label: label, Command: label, Exit: exit}
	}

	return &CmdEntry{Label: label, Command: strings.TrimSpace(s[1]), Exit: exit}
}

func readQCmd(path string) (*CmdEntry, int, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	tabSize := 4
	root := &CmdEntry{Label: "QCmd"}
	stack := []*CmdEntry{root}

	sc := bufio.NewScanner(f)

	for sc.Scan() {

		raw := sc.Text()
		trim := strings.TrimSpace(raw)

		if strings.HasPrefix(trim, "#tab=") || strings.HasPrefix(trim, "#indent=") {
			if v, err := strconv.Atoi(strings.Split(trim, "=")[1]); err == nil && v > 0 {
				tabSize = v
			}
			continue
		}

		if skip(raw) {
			continue
		}

		level := indentWidth(raw, tabSize) / tabSize
		entry := getCmdEntry(trim)

		if level >= len(stack) {
			level = len(stack) - 1
		}

		stack = stack[:level+1]

		parent := stack[len(stack)-1]
		parent.Entries = append(parent.Entries, entry)

		if !entry.Separator {
			stack = append(stack, entry)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, 0, err
	}

	return root, tabSize, nil
}

func getShellCmd(command string) *exec.Cmd {

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("pwsh"); err == nil {
			cmd = exec.Command("pwsh", "-Command", command)
		} else {
			cmd = exec.Command("powershell", "-Command", command)
		}
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd
}

func runShellCmd(command string) (int, error) {

	cmd := getShellCmd(command)

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

func flattenCommands(e *CmdEntry, list *[]*CmdEntry) {

	for _, c := range e.Entries {

		if c.Separator {
			continue
		}

		if len(c.Entries) > 0 {
			flattenCommands(c, list)
		} else {
			*list = append(*list, c)
		}
	}
}

func entryOptions(entries []*CmdEntry) []huh.Option[*CmdEntry] {

	opts := make([]huh.Option[*CmdEntry], 0, len(entries))

	for _, e := range entries {

		if e.Separator {
			opts = append(opts, huh.NewOption("────────", e))
			continue
		}

		label := e.Label

		if len(e.Entries) > 0 {
			label = "▸ " + label
		} else {
			label = "◇ " + label
		}

		opts = append(opts, huh.NewOption(label, e))
	}

	return opts
}

func runPalette(root *CmdEntry) error {

	var cmds []*CmdEntry
	flattenCommands(root, &cmds)

	opts := make([]huh.Option[*CmdEntry], 0, len(cmds))

	for _, c := range cmds {
		opts = append(opts, huh.NewOption("◇ "+c.Label, c))
	}

	var selected *CmdEntry

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[*CmdEntry]().
				Title("Command Palette").
				Filtering(true).
				Options(opts...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return err
	}

	_, err = runShellCmd(selected.Command)
	return err
}

func runMenu(menu *CmdEntry, path []*CmdEntry) error {

	for {

		var selected *CmdEntry

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[*CmdEntry]().
					Title(breadcrumb(path)).
					Filtering(true).
					Options(entryOptions(menu.Entries)...).
					Value(&selected),
			),
		).WithTheme(huh.ThemeCatppuccin())

		err := form.Run()

		if err == huh.ErrUserAborted {
			return nil
		}

		if err != nil {
			return err
		}

		if selected.Separator {
			continue
		}

		if len(selected.Entries) > 0 {
			if err := runMenu(selected, append(path, selected)); err != nil {
				return err
			}
			continue
		}

		code, err := runShellCmd(selected.Command)

		if err != nil {
			fmt.Printf("command failed (%d): %v\n", code, err)
		}

		if selected.Exit {
			return nil
		}
	}
}

func main() {

	qcmdPath := flag.String("f", ".qcmd", "path to QCMD file")
	palette := flag.Bool("p", false, "command palette")

	flag.Parse()

	if _, err := os.Stat(*qcmdPath); err != nil {
		fmt.Printf("Error: cannot open file %s: %v\n", *qcmdPath, err)
		os.Exit(1)
	}

	menu, _, err := readQCmd(*qcmdPath)
	if err != nil {
		panic(err)
	}

	if *palette {
		if err := runPalette(menu); err != nil {
			panic(err)
		}
		return
	}

	if err := runMenu(menu, []*CmdEntry{menu}); err != nil {
		panic(err)
	}
}
