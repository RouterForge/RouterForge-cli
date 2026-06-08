package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type UserInputProvider interface {
	Ask(question string) (string, error)
	Confirm(message string) (bool, error)
	Choose(prompt string, options []string) (string, error)
}

type TerminalUserProxy struct {
	reader *bufio.Reader
}

func NewTerminalUserProxy() *TerminalUserProxy {
	return &TerminalUserProxy{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (t *TerminalUserProxy) Ask(question string) (string, error) {
	fmt.Println("❓", question)
	fmt.Print("> ")
	answer, err := t.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}

func (t *TerminalUserProxy) Confirm(message string) (bool, error) {
	fmt.Print(message + " [y/N]: ")
	answer, err := t.reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func (t *TerminalUserProxy) Choose(prompt string, options []string) (string, error) {
	fmt.Println("❓", prompt)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Print("Enter number (1-" + fmt.Sprintf("%d", len(options)) + "): ")
	answer, err := t.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	for i, opt := range options {
		if fmt.Sprintf("%d", i+1) == answer {
			return opt, nil
		}
	}
	if answer != "" {
		return answer, nil
	}
	if len(options) > 0 {
		return options[0], nil
	}
	return "", nil
}

type PipelineConfig struct {
	Teams   []string `json:"teams"`
	Profile string   `json:"profile"`
}

type ConfigurableUserProxy struct {
	Provider  UserInputProvider
	Responses map[string]string
}

func NewConfigurableUserProxy(provider UserInputProvider) *ConfigurableUserProxy {
	return &ConfigurableUserProxy{
		Provider:  provider,
		Responses: make(map[string]string),
	}
}

func (c *ConfigurableUserProxy) Ask(question string) (string, error) {
	if answer, ok := c.Responses[question]; ok {
		return answer, nil
	}
	return c.Provider.Ask(question)
}

func (c *ConfigurableUserProxy) Confirm(message string) (bool, error) {
	return c.Provider.Confirm(message)
}

func (c *ConfigurableUserProxy) Choose(prompt string, options []string) (string, error) {
	return c.Provider.Choose(prompt, options)
}
