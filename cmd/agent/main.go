package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yourname/devops-agent/internal/agent"
	"github.com/yourname/devops-agent/internal/config"
)

func main() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║    DevOps Agent  (Sonnet 4.6)        ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	a := agent.New(cfg)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Tools available: shell, read_file, write_file, http_probe, git_info")
	fmt.Println("Type 'exit' to quit.")
	fmt.Println()

	for {
		fmt.Print("You › ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		response, err := a.Run(context.Background(), input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
			continue
		}
		fmt.Printf("\nAgent › %s\n\n", response)
	}
}
