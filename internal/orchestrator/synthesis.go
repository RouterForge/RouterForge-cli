package orchestrator

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/pkg/models"
)

type SynthesizedAgent struct {
	Role        string
	Description string
	Tools       []string
	Tasks       []string
	Domain      string
}

func SynthesizeTeam(project *models.Project, domains []string) []SynthesizedAgent {
	var agents []SynthesizedAgent
	goal := strings.ToLower(project.Goal)
	stack := strings.ToLower(project.TechStack)
	features := strings.ToLower(project.Description)

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		switch domain {
		case "frontend", "ui", "web":
			agents = append(agents, SynthesizedAgent{
				Role: "Frontend Developer", Domain: domain,
				Description: fmt.Sprintf("Build UI components for %s project using %s", project.Name, stack),
				Tools:       []string{"read", "write", "search", "bash"},
				Tasks:       deriveFrontendTasks(goal, stack, features),
			})
			agents = append(agents, SynthesizedAgent{
				Role: "UI/UX Designer", Domain: domain,
				Description: fmt.Sprintf("Design user flows and interfaces for %s", project.Name),
				Tools:       []string{"read", "write", "search"},
				Tasks:       []string{fmt.Sprintf("Design component hierarchy for %s", project.Name), "Create responsive layout templates"},
			})

		case "backend", "api", "server":
			agents = append(agents, SynthesizedAgent{
				Role: "API Developer", Domain: domain,
				Description: fmt.Sprintf("Build backend API for %s", project.Name),
				Tools:       []string{"read", "write", "search", "bash"},
				Tasks:       deriveBackendTasks(goal, stack, features),
			})
			agents = append(agents, SynthesizedAgent{
				Role: "Database Engineer", Domain: domain,
				Description: "Design and implement data storage",
				Tools:       []string{"read", "write", "bash"},
				Tasks:       []string{"Design data models and schema", "Implement data access layer", "Add data validation and migrations"},
			})

		case "security":
			agents = append(agents, SynthesizedAgent{
				Role: "Security Engineer", Domain: domain,
				Description: "Audit and harden security",
				Tools:       []string{"read", "search", "bash"},
				Tasks:       []string{"Review authentication flows", "Check for common vulnerabilities", "Validate input sanitization", "Audit dependency security"},
			})

		case "qa", "test", "quality":
			agents = append(agents, SynthesizedAgent{
				Role: "QA Engineer", Domain: domain,
				Description: "Ensure quality through testing",
				Tools:       []string{"read", "write", "bash"},
				Tasks:       []string{fmt.Sprintf("Write unit tests for %s core modules", project.Name), "Integration test setup", "End-to-end test scenarios", "Performance benchmark suite"},
			})

		case "devops", "infra", "deploy":
			agents = append(agents, SynthesizedAgent{
				Role: "DevOps Engineer", Domain: domain,
				Description: "Set up deployment and infrastructure",
				Tools:       []string{"read", "write", "bash"},
				Tasks:       []string{"Create CI/CD pipeline configuration", "Dockerize the application", "Set up monitoring and logging", "Write deployment automation"},
			})

		case "browser":
			agents = append(agents, SynthesizedAgent{
				Role: "Browser Automation Engineer", Domain: domain,
				Description: "Implement browser-based features and tests",
				Tools:       []string{"read", "write", "bash", "browser"},
				Tasks:       []string{fmt.Sprintf("Write browser automation for %s", project.Name), "Capture screenshots of key pages", "Test multi-tab workflows", "Extract and validate page content"},
			})

		case "cli", "terminal":
			agents = append(agents, SynthesizedAgent{
				Role: "CLI Developer", Domain: domain,
				Description: "Build command-line interface",
				Tools:       []string{"read", "write", "bash"},
				Tasks:       []string{fmt.Sprintf("Design CLI command structure for %s", project.Name), "Implement core commands", "Add help and completion", "Write CLI tests"},
			})

		case "docs", "content", "documentation":
			agents = append(agents, SynthesizedAgent{
				Role: "Technical Writer", Domain: domain,
				Description: "Create project documentation",
				Tools:       []string{"read", "write", "search"},
				Tasks:       []string{fmt.Sprintf("Write README and getting-started guide for %s", project.Name), "Create API documentation", "Write user manual", "Add architecture overview docs"},
			})

		case "data", "analytics":
			agents = append(agents, SynthesizedAgent{
				Role: "Data Engineer", Domain: domain,
				Description: "Build data processing pipeline",
				Tools:       []string{"read", "write", "bash"},
				Tasks:       []string{"Design data pipeline architecture", "Implement data collection and storage", "Build analytics queries", "Create data visualization"},
			})

		default:
			agents = append(agents, SynthesizedAgent{
				Role: fmt.Sprintf("%s Engineer", strings.Title(domain)), Domain: domain,
				Description: fmt.Sprintf("Handle %s concerns for %s", domain, project.Name),
				Tools:       []string{"read", "write", "search", "bash"},
				Tasks: []string{
					fmt.Sprintf("Analyze %s requirements for %s", domain, project.Name),
					fmt.Sprintf("Implement %s components", domain),
					fmt.Sprintf("Test %s functionality", domain),
					fmt.Sprintf("Document %s decisions", domain),
				},
			})
		}
	}
	return agents
}

func deriveFrontendTasks(goal, stack, features string) []string {
	var tasks []string

	if strings.Contains(goal, "website") || strings.Contains(goal, "site") || strings.Contains(goal, "landing") || strings.Contains(goal, "page") {
		tasks = append(tasks, "Create landing page with hero section and feature showcase")
		tasks = append(tasks, "Build responsive navigation and footer components")
		tasks = append(tasks, "Add interactive elements with CSS animations")
	}
	if strings.Contains(stack, "react") || strings.Contains(stack, "vue") || strings.Contains(stack, "svelte") || strings.Contains(stack, "angular") {
		tasks = append(tasks, "Set up component library with state management")
		tasks = append(tasks, "Implement routing and lazy loading")
	} else {
		tasks = append(tasks, "Build static HTML/CSS pages with modern layout")
		tasks = append(tasks, "Add client-side interactivity")
	}
	if strings.Contains(features, "auth") || strings.Contains(features, "login") || strings.Contains(features, "user") {
		tasks = append(tasks, "Implement login/registration UI")
	}
	tasks = append(tasks, "Ensure responsive design and cross-browser compatibility")

	return tasks
}

func deriveBackendTasks(goal, stack, features string) []string {
	var tasks []string

	if strings.Contains(stack, "go") || strings.Contains(stack, "golang") || strings.Contains(goal, "cli") || strings.Contains(goal, "tool") {
		tasks = append(tasks, "Design and implement Go package structure")
		tasks = append(tasks, "Build core logic and business rules")
		tasks = append(tasks, "Add CLI command handlers if applicable")
	} else {
		tasks = append(tasks, "Design API endpoints and request/response models")
		tasks = append(tasks, "Implement REST or GraphQL API")
	}
	if strings.Contains(features, "auth") || strings.Contains(features, "login") || strings.Contains(features, "user") {
		tasks = append(tasks, "Implement authentication and authorization")
	}
	if strings.Contains(features, "database") || strings.Contains(features, "db") || strings.Contains(features, "persist") || strings.Contains(features, "store") {
		tasks = append(tasks, "Set up database schema and migrations")
		tasks = append(tasks, "Implement data access layer")
	}
	tasks = append(tasks, "Write API tests and validation")
	tasks = append(tasks, "Add error handling and logging")

	return tasks
}

func (hm *HeadManager) designSynthesized() error {
	pterm.Info.Println("Synthesizing teams from project requirements...")

	featureHints := hm.project.Goal + " " + hm.project.TechStack + " " + hm.project.Description
	featureHints = strings.ToLower(featureHints)

	domains := []string{"frontend", "backend"}
	if strings.Contains(featureHints, "security") || strings.Contains(featureHints, "auth") {
		domains = append(domains, "security")
	}
	if strings.Contains(featureHints, "test") || strings.Contains(featureHints, "qa") || strings.Contains(featureHints, "quality") {
		domains = append(domains, "qa")
	}
	if strings.Contains(featureHints, "deploy") || strings.Contains(featureHints, "ci") || strings.Contains(featureHints, "docker") || strings.Contains(featureHints, "infra") {
		domains = append(domains, "devops")
	}
	if strings.Contains(featureHints, "browser") || strings.Contains(featureHints, "scrape") || strings.Contains(featureHints, "web") {
		domains = append(domains, "browser")
	}
	if strings.Contains(featureHints, "cli") || strings.Contains(featureHints, "terminal") || strings.Contains(featureHints, "command") {
		domains = append(domains, "cli")
	}
	if strings.Contains(featureHints, "doc") || strings.Contains(featureHints, "content") {
		domains = append(domains, "docs")
	}
	if strings.Contains(featureHints, "data") || strings.Contains(featureHints, "analytics") || strings.Contains(featureHints, "ml") || strings.Contains(featureHints, "ai") {
		domains = append(domains, "data")
	}

	domains = uniqueStrings(domains)
	agents := SynthesizeTeam(hm.project, domains)

	domainTeams := make(map[string]*TeamManager)
	for _, a := range agents {
		tm, ok := domainTeams[a.Domain]
		if !ok {
			tm, _ = hm.CreateTeam(a.Domain, hm.model)
			domainTeams[a.Domain] = tm
		}
		agent := BuildAgentFromPlan(hm.project.Goal, hm.model, models.PlanAgent{
			Role:        a.Role,
			Description: a.Description,
			Tools:       a.Tools,
			Tasks:       a.Tasks,
		}, tm.agent.ID)
		agent.SystemPrompt = GenerateSystemPrompt(a.Role, a.Description, a.Tools, a.Tasks, hm.model)

		taskDefs := make([]TaskDef, len(a.Tasks))
		for i, t := range a.Tasks {
			prio := "medium"
			if i == 0 {
				prio = "high"
			}
			taskDefs[i] = TaskDef{Description: t, Priority: prio}
		}
		tm.AdoptPreBuiltAgent(agent, taskDefs)
		hm.memPolicy.Grant(agent.ID, a.Role, AccessWrite, fmt.Sprintf("Agent %s scope", a.Role))
		hm.sandbox.RegisterAgent(agent.ID)

		pterm.Printfln("  🤖 Created agent: %s (%d tasks)", a.Role, len(a.Tasks))
	}

	hm.logDecision("synthesis", fmt.Sprintf("Synthesized %d agents across %d teams from requirements", len(agents), len(domainTeams)))
	pterm.Success.Printfln("Synthesis complete — %d agents across %d teams", len(agents), len(domainTeams))
	return nil
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var r []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			r = append(r, v)
		}
	}
	return r
}
