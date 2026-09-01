package cmd

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/repository"
)

// TestAppendRepoToProfileName pins the showRepoSourceInList gate. The option
// was defined and defaulted to false, but `list` used to suffix every
// non-local profile unconditionally.
func TestAppendRepoToProfileName(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		showRepoSource bool
		profiles       []string
		want           []string
	}{
		{
			name:           "option off leaves remote profiles alone",
			repo:           "consul",
			showRepoSource: false,
			profiles:       []string{"staging", "prod"},
			want:           []string{"staging", "prod"},
		},
		{
			name:           "option on suffixes remote profiles",
			repo:           "consul",
			showRepoSource: true,
			profiles:       []string{"staging", "prod"},
			want:           []string{"staging (consul)", "prod (consul)"},
		},
		{
			name:           "local profiles are never suffixed",
			repo:           "local",
			showRepoSource: true,
			profiles:       []string{"demo"},
			want:           []string{"demo"},
		},
		{
			name:           "no profiles, nothing to suffix",
			repo:           "vault",
			showRepoSource: true,
			profiles:       nil,
			want:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendRepoToProfileName(
				tt.repo,
				tt.showRepoSource,
				tt.profiles,
			)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFilterProfiles pins the localProfiles / consulProfiles / ssmProfiles
// allow-lists. They were declared by the redesign and never read, so the
// no-limit case matters most: an unset list must keep listing everything.
func TestFilterProfiles(t *testing.T) {
	tests := []struct {
		name     string
		allowed  []string
		profiles []string
		want     []string
	}{
		{
			name:     "unset allow-list lists everything",
			allowed:  nil,
			profiles: []string{"staging", "prod", "team-b-dev"},
			want:     []string{"staging", "prod", "team-b-dev"},
		},
		{
			name:     "empty allow-list lists everything",
			allowed:  []string{},
			profiles: []string{"staging", "prod"},
			want:     []string{"staging", "prod"},
		},
		{
			name:     "allow-list narrows to its names",
			allowed:  []string{"prod", "staging"},
			profiles: []string{"staging", "prod", "team-b-dev"},
			want:     []string{"staging", "prod"},
		},
		{
			name:     "repository order wins over allow-list order",
			allowed:  []string{"prod", "staging"},
			profiles: []string{"staging", "prod"},
			want:     []string{"staging", "prod"},
		},
		{
			name:     "a name the repository does not hold is dropped quietly",
			allowed:  []string{"prod", "typo"},
			profiles: []string{"staging", "prod"},
			want:     []string{"prod"},
		},
		{
			name:     "nothing matches",
			allowed:  []string{"typo"},
			profiles: []string{"staging", "prod"},
			want:     []string{},
		},
		{
			name:     "no profiles to narrow",
			allowed:  []string{"prod"},
			profiles: nil,
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProfiles(tt.allowed, tt.profiles)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestProfileAllowListPerRepository checks that each repository reads its own
// option, and that vault -- which has none -- is never narrowed.
func TestProfileAllowListPerRepository(t *testing.T) {
	cfg := config.Config{
		LocalProfiles:  []string{"local-only"},
		ConsulProfiles: []string{"consul-only"},
		SSMProfiles:    []string{"ssm-only"},
	}

	for repo, want := range map[string][]string{
		"local":   {"local-only"},
		"consul":  {"consul-only"},
		"ssm":     {"ssm-only"},
		"vault":   nil,
		"unknown": nil,
	} {
		if got := cfg.ProfileAllowList(repo); !reflect.DeepEqual(got, want) {
			t.Errorf("ProfileAllowList(%q) = %v, want %v", repo, got, want)
		}
	}
}

// slowRepository is a repository that takes its time answering, which is what
// every backend but local does: each one costs at least a network round trip.
// It records when it was asked, so a test can tell a parallel listing from a
// sequential one.
type slowRepository struct {
	name       string
	delay      time.Duration
	configured bool
	profiles   []string
	err        error

	calls *callLog
}

// callLog records when each repository was asked. Every fake shares one, so
// the lock lives here rather than on the repository: a mutex per fake would
// leave the appends racing each other.
type callLog struct {
	mu sync.Mutex
	at []time.Time
}

func (l *callLog) record() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.at = append(l.at, time.Now())
}

func (l *callLog) times() []time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.at)
}

func (r *slowRepository) IsConfigured() bool { return r.configured }
func (r *slowRepository) GetName() string    { return r.name }

func (r *slowRepository) List() ([]string, error) {
	r.calls.record()

	time.Sleep(r.delay)

	if r.err != nil {
		return nil, r.err
	}

	return r.profiles, nil
}

func (r *slowRepository) Add([]string) error                  { return nil }
func (r *slowRepository) Get(string) (profile.Profile, error) { return profile.Profile{}, nil }
func (r *slowRepository) Remove([]string) error               { return nil }
func (r *slowRepository) Save(profile.Profile) error          { return nil }
func (r *slowRepository) Show(string) ([]string, error)       { return nil, nil }

// useRepositories points the listing at fakes for the duration of one test.
func useRepositories(t *testing.T, repos map[string]repository.IRepository) {
	t.Helper()

	getRepository = func(name string) (repository.IRepository, error) {
		r, ok := repos[name]
		if !ok {
			return nil, errors.New("no matching Repository type")
		}

		return r, nil
	}

	t.Cleanup(func() { getRepository = repository.GetRepository })
}

// TestListRepositoriesIsParallel is the fix itself: `profiler list` used to
// wait for the sum of every backend -- and print nothing at all until the
// slowest had answered, the instant local profiles included. A backend that
// reaches out over the network costs 200ms or more, so the sum was what the
// delay was made of.
func TestListRepositoriesIsParallel(t *testing.T) {
	const delay = 100 * time.Millisecond

	calls := &callLog{}
	repos := map[string]repository.IRepository{}
	names := []string{"local", "consul", "ssm", "vault"}

	for _, name := range names {
		repos[name] = &slowRepository{
			name:       name,
			delay:      delay,
			configured: true,
			profiles:   []string{name + "-profile"},
			calls:      calls,
		}
	}

	useRepositories(t, repos)

	start := time.Now()
	if _, err := listRepositories(names, config.Config{}); err != nil {
		t.Fatalf("listRepositories(): %v", err)
	}
	elapsed := time.Since(start)

	// Sequentially this is 4 x delay. Parallel it is one, and the margin is
	// wide enough that a loaded machine does not fail the test.
	if elapsed >= 2*delay {
		t.Errorf(
			"listRepositories took %v for %d repositories of %v each; expected roughly one delay, not their sum",
			elapsed, len(names), delay,
		)
	}

	asked := calls.times()
	if len(asked) != len(names) {
		t.Fatalf("got %d List calls, want %d", len(asked), len(names))
	}

	// Every repository was asked before the first one had answered:
	for i, at := range asked {
		if spread := at.Sub(asked[0]); spread >= delay {
			t.Errorf("repository %d was asked %v after the first, so the listing is still sequential", i, spread)
		}
	}
}

// TestListRepositoriesKeepsConfiguredOrder pins the output order to
// supportedRepositories, not to which backend answered first. Ordering by
// arrival would shuffle `profiler list` from run to run.
func TestListRepositoriesKeepsConfiguredOrder(t *testing.T) {
	calls := &callLog{}

	// Deliberately inverted: the repository listed first is the slowest.
	useRepositories(t, map[string]repository.IRepository{
		"local":  &slowRepository{name: "local", delay: 60 * time.Millisecond, configured: true, profiles: []string{"a"}, calls: calls},
		"consul": &slowRepository{name: "consul", configured: true, profiles: []string{"b"}, calls: calls},
		"ssm":    &slowRepository{name: "ssm", configured: false, profiles: []string{"never"}, calls: calls},
		"vault":  &slowRepository{name: "vault", configured: true, profiles: []string{"c"}, calls: calls},
	})

	listed, err := listRepositories([]string{"local", "consul", "ssm", "vault"}, config.Config{})
	if err != nil {
		t.Fatalf("listRepositories(): %v", err)
	}

	var got []string
	for _, profiles := range listed {
		got = append(got, profiles...)
	}

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("listRepositories() = %v, want %v (an unconfigured repository contributes nothing)", got, want)
	}
}

// TestListRepositoriesReportsEveryFailure: a failing backend still names
// itself, and one failure does not hide another now that they run at once.
func TestListRepositoriesReportsEveryFailure(t *testing.T) {
	calls := &callLog{}

	useRepositories(t, map[string]repository.IRepository{
		"local":  &slowRepository{name: "local", configured: true, profiles: []string{"a"}, calls: calls},
		"consul": &slowRepository{name: "consul", configured: true, err: errors.New("connection refused"), calls: calls},
		"ssm":    &slowRepository{name: "ssm", configured: true, err: errors.New("security token is invalid"), calls: calls},
	})

	_, err := listRepositories([]string{"local", "consul", "ssm"}, config.Config{})
	if err == nil {
		t.Fatal("listRepositories() = nil error, want the two backend failures")
	}

	for _, want := range []string{"consul: connection refused", "ssm: security token is invalid"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// TestListRepositoriesUnknownRepository keeps a typo in supportedRepositories
// an error rather than a silently empty listing.
func TestListRepositoriesUnknownRepository(t *testing.T) {
	useRepositories(t, map[string]repository.IRepository{})

	if _, err := listRepositories([]string{"nope"}, config.Config{}); err == nil {
		t.Error("listRepositories([nope]) = nil error, want no matching Repository type")
	}
}
