package models

import "encoding/json"

// Location provides source code traceability for every extracted entity.
type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Codebase is the top-level semantic model of an analyzed codebase.
// It is language-agnostic: adding a new language means writing a parser
// that populates this same structure.
type Codebase struct {
	Root            string         `json:"root"`
	Language        string         `json:"language"`
	Packages        []*Package     `json:"packages"`
	DependencyGraph *DepGraph      `json:"dependency_graph,omitempty"`
	CallGraph       *CallGraph     `json:"call_graph,omitempty"`
	Architecture    *ArchProfile   `json:"architecture,omitempty"`
	Capabilities    []*Capability  `json:"capabilities,omitempty"`
}

// Package represents a language package or module.
type Package struct {
	Name       string       `json:"name"`
	Path       string       `json:"path"`
	Files      []string     `json:"files"`
	Imports    []*Import    `json:"imports"`
	Types      []*Type      `json:"types"`
	Functions  []*Function  `json:"functions"`
	Interfaces []*Interface `json:"interfaces"`
	Location   Location     `json:"location"`
}

// Import represents an imported dependency with optional alias.
type Import struct {
	Path     string   `json:"path"`
	Alias    string   `json:"alias,omitempty"`
	Location Location `json:"location"`
}

// Type represents a named type declaration (struct, interface, enum, alias).
type Type struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Package  string   `json:"package"`
	Fields   []*Field `json:"fields,omitempty"`
	Methods  []string `json:"methods,omitempty"`
	Exported bool     `json:"exported"`
	Location Location `json:"location"`
}

// Field represents a struct field with type, tag, and position.
type Field struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Tag      string   `json:"tag,omitempty"`
	Exported bool     `json:"exported"`
	Location Location `json:"location"`
}

// Function represents a function or method declaration.
type Function struct {
	Name     string      `json:"name"`
	Receiver string      `json:"receiver,omitempty"`
	Package  string      `json:"package"`
	Params   []string    `json:"params,omitempty"`
	Results  []string    `json:"results,omitempty"`
	Calls    []*CallSite `json:"calls,omitempty"`
	Exported bool        `json:"exported"`
	Location Location    `json:"location"`
}

// Interface represents a named interface type with its methods and known implementations.
type Interface struct {
	Name         string   `json:"name"`
	Package      string   `json:"package"`
	Methods      []string `json:"methods"`
	Implementors []string `json:"implementors,omitempty"`
	Location     Location `json:"location"`
}

// CallSite represents one function call at a specific location.
type CallSite struct {
	Caller   string   `json:"caller"`
	Callee   string   `json:"callee"`
	CallExpr string   `json:"call_expr"`
	Location Location `json:"location"`
}

// DepEdge is a weighted dependency edge between two packages.
type DepEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

// DepGraph is the package-level dependency graph.
type DepGraph struct {
	Nodes []string   `json:"nodes"`
	Edges []*DepEdge `json:"edges"`
}

// CallGraph is the function-level call graph.
type CallGraph struct {
	Nodes []string    `json:"nodes"`
	Edges []*CallSite `json:"edges"`
}

// ArchProfile describes the detected software architecture, derived from code relationships.
type ArchProfile struct {
	Architecture string          `json:"architecture"`
	Confidence   float64         `json:"confidence"`
	Layers       []string        `json:"layers,omitempty"`
	Patterns     []string        `json:"patterns,omitempty"`
	Evidence     []*ArchEvidence `json:"evidence,omitempty"`
}

// ArchEvidence is a single piece of evidence for or against an architectural pattern.
type ArchEvidence struct {
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Source      string   `json:"source,omitempty"`
	Target      string   `json:"target,omitempty"`
	Location    Location `json:"location,omitempty"`
	Positive    bool     `json:"positive"`
}

// Capability is a high-level feature or capability detected in the codebase,
// always traceable back to specific source locations.
type Capability struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Confidence  float64    `json:"confidence"`
	Category    string     `json:"category"`
	Sources     []Location `json:"sources"`
	Evidence    []string   `json:"evidence"`
}

func (c *Codebase) JSON() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return string(b)
}

func (c *Codebase) HasPackage(name string) bool {
	for _, p := range c.Packages {
		if p.Name == name {
			return true
		}
	}
	return false
}

func (c *Codebase) PackageByPath(path string) *Package {
	for _, p := range c.Packages {
		if p.Path == path {
			return p
		}
	}
	return nil
}

func (c *Codebase) PackageByName(name string) *Package {
	for _, p := range c.Packages {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (c *Codebase) Interfaces() []*Interface {
	var result []*Interface
	for _, pkg := range c.Packages {
		result = append(result, pkg.Interfaces...)
	}
	return result
}
