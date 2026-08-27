package profile

import (
	"reflect"
	"testing"
)

// kvMap turns a profile's variables into a map, for assertions that care about
// what a profile holds rather than in what order.
func kvMap(p Profile) map[string]string {
	m := make(map[string]string, len(p.KVs))
	for _, kv := range p.KVs {
		m[kv.Key] = kv.Value
	}

	return m
}

// TestComposeProfilePrecedence pins the answer to the precedence question:
// the profile applied last wins, and everything the earlier profile holds
// that the later one does not is kept.
func TestComposeProfilePrecedence(t *testing.T) {
	a := Profile{
		Name: "A",
		KVs: []KV{
			{Key: "profile_name", Value: "A"},
			{Key: "ONLY_IN_A", Value: "a"},
			{Key: "SHARED", Value: "from-a"},
		},
	}
	b := Profile{
		Name: "B",
		KVs: []KV{
			{Key: "profile_name", Value: "B"},
			{Key: "ONLY_IN_B", Value: "b"},
			{Key: "SHARED", Value: "from-b"},
		},
	}

	got := kvMap(ComposeProfile([]Profile{a, b}))
	want := map[string]string{
		"profile_name": "A+B",
		"ONLY_IN_A":    "a",
		"ONLY_IN_B":    "b",
		"SHARED":       "from-b",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ComposeProfile([A B]) = %v, want %v", got, want)
	}
}

func TestComposeProfileName(t *testing.T) {
	named := func(name string) Profile {
		return Profile{Name: name, KVs: []KV{{Key: "profile_name", Value: name}}}
	}

	tests := []struct {
		name     string
		profiles []Profile
		want     string
	}{
		{
			name:     "a single profile keeps its name",
			profiles: []Profile{named("A")},
			want:     "A",
		},
		{
			name:     "a stack is named after every profile in it",
			profiles: []Profile{named("A"), named("B")},
			want:     "A+B",
		},
		{
			// Composing is idempotent: `profiler use A` inside A+B stays A+B
			// rather than growing a name every time.
			name:     "a profile already in the stack does not repeat",
			profiles: []Profile{named("A+B"), named("A")},
			want:     "A+B",
		},
		{
			name:     "stacking onto a stack appends",
			profiles: []Profile{named("A+B"), named("C")},
			want:     "A+B+C",
		},
		{
			// The env files carry no profile_name of their own, so they must
			// not put No-Name-Profile into the name of the stack they join.
			name:     "an unnamed profile does not name the stack",
			profiles: []Profile{named("A"), named(NoName)},
			want:     "A",
		},
		{
			name:     "a stack of unnamed profiles is still unnamed",
			profiles: []Profile{named(NoName)},
			want:     NoName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			composed := ComposeProfile(tc.profiles)

			if composed.Name != tc.want {
				t.Errorf("Name = %q, want %q", composed.Name, tc.want)
			}

			// The name and the profile_name variable are the same fact, and
			// `profiler status` reads the variable:
			if got := kvMap(composed)["profile_name"]; got != tc.want {
				t.Errorf("profile_name = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestComposeProfileNameFromVariable covers Consul, whose Profile.Name is the
// full KV path while its profile_name variable is the profile's own name.
func TestComposeProfileNameFromVariable(t *testing.T) {
	consulProfile := Profile{
		Name: "profiler/staging",
		KVs:  []KV{{Key: "profile_name", Value: "staging"}},
	}

	if got := ComposeProfile([]Profile{consulProfile}).Name; got != "staging" {
		t.Errorf("Name = %q, want %q", got, "staging")
	}
}

// TestComposeProfileSortsKeys pins the variable order, because the composed
// profile is written to the profiler file in it.
func TestComposeProfileSortsKeys(t *testing.T) {
	p := Profile{
		Name: "A",
		KVs: []KV{
			{Key: "ZULU", Value: "z"},
			{Key: "ALPHA", Value: "a"},
			{Key: "MIKE", Value: "m"},
		},
	}

	var keys []string
	for _, kv := range ComposeProfile([]Profile{p}).KVs {
		keys = append(keys, kv.Key)
	}

	want := []string{"ALPHA", "MIKE", "ZULU", "profile_name"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %v, want %v", keys, want)
	}
}

// TestComposeProfileDropsStaleKeysMarker checks that profile_keys read back
// from a profiler file does not survive into the composed profile: it is
// bookkeeping, rewritten by SetEnvironment from the composed key set.
func TestComposeProfileDropsStaleKeysMarker(t *testing.T) {
	p := Profile{
		Name: "A",
		KVs: []KV{
			{Key: KeysVar, Value: "STALE,KEYS"},
			{Key: "FOO", Value: "bar"},
		},
	}

	if _, found := kvMap(ComposeProfile([]Profile{p}))[KeysVar]; found {
		t.Errorf("ComposeProfile kept %s", KeysVar)
	}
}

func TestActiveProfile(t *testing.T) {
	t.Run("no profile in use", func(t *testing.T) {
		t.Setenv(KeysVar, "")

		if _, ok := ActiveProfile(); ok {
			t.Error("ActiveProfile() reported a profile with no profile_keys set")
		}
	})

	t.Run("rebuilds the profile from the environment", func(t *testing.T) {
		t.Setenv(NameVar, "A")
		t.Setenv("FOO", "bar")
		t.Setenv("BAZ", "qux")
		t.Setenv(KeysVar, "profile_name,FOO,BAZ")

		active, ok := ActiveProfile()
		if !ok {
			t.Fatal("ActiveProfile() = _, false, want true")
		}

		if active.Name != "A" {
			t.Errorf("Name = %q, want %q", active.Name, "A")
		}

		want := map[string]string{"profile_name": "A", "FOO": "bar", "BAZ": "qux"}
		if got := kvMap(active); !reflect.DeepEqual(got, want) {
			t.Errorf("KVs = %v, want %v", got, want)
		}
	})

	t.Run("a key unset since drops out of the profile", func(t *testing.T) {
		t.Setenv(NameVar, "A")
		t.Setenv("FOO", "bar")
		t.Setenv(KeysVar, "FOO,GONE")

		active, _ := ActiveProfile()

		if _, found := kvMap(active)["GONE"]; found {
			t.Error("ActiveProfile() rebuilt a variable that is no longer in the environment")
		}
	})
}

// TestWithKeysMarker pins what SetEnvironment exports beside the profile's own
// variables: the marker names every key, and replaces a stale one rather than
// being written twice.
func TestWithKeysMarker(t *testing.T) {
	kvs := withKeysMarker([]KV{
		{Key: KeysVar, Value: "STALE"},
		{Key: "FOO", Value: "bar"},
		{Key: "profile_name", Value: "A"},
	})

	want := []KV{
		{Key: "FOO", Value: "bar"},
		{Key: "profile_name", Value: "A"},
		{Key: KeysVar, Value: "FOO,profile_name"},
	}

	if !reflect.DeepEqual(kvs, want) {
		t.Errorf("withKeysMarker() = %v, want %v", kvs, want)
	}
}
