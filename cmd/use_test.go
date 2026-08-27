package cmd

import "testing"

// TestResolveTarget pins the argument shapes `profiler use` accepts. The four
// provider branches this replaced each read their own argument index, which is
// how the local one came to shadow its profile and set the environment twice.
func TestResolveTarget(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantRepo    string
		wantProfile string
		wantErr     bool
	}{
		{
			name:        "bare profile name is local",
			args:        []string{"demo"},
			wantRepo:    "local",
			wantProfile: "demo",
		},
		{
			name:        "consul takes the profile name from args[1]",
			args:        []string{"consul", "staging"},
			wantRepo:    "consul",
			wantProfile: "staging",
		},
		{
			name:        "ssm takes the profile name from args[1]",
			args:        []string{"ssm", "staging"},
			wantRepo:    "ssm",
			wantProfile: "staging",
		},
		{
			name:        "vault takes the profile name from args[1]",
			args:        []string{"vault", "staging"},
			wantRepo:    "vault",
			wantProfile: "staging",
		},
		{
			name:    "consul without a profile name is an error, not a panic",
			args:    []string{"consul"},
			wantErr: true,
		},
		{
			name:    "ssm without a profile name is an error, not a panic",
			args:    []string{"ssm"},
			wantErr: true,
		},
		{
			name:    "vault without a profile name is an error, not a panic",
			args:    []string{"vault"},
			wantErr: true,
		},
		{
			// A local profile name that happens to look like a repository is
			// still a local profile: only the three remote names are special.
			name:        "an unknown first argument stays local",
			args:        []string{"azure", "staging"},
			wantRepo:    "local",
			wantProfile: "azure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, profileName, err := resolveTarget(tc.args)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveTarget(%q) = (%q, %q, nil), want an error",
						tc.args, repo, profileName)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveTarget(%q): %v", tc.args, err)
			}

			if repo != tc.wantRepo || profileName != tc.wantProfile {
				t.Errorf("resolveTarget(%q) = (%q, %q), want (%q, %q)",
					tc.args, repo, profileName, tc.wantRepo, tc.wantProfile)
			}
		})
	}
}
