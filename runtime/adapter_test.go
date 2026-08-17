package runtime

import (
	"testing"

	"github.com/plexusone/omnideploy/config"
)

// fakeAdapter is a minimal Adapter for exercising Detect's resolution
// order without depending on the real omniagent/container adapters
// (which would import this package, creating a cycle).
type fakeAdapter struct {
	name    string
	matches bool
}

func (f *fakeAdapter) Name() string        { return f.name }
func (f *fakeAdapter) Description() string { return f.name }
func (f *fakeAdapter) Detect(string) bool  { return f.matches }
func (f *fakeAdapter) Load(string) (*config.DeployConfig, error) {
	return &config.DeployConfig{Name: f.name}, nil
}

// withRegistry runs fn against a temporary, isolated registry (saving and
// restoring the package-level one), so tests don't interfere with each
// other or with real adapters registered by other packages' init()s.
func withRegistry(t *testing.T, adapters ...*fakeAdapter) {
	t.Helper()
	saved := registry
	registry = make(map[string]Adapter)
	t.Cleanup(func() { registry = saved })

	for _, a := range adapters {
		Register(a)
	}
}

func TestDetect_SpecificAdapterWinsOverGeneric(t *testing.T) {
	// Both a specific adapter and the generic "container" adapter match
	// the same file (e.g. a "deploy.yaml" that's also valid omniagent
	// content) — the specific one must win, deterministically, every
	// time, not depend on map iteration order.
	for i := 0; i < 20; i++ {
		withRegistry(t,
			&fakeAdapter{name: genericAdapterName, matches: true},
			&fakeAdapter{name: "omniagent", matches: true},
		)

		got, err := Detect("deploy.yaml")
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if got.Name() != "omniagent" {
			t.Fatalf("Detect() = %q, want \"omniagent\" (specific adapter should win over the generic one)", got.Name())
		}
	}
}

func TestDetect_FallsBackToGenericWhenNothingElseMatches(t *testing.T) {
	withRegistry(t,
		&fakeAdapter{name: genericAdapterName, matches: true},
		&fakeAdapter{name: "omniagent", matches: false},
	)

	got, err := Detect("deploy.yaml")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Name() != genericAdapterName {
		t.Errorf("Detect() = %q, want the generic adapter as fallback", got.Name())
	}
}

func TestDetect_NoMatch(t *testing.T) {
	withRegistry(t, &fakeAdapter{name: "omniagent", matches: false})

	if _, err := Detect("nothing.yaml"); err == nil {
		t.Error("Detect() error = nil, want an error when no adapter matches")
	}
}

func TestDetect_DeterministicAmongMultipleSpecificAdapters(t *testing.T) {
	// Two non-generic adapters both match: resolution must be stable
	// (alphabetical), not random, across repeated calls.
	withRegistry(t,
		&fakeAdapter{name: "zeta", matches: true},
		&fakeAdapter{name: "alpha", matches: true},
	)

	for i := 0; i < 20; i++ {
		got, err := Detect("x.yaml")
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if got.Name() != "alpha" {
			t.Fatalf("Detect() = %q, want \"alpha\" (alphabetically first) on iteration %d", got.Name(), i)
		}
	}
}
