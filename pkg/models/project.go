package models

type Project struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Goal            string         `json:"goal"`
	TechStack       string         `json:"tech_stack"`
	Phase           Phase          `json:"phase"`
	LifecyclePhase  LifecyclePhase `json:"lifecycle_phase"`
	Model           string         `json:"model"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
	Config          string         `json:"config,omitempty"`
}
