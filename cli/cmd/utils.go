package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func AutocompleteServiceName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	services, err := Client.Services()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var names []string
	for _, service := range services {
		names = append(names, service.Config.Name)
	}

	var filtered []string
	for _, name := range names {
		if strings.HasPrefix(name, toComplete) {
			filtered = append(filtered, name)
		}
	}

	return filtered, cobra.ShellCompDirectiveNoFileComp
}

type Table [][]string

func (t Table) Print() {
	for i, row := range t {
		for _, cell := range row {
			if i == 0 {
				fmt.Printf("\033[1m%-20s\033[0m", cell)
			} else {
				fmt.Printf("%-20s", cell)
			}
		}
		fmt.Println()
	}
}

func PrintlnBold(text string) {
	fmt.Printf("\033[1m%s\033[0m\n", text)
}

func Prompt(prompt string, dest *string) {
	PrintlnBold(prompt)
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	*dest = strings.TrimSpace(s)
}

func PromptBool(prompt string, dest *bool) {
	PrintlnBold(prompt + " (y/n)")
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	s = strings.TrimSpace(s)
	if s == "y" {
		*dest = true
	} else {
		*dest = false
	}
}

func PromptInt(prompt string, dest *int) {
	PrintlnBold(prompt)
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	_, err := fmt.Sscanf(s, "%d", dest)
	if err != nil {
		fmt.Println("Invalid input")
		PromptInt(prompt, dest)
	}
}
