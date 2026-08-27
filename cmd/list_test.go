package cmd

import (
	"reflect"
	"testing"

	"github.com/julienlevasseur/profiler/config"
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
