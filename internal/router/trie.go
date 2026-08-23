package router

import (
	"strconv"
	"strings"
)

// TrieNode represents a single segment node in the Radix Route Tree.
type TrieNode struct {
	Segment    string
	IsParam    bool
	ParamName  string
	ParamType  string // "int", "string"
	Route      *Route
	Children   map[string]*TrieNode
	ParamChild *TrieNode
}

// TrieRouter provides high-performance O(k) prefix-tree route matching with typed parameter extraction.
type TrieRouter struct {
	Root *TrieNode
}

func NewTrieRouter() *TrieRouter {
	return &TrieRouter{
		Root: &TrieNode{
			Children: make(map[string]*TrieNode),
		},
	}
}

// Insert adds a route pattern into the Radix Tree.
func (tr *TrieRouter) Insert(pattern string, route *Route) {
	cleanPattern := strings.Trim(pattern, "/")
	if cleanPattern == "" {
		tr.Root.Route = route
		return
	}

	segments := strings.Split(cleanPattern, "/")
	curr := tr.Root

	for _, seg := range segments {
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			// Parameter segment e.g. [id] or [id:int] or [slug:string]
			inner := strings.Trim(seg, "[]")
			paramName := inner
			paramType := "string"

			if strings.Contains(inner, ":") {
				parts := strings.SplitN(inner, ":", 2)
				paramName = parts[0]
				paramType = parts[1]
			}

			if curr.ParamChild == nil {
				curr.ParamChild = &TrieNode{
					Segment:   seg,
					IsParam:   true,
					ParamName: paramName,
					ParamType: paramType,
					Children:  make(map[string]*TrieNode),
				}
			}
			curr = curr.ParamChild
		} else {
			// Static segment
			if _, exists := curr.Children[seg]; !exists {
				curr.Children[seg] = &TrieNode{
					Segment:  seg,
					Children: make(map[string]*TrieNode),
				}
			}
			curr = curr.Children[seg]
		}
	}

	curr.Route = route
}

// Match searches the Trie for the matching route and extracts typed parameters.
func (tr *TrieRouter) Match(path string) (*Route, map[string]string) {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		if tr.Root.Route != nil {
			return tr.Root.Route, make(map[string]string)
		}
		return nil, nil
	}

	segments := strings.Split(cleanPath, "/")
	params := make(map[string]string)

	route := tr.matchRecursive(tr.Root, segments, 0, params)
	if route != nil {
		return route, params
	}
	return nil, nil
}

func (tr *TrieRouter) matchRecursive(node *TrieNode, segments []string, index int, params map[string]string) *Route {
	if index == len(segments) {
		return node.Route
	}

	seg := segments[index]

	// 1. Try exact static match first
	if child, exists := node.Children[seg]; exists {
		if r := tr.matchRecursive(child, segments, index+1, params); r != nil {
			return r
		}
	}

	// 2. Try parameter match fallback
	if node.ParamChild != nil {
		// If param has a type restriction (e.g. int), validate it
		if node.ParamChild.ParamType == "int" {
			if _, err := strconv.ParseInt(seg, 10, 64); err != nil {
				return nil
			}
		}

		params[node.ParamChild.ParamName] = seg
		if r := tr.matchRecursive(node.ParamChild, segments, index+1, params); r != nil {
			return r
		}
		delete(params, node.ParamChild.ParamName)
	}

	return nil
}
