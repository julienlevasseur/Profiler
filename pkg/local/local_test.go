package local

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/spf13/viper"

	"github.com/julienlevasseur/profiler/config"
)

// profileEnv flattens a profile's KVs into a map, so a test can assert on the
// value a key ended up with rather than on slice ordering.
func profileEnv(t *testing.T, profileName string) map[string]string {
	t.Helper()

	p, err := GetProfile(profileName)
	if err != nil {
		t.Fatalf("GetProfile(%q): %v", profileName, err)
	}

	env := make(map[string]string, len(p.KVs))
	for _, kv := range p.KVs {
		env[kv.Key] = kv.Value
	}

	return env
}

// writeFile writes name inside dir, failing the test rather than returning.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// setupProfileDir points profilesFolder at a fresh directory holding a single
// named profile, and moves the test into an empty working directory -- the
// dot-env loaders all resolve their files relative to the cwd.
func setupProfileDir(t *testing.T, profileName, profileContent string) {
	t.Helper()

	profilesFolder := t.TempDir()
	writeFile(t, profilesFolder, "."+profileName+".yml", profileContent)

	viper.Set("profilesFolder", profilesFolder)
	t.Cleanup(func() { viper.Set("profilesFolder", "") })

	t.Chdir(t.TempDir())
}

// TestGetProfileLayersDotEnvFiles is the regression test for the three
// identical loadDotEnvFiles calls: .env.yml and .envrc were never merged into a
// named profile, so the direnv-style layering was silently dead.
func TestGetProfileLayersDotEnvFiles(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nFROM_PROFILE: profile\nLAYERED: profile\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, cwd, "app.env", "export FROM_DOT_ENV=dotenv\nexport LAYERED=dotenv\n")
	writeFile(t, cwd, ".env.yml", "FROM_ENV_YAML: envyaml\nLAYERED: envyaml\n")
	writeFile(t, cwd, ".envrc", "export FROM_ENVRC=envrc\nexport LAYERED=envrc\n")

	env := profileEnv(t, "demo")

	// Every loader ran: each file contributed its own key.
	for _, tc := range []struct{ key, want string }{
		{"FROM_PROFILE", "profile"},
		{"FROM_DOT_ENV", "dotenv"},
		{"FROM_ENV_YAML", "envyaml"},
		{"FROM_ENVRC", "envrc"},
	} {
		if got := env[tc.key]; got != tc.want {
			t.Errorf("%s = %q, want %q", tc.key, got, tc.want)
		}
	}

	// And they ran in the documented order: profile < *.env < .env.yml <
	// .envrc, so the last writer of a shared key wins.
	if got := env["LAYERED"]; got != "envrc" {
		t.Errorf("LAYERED = %q, want %q (.envrc has the last word)", got, "envrc")
	}

	if got := env["profile_name"]; got != "demo" {
		t.Errorf("profile_name = %q, want %q", got, "demo")
	}
}

// TestGetProfileDotEnvYamlBeatsDotEnv pins the middle of the precedence chain,
// which a single .envrc-wins assertion would not catch.
func TestGetProfileDotEnvYamlBeatsDotEnv(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nLAYERED: profile\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, cwd, "app.env", "export LAYERED=dotenv\n")
	writeFile(t, cwd, ".env.yml", "LAYERED: envyaml\n")

	if got := profileEnv(t, "demo")["LAYERED"]; got != "envyaml" {
		t.Errorf("LAYERED = %q, want %q (.env.yml overrides *.env)", got, "envyaml")
	}
}

// TestGetProfileWithoutDotEnvFiles keeps the profile file authoritative when
// there is nothing in the working directory to layer on top of it.
func TestGetProfileWithoutDotEnvFiles(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nLAYERED: profile\n")

	if got := profileEnv(t, "demo")["LAYERED"]; got != "profile" {
		t.Errorf("LAYERED = %q, want %q", got, "profile")
	}
}

// TestShowProfileReturnsProfileKeys is the regression test for the &profileName
// formatted into the file path: ShowProfile looked for a file named after a
// pointer address, so it never found the profile it was asked about.
func TestShowProfileReturnsProfileKeys(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nFOO: bar\nBAZ: qux\n")

	vars, err := ShowProfile("demo")
	if err != nil {
		t.Fatalf("ShowProfile(%q): %v", "demo", err)
	}

	slices.Sort(vars)
	want := []string{"BAZ", "FOO", "profile_name"}

	if !slices.Equal(vars, want) {
		t.Errorf("ShowProfile(%q) = %v, want %v", "demo", vars, want)
	}
}

// TestShowProfileUnknownProfile keeps the not-found case an error rather than
// an empty success -- before the fix every lookup failed this way, which is
// what made the pointer bug look like normal behaviour.
func TestShowProfileUnknownProfile(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\n")

	if _, err := ShowProfile("nope"); err == nil {
		t.Error("ShowProfile(\"nope\") returned no error for a profile that does not exist")
	}
}

// TestProfilePathIsTheDocumentedLayout pins the layout AddProfile, GetProfile
// and ShowProfile now share: a dot-prefixed .yml inside profilesFolder.
func TestProfilePathIsTheDocumentedLayout(t *testing.T) {
	cfg := config.Config{ProfilesFolder: "/tmp/profiles"}

	if got, want := profilePath(cfg, "demo"), "/tmp/profiles/.demo.yml"; got != want {
		t.Errorf("profilePath = %q, want %q", got, want)
	}
}
