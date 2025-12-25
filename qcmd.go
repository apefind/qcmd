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
	"gopkg.in/yaml.v3"
)

type CmdEntry struct {
	Label   string      `yaml:"label"`
	Command string      `yaml:"command,omitempty"`
	Exit    bool        `yaml:"exit,omitempty"`
	Entries []*CmdEntry `yaml:"entries,omitempty"`
}

func (e CmdEntry) String() string {
	out, err := yaml.Marshal(e)
	if err != nil {
		panic(err)
	}
	return string(out)
}

func breadcrumb(path []*CmdEntry) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = p.Label
	}
	return strings.Join(parts, " › ")
}

func indentWidth(line string, tabSize int) int {
	width := 0
	for _, r := range line {
		switch r {
		case ' ':
			width++
		case '\t':
			width += tabSize
		default:
			return width
		}
	}
	return width
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
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
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

func skip(ln string) bool {
	ln = strings.TrimSpace(ln)
	return ln == "" || strings.HasPrefix(ln, "#") && !strings.HasPrefix(ln, "#tab=")
}

func getCmdEntry(ln string) *CmdEntry {
	ln = strings.TrimSpace(strings.Trim(ln, "␍"))
	exit := true
	if strings.HasSuffix(ln, "␍") { // optional suffix to return to menu
		exit = false
		ln = strings.TrimSpace(ln[:len(ln)-1])
	}
	if strings.HasSuffix(ln, ":") {
		return &CmdEntry{Label: ln[:len(ln)-1], Exit: exit}
	}
	s := strings.SplitN(ln, ":", 2)
	label := strings.TrimSpace(s[0])
	if len(s) == 1 {
		return &CmdEntry{Label: label, Command: label, Exit: exit}
	}
	return &CmdEntry{Label: label, Command: strings.TrimSpace(s[1]), Exit: exit}
}

// readQCmd reads the file and returns root CmdEntry and tab size
func readQCmd(path string) (*CmdEntry, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	tabSize := 4 // default
	root := &CmdEntry{Label: "Main Menu"}
	stack := []*CmdEntry{root}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ln := scanner.Text()
		trim := strings.TrimSpace(ln)

		// check for tab directive
		if strings.HasPrefix(trim, "#tab=") || strings.HasPrefix(trim, "#indent=") {
			parts := strings.SplitN(trim, "=", 2)
			if len(parts) == 2 {
				if val, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && val > 0 {
					tabSize = val
				}
			}
			continue
		}

		if skip(ln) {
			continue
		}

		indent := indentWidth(ln, tabSize) / tabSize
		entry := getCmdEntry(strings.TrimSpace(ln))

		if indent+1 > len(stack) {
			stack = append(stack, stack[len(stack)-1])
		} else {
			stack = stack[:indent+1]
		}
		parent := stack[len(stack)-1]
		parent.Entries = append(parent.Entries, entry)
		if len(stack) == indent+1 {
			stack = append(stack, entry)
		} else {
			stack[indent+1] = entry
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	return root, tabSize, nil
}

func entryOptions(entries []*CmdEntry) []huh.Option[*CmdEntry] {
	opts := make([]huh.Option[*CmdEntry], 0, len(entries))
	for _, e := range entries {
		label := e.Label
		if len(e.Entries) > 0 {
			label = "▸ " + label
		} else {
			label = "▶ " + label
		}
		opts = append(opts, huh.NewOption(label, e))
	}
	return opts
}

func runMenu(menu *CmdEntry, path []*CmdEntry) error {
	for {
		var selected *CmdEntry
		km := huh.NewDefaultKeyMap()
		km.Quit.SetKeys("esc", "ctrl+c")

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[*CmdEntry]().
					Title(breadcrumb(path)).
					Options(entryOptions(menu.Entries)...).
					Value(&selected),
			),
		).WithTheme(huh.ThemeCatppuccin()).
			WithWidth(60).
			WithKeyMap(km)

		err := form.Run()
		if err == huh.ErrUserAborted {
			return nil
		}
		if err != nil {
			return err
		}

		if len(selected.Entries) > 0 {
			if exitErr := runMenu(selected, append(path, selected)); exitErr != nil {
				return exitErr
			}
			continue
		}

		code, err := runShellCmd(selected.Command)
		if err != nil {
			fmt.Printf("command failed (%d): %v\n", code, err)
		}

		if selected.Exit {
			return nil // normal exit, don't panic
		}
	}
}

func main() {
	qcmdPath := flag.String("file", ".qcmd", "path to the QCMD file")
	flag.Parse()
	if _, err := os.Stat(*qcmdPath); err != nil {
		fmt.Printf("Error: cannot open file %s: %v\n", *qcmdPath, err)
		os.Exit(1)
	}
	menu, _, err := readQCmd(*qcmdPath)
	if err != nil {
		panic(err)
	}
	if err := runMenu(menu, []*CmdEntry{menu}); err != nil {
		panic(err)
	}
}
