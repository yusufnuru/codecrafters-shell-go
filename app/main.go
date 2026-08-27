package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Stdout io.Writer
	Stderr io.Writer
	exec   func(stdout, stderr io.Writer, args []string) error
}

func (cmd Command) Run(args []string) error {
	return cmd.exec(cmd.Stdout, cmd.Stderr, args[1:])
}

func handleExit(stdout, stderr io.Writer, args []string) error {
	os.Exit(0)
	return nil
}

func handleCD(stdout, stderr io.Writer, args []string) error {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "cd: missing argument")
		return fmt.Errorf("cd: missing argument")
	}

	path := args[0]
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(stderr, "cd:", err)
			return err
		}
		path = home
	}

	if err := os.Chdir(path); err != nil {
		fmt.Fprintln(stderr, "cd: "+args[0]+": No such file or directory")
		return err
	}

	return nil
}

func handlePWD(stdout, stderr io.Writer, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, "pwd:", err)
		return err
	} else {
		fmt.Fprintln(stdout, dir)
	}
	return nil
}

func handleEcho(stdout, stderr io.Writer, args []string) error {
	fmt.Fprintln(stdout, strings.Join(args, " "))
	return nil
}

func handleType(stdout, stderr io.Writer, args []string) error {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "type: missing argument")
		return fmt.Errorf("type: missing argument")
	}

	if _, exists := builtins[args[0]]; exists {
		fmt.Fprintln(stdout, args[0]+" is a shell builtin")
	} else if path, _ := exec.LookPath(args[0]); path != "" {
		fmt.Fprintln(stdout, args[0]+" is "+path)
	} else {
		fmt.Fprintln(stderr, args[0]+" not found")
	}
	return nil
}

var builtins map[string]Command

func init() {
	builtins = map[string]Command{
		"exit": {exec: handleExit},
		"echo": {exec: handleEcho},
		"type": {exec: handleType},
		"pwd":  {exec: handlePWD},
		"cd":   {exec: handleCD},
	}
}

var stdinReader = bufio.NewReader(os.Stdin)

func readCommand() []string {
	fmt.Fprint(os.Stdout, "$ ")

	line, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	line = strings.TrimSpace(line)
	return tokenize(line)
}

func executeCommand(args []string) {
	if len(args) == 0 {
		return
	}

	stdout := os.Stdout
	if len(args) > 2 && (args[len(args)-2] == ">" || args[len(args)-2] == "1>") {
		outputFile, err := os.Create(args[len(args)-1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: cannot create", args[len(args)-1])
			return
		}
		defer outputFile.Close()
		stdout = outputFile
		args = args[:len(args)-2]
	}

	if cmd, builtin := builtins[args[0]]; builtin {
		cmd.Stderr = os.Stderr
		cmd.Stdout = stdout
		cmd.Run(args)
	} else if _, err := exec.LookPath(args[0]); err == nil {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		cmd.Wait()
	} else {
		fmt.Fprintln(os.Stderr, args[0]+": command not found")
	}
}

func main() {
	for {
		args := readCommand()
		executeCommand(args)
	}
}

func tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '\'':
			if inDoubleQuote {
				current.WriteByte(ch)
			} else {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if inSingleQuote {
				current.WriteByte(ch)
			} else {
				inDoubleQuote = !inDoubleQuote
			}
		case ' ':
			if inSingleQuote || inDoubleQuote {
				current.WriteByte(ch)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case '\\':
			if inSingleQuote {
				current.WriteByte(ch)
			} else {
				i++
				if i < len(input) {
					current.WriteByte(input[i])
				}
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
