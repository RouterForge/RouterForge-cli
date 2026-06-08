package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceAnalyzer_DetectLanguage_Go(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	sa := &SourceAnalyzer{}
	lang := sa.DetectLanguage(dir)
	if lang != "go" {
		t.Fatalf("expected 'go', got '%s'", lang)
	}
}

func TestSourceAnalyzer_DetectLanguage_Python(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hi')"), 0644)

	sa := &SourceAnalyzer{}
	lang := sa.DetectLanguage(dir)
	if lang != "python" {
		t.Fatalf("expected 'python', got '%s'", lang)
	}
}

func TestSourceAnalyzer_DetectLanguage_Unknown(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.txt"), []byte("hello"), 0644)

	sa := &SourceAnalyzer{}
	lang := sa.DetectLanguage(dir)
	if lang != "unknown" {
		t.Fatalf("expected 'unknown', got '%s'", lang)
	}
}

func TestPatternDetector_GoModule(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	pd := &PatternDetector{}
	patterns := pd.Detect(dir)
	found := false
	for _, p := range patterns {
		if p.Name == "go_module" {
			found = true
			if p.Confidence != 1.0 {
				t.Fatalf("expected confidence 1.0, got %f", p.Confidence)
			}
		}
	}
	if !found {
		t.Fatal("expected go_module pattern")
	}
}

func TestPatternDetector_NpmPackage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	pd := &PatternDetector{}
	patterns := pd.Detect(dir)
	found := false
	for _, p := range patterns {
		if p.Name == "npm_package" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected npm_package pattern")
	}
}

func TestCapabilityGraph_Add(t *testing.T) {
	g := NewCapabilityGraph()
	g.Add("test", "test capability", nil)
	if _, ok := g.Nodes["test"]; !ok {
		t.Fatal("expected test node")
	}
}

func TestRecommender_Simple(t *testing.T) {
	g := NewCapabilityGraph()
	g.Add("a", "base", nil)
	g.Add("b", "depends on a", []string{"a"})

	r := NewRecommender(g)
	order := r.Recommend([]string{"b"})
	if len(order) != 2 {
		t.Fatalf("expected 2 items, got %d: %v", len(order), order)
	}
	if order[0] != "a" {
		t.Fatalf("expected 'a' first, got '%s'", order[0])
	}
	if order[1] != "b" {
		t.Fatalf("expected 'b' second, got '%s'", order[1])
	}
}

func TestRecommender_Dedup(t *testing.T) {
	g := NewCapabilityGraph()
	g.Add("a", "base", nil)
	g.Add("b", "depends on a", []string{"a"})
	g.Add("c", "depends on a too", []string{"a"})

	r := NewRecommender(g)
	order := r.Recommend([]string{"b", "c"})
	if len(order) != 3 {
		t.Fatalf("expected 3 items (no dups), got %d: %v", len(order), order)
	}
}

func TestFeatureMatrix(t *testing.T) {
	fm := NewFeatureMatrix()
	fm.AddRepo("repo1")
	fm.AddRepo("repo2")
	fm.SetFeature("repo1", "has_go", true)
	fm.SetFeature("repo1", "has_docker", true)
	fm.SetFeature("repo2", "has_go", true)

	if len(fm.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(fm.Repos))
	}
	if len(fm.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(fm.Features))
	}
	if !fm.Data["repo1"]["has_docker"] {
		t.Fatal("expected repo1 to have has_docker")
	}
	if fm.Data["repo2"]["has_docker"] {
		t.Fatal("expected repo2 to not have has_docker")
	}
}

func TestFeatureMatrix_Markdown(t *testing.T) {
	fm := NewFeatureMatrix()
	fm.AddRepo("r1")
	fm.SetFeature("r1", "f1", true)

	md := fm.Markdown()
	if md == "" {
		t.Fatal("expected non-empty markdown")
	}
}

func TestManager_New(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}
