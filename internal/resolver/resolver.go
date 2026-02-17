package resolver

import (
	"fmt"
	"sort"
	"strings"
)

// StackInfo represents a stack's metadata needed for resolution.
type StackInfo struct {
	ID      string
	Depends []string
}

// Resolution is the result of dependency resolution.
type Resolution struct {
	// Order is the topologically sorted list of all stack IDs.
	Order []string
	// Explicit are the stacks directly requested.
	Explicit map[string]bool
	// DependencyOf maps transitive deps to the stack that requires them.
	DependencyOf map[string]string
}

// CircularDependencyError indicates a cycle in the dependency graph.
type CircularDependencyError struct {
	Cycle []string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency: %s", strings.Join(e.Cycle, " → "))
}

// MissingStackError indicates a requested stack doesn't exist.
type MissingStackError struct {
	Stack string
}

func (e *MissingStackError) Error() string {
	return fmt.Sprintf("stack not found: %s", e.Stack)
}

// MissingDependencyError indicates a dependency doesn't exist.
type MissingDependencyError struct {
	Stack      string
	Dependency string
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("stack %q depends on %q, which does not exist", e.Stack, e.Dependency)
}

// Resolver resolves stack dependencies.
type Resolver struct {
	stacks map[string]StackInfo
}

// NewResolver creates a resolver with the given stacks.
func NewResolver(stacks map[string]StackInfo) *Resolver {
	return &Resolver{stacks: stacks}
}

// Resolve resolves dependencies for the given explicit stacks using Kahn's algorithm.
func (r *Resolver) Resolve(explicit []string) (*Resolution, error) {
	// Validate explicit stacks exist
	for _, id := range explicit {
		if _, ok := r.stacks[id]; !ok {
			return nil, &MissingStackError{Stack: id}
		}
	}

	// Collect all needed stacks (explicit + transitive deps)
	needed := make(map[string]bool)
	explicitSet := make(map[string]bool)
	dependencyOf := make(map[string]string)

	for _, id := range explicit {
		explicitSet[id] = true
	}

	// BFS to find all transitive dependencies
	queue := make([]string, len(explicit))
	copy(queue, explicit)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if needed[current] {
			continue
		}
		needed[current] = true

		info, ok := r.stacks[current]
		if !ok {
			return nil, &MissingStackError{Stack: current}
		}

		for _, dep := range info.Depends {
			depInfo, ok := r.stacks[dep]
			if !ok {
				return nil, &MissingDependencyError{Stack: current, Dependency: dep}
			}
			_ = depInfo
			if !explicitSet[dep] && dependencyOf[dep] == "" {
				dependencyOf[dep] = current
			}
			queue = append(queue, dep)
		}
	}

	// Kahn's algorithm for topological sort
	// Build in-degree map restricted to needed stacks
	inDegree := make(map[string]int)
	adj := make(map[string][]string) // dep -> dependents
	for id := range needed {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, dep := range r.stacks[id].Depends {
			if needed[dep] {
				adj[dep] = append(adj[dep], id)
				inDegree[id]++
			}
		}
	}

	// Find all nodes with in-degree 0
	var queue2 []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue2 = append(queue2, id)
		}
	}
	sort.Strings(queue2) // deterministic order

	var order []string
	for len(queue2) > 0 {
		// Sort for deterministic ordering
		sort.Strings(queue2)
		node := queue2[0]
		queue2 = queue2[1:]
		order = append(order, node)

		for _, dependent := range adj[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue2 = append(queue2, dependent)
			}
		}
	}

	// If not all nodes are in the order, there's a cycle
	if len(order) != len(needed) {
		cycle := r.findCycle(needed)
		return nil, &CircularDependencyError{Cycle: cycle}
	}

	return &Resolution{
		Order:        order,
		Explicit:     explicitSet,
		DependencyOf: dependencyOf,
	}, nil
}

// ResolveRemoval determines which stacks become orphans when removing stacks.
func (r *Resolver) ResolveRemoval(currentExplicit []string, removing []string) (orphans []string) {
	removingSet := make(map[string]bool)
	for _, id := range removing {
		removingSet[id] = true
	}

	// Compute remaining explicit
	var remaining []string
	for _, id := range currentExplicit {
		if !removingSet[id] {
			remaining = append(remaining, id)
		}
	}

	// Resolve with remaining stacks
	res, err := r.Resolve(remaining)
	if err != nil {
		return nil
	}

	// Resolve with current stacks
	currentRes, err := r.Resolve(currentExplicit)
	if err != nil {
		return nil
	}

	// Anything in current resolution but not in remaining resolution is an orphan
	newNeeded := make(map[string]bool)
	for _, id := range res.Order {
		newNeeded[id] = true
	}

	for _, id := range currentRes.Order {
		if !newNeeded[id] && !removingSet[id] {
			orphans = append(orphans, id)
		}
	}

	sort.Strings(orphans)
	return orphans
}

// findCycle finds a cycle in the dependency graph among the needed stacks.
func (r *Resolver) findCycle(needed map[string]bool) []string {
	visited := make(map[string]int) // 0=unvisited, 1=in-progress, 2=done
	var path []string

	var dfs func(node string) []string
	dfs = func(node string) []string {
		visited[node] = 1
		path = append(path, node)

		info := r.stacks[node]
		for _, dep := range info.Depends {
			if !needed[dep] {
				continue
			}
			if visited[dep] == 1 {
				// Found cycle: find where dep appears in path
				for i, n := range path {
					if n == dep {
						cycle := make([]string, len(path[i:])+1)
						copy(cycle, path[i:])
						cycle[len(cycle)-1] = dep
						return cycle
					}
				}
			}
			if visited[dep] == 0 {
				if cycle := dfs(dep); cycle != nil {
					return cycle
				}
			}
		}

		path = path[:len(path)-1]
		visited[node] = 2
		return nil
	}

	// Sort for deterministic cycle detection
	ids := make([]string, 0, len(needed))
	for id := range needed {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if visited[id] == 0 {
			if cycle := dfs(id); cycle != nil {
				return cycle
			}
		}
	}
	return nil
}
