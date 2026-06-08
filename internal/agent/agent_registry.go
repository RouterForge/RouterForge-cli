package agent

import (
	"fmt"

	"github.com/routerforge/cli/pkg/models"
)

type AgentTemplate struct {
	Role            string
	Type            models.AgentType
	SystemPrompt    string
	SuccessCriteria []string
	Tools           []string
	MemoryScope     []string
	ReportingFormat string
}

var AgentTemplates = map[string]AgentTemplate{
	"requirement_analyst": {
		Role:            "requirement_analyst",
		Type:            models.TypeMicroAgent,
		SystemPrompt:    "You are a requirement analyst. You analyze project requirements and break them down into detailed specifications.",
		SuccessCriteria: []string{"All requirements documented", "Acceptance criteria defined"},
		Tools:           []string{"read", "write", "search"},
		MemoryScope:     []string{"project"},
		ReportingFormat: "markdown",
	},
	"api_designer": {
		Role:            "api_designer",
		Type:            models.TypeMicroAgent,
		SystemPrompt:    "You are an API designer. You design RESTful APIs with endpoints, schemas, and documentation.",
		SuccessCriteria: []string{"All endpoints defined", "Request/response schemas specified"},
		Tools:           []string{"read", "write"},
		MemoryScope:     []string{"project", "architecture"},
		ReportingFormat: "markdown",
	},
	"component_builder": {
		Role:            "component_builder",
		Type:            models.TypeMicroAgent,
		SystemPrompt:    "You are a frontend component builder. You build UI components following design specs.",
		SuccessCriteria: []string{"All components built", "Components are responsive"},
		Tools:           []string{"read", "write", "search"},
		MemoryScope:     []string{"project", "frontend"},
		ReportingFormat: "markdown",
	},
	"db_schema_designer": {
		Role:            "db_schema_designer",
		Type:            models.TypeMicroAgent,
		SystemPrompt:    "You are a database schema designer. You design data models and migrations.",
		SuccessCriteria: []string{"All tables defined", "Relationships specified", "Indexes defined"},
		Tools:           []string{"read", "write"},
		MemoryScope:     []string{"project", "architecture"},
		ReportingFormat: "markdown",
	},
	"security_reviewer": {
		Role:            "security_reviewer",
		Type:            models.TypeMicroAgent,
		SystemPrompt:    "You are a security reviewer. You audit code for vulnerabilities and recommend fixes.",
		SuccessCriteria: []string{"No critical vulnerabilities", "OWASP top 10 checked"},
		Tools:           []string{"read", "search"},
		MemoryScope:     []string{"project"},
		ReportingFormat: "markdown",
	},
	"docs_writer": {
		Role:            "docs_writer",
		Type:            models.TypeMicroAgent,
		SystemPrompt:    "You are a documentation writer. You produce clear, comprehensive documentation.",
		SuccessCriteria: []string{"All features documented", "README updated"},
		Tools:           []string{"read", "write"},
		MemoryScope:     []string{"project"},
		ReportingFormat: "markdown",
	},
}

func GetTemplate(role string) (AgentTemplate, error) {
	t, ok := AgentTemplates[role]
	if !ok {
		return AgentTemplate{}, fmt.Errorf("unknown agent role: %s", role)
	}
	return t, nil
}

func ListRoles() []string {
	var roles []string
	for k := range AgentTemplates {
		roles = append(roles, k)
	}
	return roles
}
