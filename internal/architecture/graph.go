package architecture

import (
	"fmt"
	"strings"
)

// NodeType identifies the architectural layer of a graph node.
type NodeType string

const (
	NodeRoute      NodeType = "Route"
	NodeHandler    NodeType = "Handler"
	NodeService    NodeType = "Service"
	NodeRepository NodeType = "Repository"
	NodeDatabase   NodeType = "Database"
)

// Node represents a single architectural component.
type Node struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Type  NodeType `json:"type"`
}

// Edge represents a directional dependency between components.
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// DependencyGraph represents the complete architectural flow of an application.
type DependencyGraph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges []*Edge          `json:"edges"`
}

// NewGraph creates an empty dependency graph.
func NewGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}
}

// AddNode adds a node to the graph if it doesn't already exist.
func (g *DependencyGraph) AddNode(id, label string, nodeType NodeType) *Node {
	if n, exists := g.Nodes[id]; exists {
		return n
	}
	n := &Node{ID: id, Label: label, Type: nodeType}
	g.Nodes[id] = n
	return n
}

// AddEdge adds a directed dependency edge.
func (g *DependencyGraph) AddEdge(from, to, relation string) {
	for _, e := range g.Edges {
		if e.From == from && e.To == to && e.Relation == relation {
			return
		}
	}
	g.Edges = append(g.Edges, &Edge{From: from, To: to, Relation: relation})
}

// ToMermaid exports the graph to Mermaid flowchart syntax.
func (g *DependencyGraph) ToMermaid() string {
	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	// Declare nodes with layer styles
	for _, n := range g.Nodes {
		var shape string
		switch n.Type {
		case NodeRoute:
			shape = fmt.Sprintf("  %s([\"%s\"]):::route\n", sanitizeID(n.ID), n.Label)
		case NodeDatabase:
			shape = fmt.Sprintf("  %s[(\"%s\")]:::db\n", sanitizeID(n.ID), n.Label)
		default:
			shape = fmt.Sprintf("  %s[\"%s\"]:::%s\n", sanitizeID(n.ID), n.Label, strings.ToLower(string(n.Type)))
		}
		sb.WriteString(shape)
	}

	sb.WriteString("\n")

	// Declare edges
	for _, e := range g.Edges {
		if e.Relation != "" {
			sb.WriteString(fmt.Sprintf("  %s -->|\"%s\"| %s\n", sanitizeID(e.From), e.Relation, sanitizeID(e.To)))
		} else {
			sb.WriteString(fmt.Sprintf("  %s --> %s\n", sanitizeID(e.From), sanitizeID(e.To)))
		}
	}

	sb.WriteString("\n")
	sb.WriteString("  classDef route fill:#38bdf8,stroke:#0284c7,color:#000;\n")
	sb.WriteString("  classDef handler fill:#818cf8,stroke:#4f46e5,color:#fff;\n")
	sb.WriteString("  classDef service fill:#34d399,stroke:#059669,color:#000;\n")
	sb.WriteString("  classDef repository fill:#fbbf24,stroke:#d97706,color:#000;\n")
	sb.WriteString("  classDef db fill:#f43f5e,stroke:#e11d48,color:#fff;\n")

	return sb.String()
}

// ToASCII exports a structured ASCII representation of the graph.
func (g *DependencyGraph) ToASCII() string {
	var sb strings.Builder
	sb.WriteString("=== Pogonian Architecture Graph ===\n")
	for _, n := range g.Nodes {
		sb.WriteString(fmt.Sprintf("[%s] %s (%s)\n", n.Type, n.Label, n.ID))
		for _, e := range g.Edges {
			if e.From == n.ID {
				sb.WriteString(fmt.Sprintf("    ↳ %s → %s\n", e.Relation, e.To))
			}
		}
	}
	return sb.String()
}

// DetectCycles checks for circular dependencies in the graph using DFS.
func (g *DependencyGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(u string)
	dfs = func(u string) {
		visited[u] = true
		recStack[u] = true
		path = append(path, u)

		for _, e := range g.Edges {
			if e.From == u {
				v := e.To
				if !visited[v] {
					dfs(v)
				} else if recStack[v] {
					// Cycle found
					cycleStartIdx := -1
					for i, p := range path {
						if p == v {
							cycleStartIdx = i
							break
						}
					}
					if cycleStartIdx != -1 {
						cycle := append([]string{}, path[cycleStartIdx:]...)
						cycle = append(cycle, v)
						cycles = append(cycles, cycle)
					}
				}
			}
		}

		path = path[:len(path)-1]
		recStack[u] = false
	}

	for id := range g.Nodes {
		if !visited[id] {
			dfs(id)
		}
	}

	return cycles
}

func sanitizeID(id string) string {
	r := strings.NewReplacer("/", "_", ":", "_", ".", "_", "-", "_", " ", "_")
	return r.Replace(id)
}
