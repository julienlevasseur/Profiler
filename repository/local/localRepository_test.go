package local_test

import (
	"slices"
	"testing"

	"github.com/spf13/viper"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/repository"
	"github.com/julienlevasseur/profiler/repository/local"
)

// NewLocalRepository has to satisfy the interface every command talks to. This
// is the assertion the redesign is really about: cmd/ never sees a
// *localRepository, only an IRepository, so the suite below drives the local
// backend the same way -- through the interface, so it is a template for the
// other providers rather than a test of one struct's methods.
//
// It lives in an external test package on purpose: the repository package
// imports repository/local (its factory does), so a same-package test that
// imported repository back would be an import cycle.
var _ repository.IRepository = local.NewLocalRepository()

// newRepo points profilesFolder at a fresh directory and moves the test into
// an empty working directory, then returns the local backend as the interface
// the rest of profiler holds it by. The empty cwd matters: Get merges any
// .env / .env.yml / .envrc found there, so a stray file would leak into the
// assertions below.
func newRepo(t *testing.T) repository.IRepository {
	t.Helper()

	viper.Set("profilesFolder", t.TempDir())
	t.Cleanup(func() { viper.Set("profilesFolder", "") })

	t.Chdir(t.TempDir())

	return local.NewLocalRepository()
}

// getEnv flattens a profile's KVs into a map so a test can assert on the value
// a key ended up with rather than on slice ordering.
func getEnv(t *testing.T, repo repository.IRepository, name string) map[string]string {
	t.Helper()

	p, err := repo.Get(name)
	if err != nil {
		t.Fatalf("Get(%q): %v", name, err)
	}

	env := make(map[string]string, len(p.KVs))
	for _, kv := range p.KVs {
		env[kv.Key] = kv.Value
	}

	return env
}

// TestLocalRepositoryIdentity covers the two methods cmd/list calls before it
// asks a repository for anything: the name it is listed under, and whether it
// is usable at all. Local is always both.
func TestLocalRepositoryIdentity(t *testing.T) {
	repo := newRepo(t)

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"GetName", repo.GetName(), "local"},
		{"IsConfigured", repo.IsConfigured(), true},
	}

	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s() = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// TestLocalRepositoryLifecycle exercises the whole write/read surface of
// IRepository as one sequence against a single repository, because that is how
// it is used: Add then List then Show then Get then Save then Remove. Each step
// asserts on the state the previous ones left, so a method that silently does
// nothing -- the failure mode the stubbed providers had -- surfaces here.
func TestLocalRepositoryLifecycle(t *testing.T) {
	repo := newRepo(t)

	steps := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Add creates a profile and appends a variable", func(t *testing.T) {
			if err := repo.Add([]string{"demo"}); err != nil {
				t.Fatalf("Add([demo]): %v", err)
			}
			if err := repo.Add([]string{"demo", "FOO", "bar"}); err != nil {
				t.Fatalf("Add([demo FOO bar]): %v", err)
			}
		}},

		{"List names the profile exactly once", func(t *testing.T) {
			got, err := repo.List()
			if err != nil {
				t.Fatalf("List(): %v", err)
			}
			count := 0
			for _, p := range got {
				if p == "demo" {
					count++
				}
			}
			if count != 1 {
				t.Errorf("List() = %v, want demo present exactly once", got)
			}
		}},

		{"Show returns the profile's keys, sorted", func(t *testing.T) {
			got, err := repo.Show("demo")
			if err != nil {
				t.Fatalf("Show(demo): %v", err)
			}
			want := []string{"FOO", "profile_name"}
			if !slices.Equal(got, want) {
				t.Errorf("Show(demo) = %v, want %v", got, want)
			}
		}},

		{"Get returns the profile with its values", func(t *testing.T) {
			env := getEnv(t, repo, "demo")
			if env["profile_name"] != "demo" || env["FOO"] != "bar" {
				t.Errorf("Get(demo) env = %v, want profile_name=demo FOO=bar", env)
			}
		}},

		{"Save replaces the profile wholesale", func(t *testing.T) {
			err := repo.Save(profile.Profile{
				Name: "demo",
				KVs: []profile.KV{
					{Key: "profile_name", Value: "demo"},
					{Key: "FOO", Value: "bar"},
					{Key: "BAZ", Value: "qux"},
				},
			})
			if err != nil {
				t.Fatalf("Save(demo): %v", err)
			}
			if env := getEnv(t, repo, "demo"); env["BAZ"] != "qux" {
				t.Errorf("after Save, BAZ = %q, want qux", env["BAZ"])
			}
		}},

		{"Remove of a key drops just that variable", func(t *testing.T) {
			if err := repo.Remove([]string{"demo", "FOO"}); err != nil {
				t.Fatalf("Remove([demo FOO]): %v", err)
			}
			env := getEnv(t, repo, "demo")
			if _, ok := env["FOO"]; ok {
				t.Errorf("FOO still present after Remove([demo FOO]): %v", env)
			}
			if env["BAZ"] != "qux" {
				t.Errorf("Remove([demo FOO]) took BAZ too: %v", env)
			}
		}},

		{"Remove of a profile deletes it", func(t *testing.T) {
			if err := repo.Remove([]string{"demo"}); err != nil {
				t.Fatalf("Remove([demo]): %v", err)
			}
			got, err := repo.List()
			if err != nil {
				t.Fatalf("List(): %v", err)
			}
			if slices.Contains(got, "demo") {
				t.Errorf("demo still listed after Remove([demo]): %v", got)
			}
		}},
	}

	for _, s := range steps {
		if !t.Run(s.name, s.run) {
			// The steps build on each other, so once one fails the rest are
			// asserting against a state that never happened.
			t.FailNow()
		}
	}
}
