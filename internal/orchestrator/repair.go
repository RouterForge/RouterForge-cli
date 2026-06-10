package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/engine"
)

type ValidationCommand struct {
	Name     string
	Args     []string
	Required bool
}

type CommandResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout,omitempty"`
	Stderr   string        `json:"stderr,omitempty"`
	Duration time.Duration `json:"duration"`
}

type ValidationResult struct {
	Success     bool            `json:"success"`
	ProjectType string          `json:"project_type"`
	Errors      []string        `json:"errors,omitempty"`
	Commands    []CommandResult `json:"commands,omitempty"`
}

func (hm *HeadManager) RepairUntilValid(maxRetries int) error {
	if hm.projectDir == "" {
		hm.projectDir = "."
	}

	pterm.DefaultSection.Printfln("Repair Loop — Build, Run, Test, Repair")
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result := ValidateProject(hm.projectDir)
		hm.writeValidationArtifact(result, attempt)
		if result.Success {
			hm.logDecision("validation", fmt.Sprintf("Project validation passed on attempt %d", attempt+1))
			hm.WriteTrace("validation_passed", "repair_engine", "repair", "", "completed", result.ProjectType)
			pterm.Success.Printfln("Validation passed (%s)", result.ProjectType)
			return nil
		}

		pterm.Warning.Printfln("Validation failed on attempt %d/%d", attempt+1, maxRetries+1)
		for _, err := range result.Errors {
			pterm.Warning.Printfln("  %s", err)
		}
		if attempt == maxRetries {
			hm.WriteTrace("validation_failed", "repair_engine", "repair", "", "failed", strings.Join(result.Errors, "; "))
			return fmt.Errorf("validation failed after %d repair attempts: %s", maxRetries, strings.Join(result.Errors, "; "))
		}

		if err := hm.repairProject(result, attempt+1); err != nil {
			hm.WriteTrace("repair_failed", "repair_engine", "repair", "", "failed", err.Error())
			return err
		}
	}
	return fmt.Errorf("repair loop exited unexpectedly")
}

func ValidateProject(projectDir string) ValidationResult {
	result := ValidationResult{ProjectType: detectProjectType(projectDir)}

	if result.ProjectType == "empty" {
		result.Errors = append(result.Errors, "no generated project files found")
		return result
	}

	switch result.ProjectType {
	case "go":
		if !fileExists(filepath.Join(projectDir, "go.mod")) {
			result.Errors = append(result.Errors, "go.mod is missing")
			return result
		}
		result.Commands = append(result.Commands, runValidationCommand(projectDir, ValidationCommand{Name: "go", Args: []string{"test", "./..."}, Required: true}))
	case "node":
		for _, cmd := range nodeValidationCommands(projectDir) {
			result.Commands = append(result.Commands, runValidationCommand(projectDir, cmd))
		}
	case "static-web":
		result.Errors = append(result.Errors, validateStaticWeb(projectDir)...)
		for _, js := range localJavaScriptFiles(projectDir) {
			if _, err := exec.LookPath("node"); err == nil {
				rel, _ := filepath.Rel(projectDir, js)
				result.Commands = append(result.Commands, runValidationCommand(projectDir, ValidationCommand{Name: "node", Args: []string{"--check", rel}, Required: true}))
			}
		}
	default:
		result.Errors = append(result.Errors, "unable to determine how to validate generated project")
	}

	for _, cmd := range result.Commands {
		if cmd.ExitCode != 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("%s failed", cmd.Command))
		}
	}
	result.Success = len(result.Errors) == 0
	return result
}

func (hm *HeadManager) repairProject(result ValidationResult, attempt int) error {
	prompt := fmt.Sprintf(`Repair this generated software project so it becomes working software.

Project: %s
Goal: %s
Tech Stack: %s
Description: %s
Detected type: %s
Repair attempt: %d

Validation result:
%s

Current project files:
%s

Rules:
- Return ONLY FILE sections in this exact format:
FILE: relative/path/to/file.ext
---
<complete file contents>
- Include complete replacement files, not diffs.
- Use paths relative to the project root.
- If a manifest or entry point is missing, create it.
- Remove invalid markdown fences and separator lines from source code.
- Prefer a minimal complete working implementation over partial architecture.
- For a browser app, ensure index.html references existing local assets and the required UI is actually wired together.
- For a Go app, include go.mod and code that passes go test ./....`,
		hm.project.Name, hm.project.Goal, hm.project.TechStack, hm.project.Description, result.ProjectType, attempt, validationJSON(result), projectSnapshot(hm.projectDir))

	llm := engine.NewLLMClient(hm.model)
	llm.AgentID = "repair_engine"
	llm.Phase = "repair"
	llm.CostHandler = func(model, agentID, phase string, usage engine.Usage) {
		hm.costTracker.Track(model, agentID, phase, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.Cost)
	}

	response, err := llm.Chat("You are a senior software repair engineer. Return only FILE sections.", prompt)
	if err != nil {
		return fmt.Errorf("repair LLM call failed: %w", err)
	}
	files, err := writeFileSections(hm.projectDir, stripMarkdown(response))
	if err != nil {
		return err
	}
	if files == 0 {
		return fmt.Errorf("repair produced no FILE sections")
	}
	hm.logDecision("repair", fmt.Sprintf("Repair attempt %d wrote %d file(s)", attempt, files))
	hm.WriteTrace("repair_applied", "repair_engine", "repair", "", "completed", fmt.Sprintf("wrote %d files", files))
	pterm.Success.Printfln("Repair attempt %d wrote %d file(s)", attempt, files)
	return nil
}

func detectProjectType(projectDir string) string {
	files := generatedFiles(projectDir)
	if len(files) == 0 {
		return "empty"
	}
	if fileExists(filepath.Join(projectDir, "package.json")) {
		return "node"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			return "go"
		}
	}
	if fileExists(filepath.Join(projectDir, "index.html")) {
		return "static-web"
	}
	return "unknown"
}

func generatedFiles(projectDir string) []string {
	var files []string
	filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".routerforge" || d.Name() == "node_modules" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".log") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files
}

func runValidationCommand(projectDir string, command ValidationCommand) CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return CommandResult{
		Command:  strings.TrimSpace(command.Name + " " + strings.Join(command.Args, " ")),
		ExitCode: exitCode,
		Stderr:   truncate(string(out), 4000),
		Duration: time.Since(start),
	}
}

func nodeValidationCommands(projectDir string) []ValidationCommand {
	scripts := map[string]string{}
	data, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err == nil {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			scripts = pkg.Scripts
		}
	}
	var commands []ValidationCommand
	if _, ok := scripts["build"]; ok {
		commands = append(commands, ValidationCommand{Name: "npm", Args: []string{"run", "build"}, Required: true})
	}
	if _, ok := scripts["test"]; ok {
		commands = append(commands, ValidationCommand{Name: "npm", Args: []string{"test"}, Required: true})
	}
	if len(commands) == 0 {
		commands = append(commands, ValidationCommand{Name: "npm", Args: []string{"--version"}, Required: true})
	}
	return commands
}

func validateStaticWeb(projectDir string) []string {
	indexPath := filepath.Join(projectDir, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return []string{"index.html is missing"}
	}
	var errors []string
	re := regexp.MustCompile(`(?i)(?:src|href)=["']([^"']+)["']`)
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		ref := strings.TrimSpace(match[1])
		if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "mailto:") {
			continue
		}
		ref = strings.TrimPrefix(ref, "/")
		ref = strings.Split(ref, "?")[0]
		if !fileExists(filepath.Join(projectDir, ref)) {
			errors = append(errors, fmt.Sprintf("index.html references missing asset %s", match[1]))
		}
	}
	return errors
}

func localJavaScriptFiles(projectDir string) []string {
	var files []string
	for _, f := range generatedFiles(projectDir) {
		if strings.HasSuffix(f, ".js") {
			files = append(files, f)
		}
	}
	return files
}

func projectSnapshot(projectDir string) string {
	files := generatedFiles(projectDir)
	var b strings.Builder
	for i, file := range files {
		rel, _ := filepath.Rel(projectDir, file)
		b.WriteString("- " + rel + "\n")
		if i >= 11 {
			continue
		}
		if shouldIncludeSnapshot(file) {
			data, err := os.ReadFile(file)
			if err == nil {
				b.WriteString("```\n")
				b.WriteString(truncate(string(data), 1400))
				b.WriteString("\n```\n")
			}
		}
	}
	if len(files) == 0 {
		b.WriteString("(no generated files)\n")
	}
	return b.String()
}

func shouldIncludeSnapshot(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".html", ".css", ".js", ".json", ".md":
		return true
	default:
		return false
	}
}

func validationJSON(result ValidationResult) string {
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

func (hm *HeadManager) writeValidationArtifact(result ValidationResult, attempt int) {
	if hm.projectDir == "" {
		return
	}
	artifactsDir := filepath.Join(hm.projectDir, ".routerforge", "artifacts")
	os.MkdirAll(artifactsDir, 0755)
	data, _ := json.MarshalIndent(result, "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, fmt.Sprintf("validation-%d.json", attempt)), data, 0644)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... truncated ..."
}
