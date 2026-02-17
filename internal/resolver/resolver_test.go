package resolver

import (
	"errors"
	"testing"
)

func makeStacks(defs map[string][]string) map[string]StackInfo {
	stacks := make(map[string]StackInfo)
	for id, deps := range defs {
		stacks[id] = StackInfo{ID: id, Depends: deps}
	}
	return stacks
}

func TestSimpleChain(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"php":     {},
		"laravel": {"php"},
	})

	r := NewResolver(stacks)
	res, err := r.Resolve([]string{"laravel"})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(res.Order) != 2 {
		t.Fatalf("Order len = %d, want 2", len(res.Order))
	}

	// php must come before laravel
	phpIdx, laravelIdx := -1, -1
	for i, id := range res.Order {
		if id == "php" {
			phpIdx = i
		}
		if id == "laravel" {
			laravelIdx = i
		}
	}
	if phpIdx > laravelIdx {
		t.Errorf("php (idx %d) should come before laravel (idx %d)", phpIdx, laravelIdx)
	}

	if !res.Explicit["laravel"] {
		t.Error("laravel should be explicit")
	}
	if res.Explicit["php"] {
		t.Error("php should not be explicit")
	}
	if res.DependencyOf["php"] != "laravel" {
		t.Errorf("php dependency_of = %q, want %q", res.DependencyOf["php"], "laravel")
	}
}

func TestMultiLevel(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"vue":     {},
		"nuxt":    {"vue"},
		"nuxt-ui": {"nuxt"},
	})

	r := NewResolver(stacks)
	res, err := r.Resolve([]string{"nuxt-ui"})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(res.Order) != 3 {
		t.Fatalf("Order len = %d, want 3", len(res.Order))
	}

	// Check ordering
	indexOf := make(map[string]int)
	for i, id := range res.Order {
		indexOf[id] = i
	}

	if indexOf["vue"] > indexOf["nuxt"] {
		t.Error("vue should come before nuxt")
	}
	if indexOf["nuxt"] > indexOf["nuxt-ui"] {
		t.Error("nuxt should come before nuxt-ui")
	}
}

func TestMultipleExplicit(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"php":     {},
		"laravel": {"php"},
		"vue":     {},
		"nuxt":    {"vue"},
		"nuxt-ui": {"nuxt"},
	})

	r := NewResolver(stacks)
	res, err := r.Resolve([]string{"laravel", "nuxt-ui"})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(res.Order) != 5 {
		t.Fatalf("Order len = %d, want 5", len(res.Order))
	}

	if !res.Explicit["laravel"] || !res.Explicit["nuxt-ui"] {
		t.Error("laravel and nuxt-ui should be explicit")
	}
}

func TestCircularDeps(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	})

	r := NewResolver(stacks)
	_, err := r.Resolve([]string{"a"})
	if err == nil {
		t.Fatal("Resolve() should fail with circular dependency")
	}

	var cycleErr *CircularDependencyError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected CircularDependencyError, got %T: %v", err, err)
	}
}

func TestMissingStack(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"php": {},
	})

	r := NewResolver(stacks)
	_, err := r.Resolve([]string{"laravel"})
	if err == nil {
		t.Fatal("Resolve() should fail for missing stack")
	}

	var missing *MissingStackError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingStackError, got %T: %v", err, err)
	}
	if missing.Stack != "laravel" {
		t.Errorf("Stack = %q, want %q", missing.Stack, "laravel")
	}
}

func TestMissingDependency(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"laravel": {"php"},
	})

	r := NewResolver(stacks)
	_, err := r.Resolve([]string{"laravel"})
	if err == nil {
		t.Fatal("Resolve() should fail for missing dependency")
	}

	var missing *MissingDependencyError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingDependencyError, got %T: %v", err, err)
	}
}

func TestRemoveWithOrphans(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"vue":     {},
		"nuxt":    {"vue"},
		"nuxt-ui": {"nuxt"},
		"php":     {},
		"laravel": {"php"},
	})

	r := NewResolver(stacks)
	orphans := r.ResolveRemoval([]string{"laravel", "nuxt-ui"}, []string{"nuxt-ui"})

	// Removing nuxt-ui should orphan nuxt and vue
	if len(orphans) != 2 {
		t.Fatalf("orphans len = %d, want 2: %v", len(orphans), orphans)
	}

	orphanSet := make(map[string]bool)
	for _, o := range orphans {
		orphanSet[o] = true
	}
	if !orphanSet["nuxt"] {
		t.Error("nuxt should be orphaned")
	}
	if !orphanSet["vue"] {
		t.Error("vue should be orphaned")
	}
}

func TestRemoveNoOrphans(t *testing.T) {
	stacks := makeStacks(map[string][]string{
		"php":     {},
		"laravel": {"php"},
		"symfony": {"php"},
	})

	r := NewResolver(stacks)
	orphans := r.ResolveRemoval([]string{"laravel", "symfony"}, []string{"laravel"})

	// php is still needed by symfony
	if len(orphans) != 0 {
		t.Fatalf("orphans len = %d, want 0: %v", len(orphans), orphans)
	}
}
