// Package knowledge provides cross-project knowledge graph functionality.
package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Graph manages cross-project knowledge.
type Graph struct {
	path     string
	entries  []*Entry
	mu       sync.RWMutex
}

// Entry represents a knowledge entry.
type Entry struct {
	ID          string    `json:"id"`
	Project     string    `json:"project"`
	Type        string    `json:"type"`        // pattern, solution, decision
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Code        string    `json:"code,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	UseCount    int       `json:"use_count"`
}

// Query represents a knowledge query.
type Query struct {
	Text    string   `json:"text"`
	Tags    []string `json:"tags,omitempty"`
	Type    string   `json:"type,omitempty"`
	Project string   `json:"project,omitempty"`
	Limit   int      `json:"limit,omitempty"`
}

// Result represents a query result.
type Result struct {
	Entry      *Entry  `json:"entry"`
	Relevance  float64 `json:"relevance"`
	Highlights []string `json:"highlights,omitempty"`
}

// NewGraph creates a new knowledge graph.
func NewGraph(path string) (*Graph, error) {
	g := &Graph{
		path:    path,
		entries: make([]*Entry, 0),
	}

	if err := g.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return g, nil
}

// Add adds a new entry to the graph.
func (g *Graph) Add(entry *Entry) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	entry.CreatedAt = time.Now()
	entry.LastUsed = time.Now()
	entry.UseCount = 0

	g.entries = append(g.entries, entry)
	return g.save()
}

// Query searches the knowledge graph.
func (g *Graph) Query(q Query) []*Result {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var results []*Result

	for _, entry := range g.entries {
		score := g.scoreEntry(entry, q)
		if score > 0 {
			results = append(results, &Result{
				Entry:     entry,
				Relevance: score,
			})
		}
	}

	// Sort by relevance
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Relevance > results[i].Relevance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply limit
	if q.Limit > 0 && len(results) > q.Limit {
		results = results[:q.Limit]
	}

	return results
}

// scoreEntry calculates relevance score for an entry.
func (g *Graph) scoreEntry(entry *Entry, q Query) float64 {
	score := 0.0

	// Text matching
	if q.Text != "" {
		if containsIgnoreCase(entry.Title, q.Text) {
			score += 0.5
		}
		if containsIgnoreCase(entry.Description, q.Text) {
			score += 0.3
		}
		if containsIgnoreCase(entry.Code, q.Text) {
			score += 0.2
		}
	}

	// Tag matching
	if len(q.Tags) > 0 {
		for _, qTag := range q.Tags {
			for _, eTag := range entry.Tags {
				if qTag == eTag {
					score += 0.2
				}
			}
		}
	}

	// Type matching
	if q.Type != "" && entry.Type == q.Type {
		score += 0.3
	}

	// Project matching (lower score for same project to encourage cross-project)
	if q.Project != "" && entry.Project != q.Project {
		score += 0.1
	}

	// Boost frequently used entries
	score += float64(entry.UseCount) * 0.05

	return score
}

// MarkUsed marks an entry as used.
func (g *Graph) MarkUsed(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, entry := range g.entries {
		if entry.ID == id {
			entry.LastUsed = time.Now()
			entry.UseCount++
			return g.save()
		}
	}

	return nil
}

// GetByID retrieves an entry by ID.
func (g *Graph) GetByID(id string) *Entry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, entry := range g.entries {
		if entry.ID == id {
			return entry
		}
	}

	return nil
}

// Transfer transfers knowledge from one project to another context.
func (g *Graph) Transfer(fromProject, toProject string) []*Entry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var transferred []*Entry
	for _, entry := range g.entries {
		if entry.Project == fromProject {
			transferred = append(transferred, entry)
		}
	}

	return transferred
}

// load loads the graph from disk.
func (g *Graph) load() error {
	if g.path == "" {
		return nil
	}

	data, err := os.ReadFile(g.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &g.entries)
}

// save saves the graph to disk.
func (g *Graph) save() error {
	if g.path == "" {
		return nil
	}

	dir := filepath.Dir(g.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(g.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(g.path, data, 0644)
}

// Stats returns graph statistics.
func (g *Graph) Stats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	projects := make(map[string]int)
	types := make(map[string]int)

	for _, entry := range g.entries {
		projects[entry.Project]++
		types[entry.Type]++
	}

	return map[string]interface{}{
		"total_entries": len(g.entries),
		"projects":      len(projects),
		"types":         types,
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 (len(s) > len(substr) && containsLower(lower(s), lower(substr))))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
