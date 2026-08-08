package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	builtins := []string{"echo", "type", "exit", "pwd", "cd"}

	for {
		fmt.Print("$ ")

		// Wait for user input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		input = strings.TrimSpace(input)
		tokens := tokenize(input)
		if len(tokens) == 0 {
			continue
		}

		cmd := tokens[0]
		args := tokens[1:]

		if len(tokens) > 2 && ( tokens[len(tokens)-2] == "1>" || tokens[len(tokens)-2] == "2>" 	) {
			file, err := os.Create(tokens[2])
			if err != nil {
				panic(err)
			}
			fmt.Fprintln(file, strings.Join(args[:1], " "))
			file.Close()
			args = args[:len(args)-2]
			continue
		}

		switch cmd {
		case "exit":
			os.Exit(0)
		case "echo":
			fmt.Println(strings.Join(args, " "))

		case "type":
			if len(tokens) < 2 {
				fmt.Fprintln(os.Stderr, "type: missing argument")
				continue
			}
			if slices.Contains(builtins, args[0]) {
				fmt.Println(args[0] + " is a shell builtin")
			} else if path, _ := exec.LookPath(tokens[1]); path != "" {
				fmt.Println(args[0] + " is " + path)
			} else {
				fmt.Println(args[0] + " not found")
			}
		case "pwd":
			dir, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, "pwd:", err)
			} else {
				fmt.Println(dir)
			}
		case "cd":
			path := args[0]
			if path == "~" {
				home, err := os.UserHomeDir()
				if err != nil {
					fmt.Fprintln(os.Stderr, "cd:", err)
					continue
				}
				path = home
			}
			if err := os.Chdir(path); err != nil {
				fmt.Fprintln(os.Stderr, "cd: "+args[0]+": No such file or directory")
			}
		default:
			if _, err := exec.LookPath(cmd); err == nil {
				c := exec.Command(cmd, args...)
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					fmt.Fprintln(os.Stderr, "Error executing command:", err)
				}
			} else {
				fmt.Println(cmd + ": command not found")
			}
		}
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
