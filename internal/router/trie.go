package router

import (
	"strconv"
	"strings"
)

// TrieNode represents a single segment node in the Radix Route Tree.
type TrieNode struct {
	Segment    string
	IsParam    bool
	IsCatchAll bool
	ParamName  string
	ParamType  string // "int", "string"
	Route      *Route
	Children   map[string]*TrieNode
	ParamChild *TrieNode
	CatchAll   *TrieNode
}

// TrieRouter provides high-performance O(k) prefix-tree route matching with typed parameter extraction.
type TrieRouter struct {
	Root *TrieNode
}

// NewTrieRouter creates a new TrieRouter.
func NewTrieRouter() *TrieRouter {
	return &TrieRouter{
		Root: &TrieNode{
			Children: make(map[string]*TrieNode),
		},
	}
}

// Insert adds a route pattern into the Radix Tree.
// Supports:
// - Static segments: "products", "about"
// - Param segments (brackets): "[id]", "[id:int]", "[slug:string]"
// - Param segments (colon): ":id", ":id:int"
// - Catch-all segments: "*rest", "[...slug]"
func (tr *TrieRouter) Insert(pattern string, route *Route) {
	cleanPattern := strings.Trim(pattern, "/")
	if cleanPattern == "" {
		tr.Root.Route = route
		return
	}

	segments := strings.Split(cleanPattern, "/")
	curr := tr.Root

	for _, seg := range segments {
		// Check Catch-all: "*path" or "[...slug]"
		if strings.HasPrefix(seg, "*") || (strings.HasPrefix(seg, "[...") && strings.HasSuffix(seg, "]")) {
			paramName := "path"
			if strings.HasPrefix(seg, "*") {
				paramName = strings.TrimPrefix(seg, "*")
			} else {
				paramName = strings.TrimSuffix(strings.TrimPrefix(seg, "[..."), "]")
			}
			if paramName == "" {
				paramName = "path"
			}

			if curr.CatchAll == nil {
				curr.CatchAll = &TrieNode{
					Segment:    seg,
					IsCatchAll: true,
					ParamName:  paramName,
					Children:   make(map[string]*TrieNode),
				}
			}
			curr = curr.CatchAll
			break
		}

		// Check Param: "[id]" or ":id"
		isParam := false
		paramName := ""
		paramType := "string"

		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			isParam = true
			inner := strings.Trim(seg, "[]")
			paramName = inner
			if strings.Contains(inner, ":") {
				parts := strings.SplitN(inner, ":", 2)
				paramName = parts[0]
				paramType = parts[1]
			}
		} else if strings.HasPrefix(seg, ":") {
			isParam = true
			inner := strings.TrimPrefix(seg, ":")
			paramName = inner
			if strings.Contains(inner, ":") {
				parts := strings.SplitN(inner, ":", 2)
				paramName = parts[0]
				paramType = parts[1]
			}
		}

		if isParam {
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
		valid := true
		if node.ParamChild.ParamType == "int" {
			if _, err := strconv.ParseInt(seg, 10, 64); err != nil {
				valid = false
			}
		}

		if valid {
			params[node.ParamChild.ParamName] = seg
			if r := tr.matchRecursive(node.ParamChild, segments, index+1, params); r != nil {
				return r
			}
			delete(params, node.ParamChild.ParamName)
		}
	}

	// 3. Try catch-all match fallback
	if node.CatchAll != nil {
		remaining := strings.Join(segments[index:], "/")
		params[node.CatchAll.ParamName] = remaining
		if node.CatchAll.Route != nil {
			return node.CatchAll.Route
		}
	}

	return nil
}
