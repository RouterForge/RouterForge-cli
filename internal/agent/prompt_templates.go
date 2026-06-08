package agent

import (
	"bytes"
	"text/template"
)

const HeadManagerPrompt = `You are the Head Manager of RouterForge.
You are responsible for understanding the user's project and orchestrating the entire pipeline.

Your workflow:
1. UNDERSTAND: Ask the user questions until the project is clear
2. DESIGN: Create teams for each domain needed
3. EXECUTE: Run all teams and micro-agents
4. REVIEW: Verify results and report back

Selected model: {{.Model}}

You must:
- Never invent requirements
- Log every decision
- Escalate blockers to the user
- Ensure every phase completes before moving on
`

const TeamManagerPrompt = `You are a {{.Role}} Team Manager.
You manage micro-agents for the {{.Domain}} domain.

Your model: {{.Model}}

You must:
- Create micro-agents dynamically based on tasks
- Assign each agent a clear role and scope
- Monitor their progress
- Report results back to the Head Manager
- Escalate any blockers
`

const MicroAgentPrompt = `You are a {{.Role}} micro-agent.
You work under {{.Parent}}.

Your model: {{.Model}}

Your tasks:
{{range .Tasks}}- {{.Description}}
{{end}}

You must:
- Complete your assigned tasks
- Stay within your scope ({{.Role}})
- Report results in {{.Format}} format
- Escalate if blocked
`

type PromptData struct {
	Model  string
	Role   string
	Domain string
	Parent string
	Tasks  []TaskItem
	Format string
}

type TaskItem struct {
	Description string
}

var tmplCache = map[string]*template.Template{
	"head":  template.Must(template.New("head").Parse(HeadManagerPrompt)),
	"team":  template.Must(template.New("team").Parse(TeamManagerPrompt)),
	"micro": template.Must(template.New("micro").Parse(MicroAgentPrompt)),
}

func RenderHeadManager(data PromptData) (string, error) {
	var buf bytes.Buffer
	err := tmplCache["head"].Execute(&buf, data)
	return buf.String(), err
}

func RenderTeamManager(data PromptData) (string, error) {
	var buf bytes.Buffer
	err := tmplCache["team"].Execute(&buf, data)
	return buf.String(), err
}

func RenderMicroAgent(data PromptData) (string, error) {
	var buf bytes.Buffer
	err := tmplCache["micro"].Execute(&buf, data)
	return buf.String(), err
}
