package local

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/spf13/viper"

	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
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

// TestRemoveProfileWholeProfile is the bare-name case: with no variable after
// the profile name, the profile file itself goes -- which is what cmd/local did
// inline with os.Remove before the repository method existed.
func TestRemoveProfileWholeProfile(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nFOO: bar\n")

	path := profilePath(config.Get(), "demo")

	if err := RemoveProfile([]string{"demo"}); err != nil {
		t.Fatalf("RemoveProfile([demo]): %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s still exists after RemoveProfile([demo])", path)
	}
}

// TestRemoveProfileVariables is the other case: the named variables go, the
// profile stays. Consul and vault take a whole slice here, so local does too --
// cmd/local only ever passed one.
func TestRemoveProfileVariables(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nFOO: bar\nBAZ: qux\nQUUX: quux\n")

	if err := RemoveProfile([]string{"demo", "FOO", "BAZ"}); err != nil {
		t.Fatalf("RemoveProfile([demo FOO BAZ]): %v", err)
	}

	vars, err := ShowProfile("demo")
	if err != nil {
		t.Fatalf("ShowProfile(%q): %v", "demo", err)
	}

	slices.Sort(vars)
	want := []string{"QUUX", "profile_name"}

	if !slices.Equal(vars, want) {
		t.Errorf("after RemoveProfile, keys = %v, want %v", vars, want)
	}
}

// TestRemoveProfileKeepsName guards the one key that cannot be removed on its
// own: without profile_name no loader can name the profile, so removing it is
// a whole-profile removal and has to be asked for that way.
func TestRemoveProfileKeepsName(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nFOO: bar\n")

	if err := RemoveProfile([]string{"demo", "profile_name"}); err != nil {
		t.Fatalf("RemoveProfile([demo profile_name]): %v", err)
	}

	p, err := GetProfile("demo")
	if err != nil {
		t.Fatalf("GetProfile(%q): %v", "demo", err)
	}

	if p.Name != "demo" {
		t.Errorf("profile name = %q, want %q", p.Name, "demo")
	}
}

// TestRemoveProfileUnknownProfile keeps a typo an error rather than a silent
// success, matching consul and vault.
func TestRemoveProfileUnknownProfile(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\n")

	if err := RemoveProfile([]string{"nope"}); err == nil {
		t.Error("RemoveProfile([nope]) returned no error for a profile that does not exist")
	}
}

func TestRemoveProfileNoArgs(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\n")

	if err := RemoveProfile(nil); err == nil {
		t.Error("RemoveProfile(nil) returned no error")
	}
}

// TestSaveProfileRoundTrip is the property Save has to hold, and the only path
// that exercises it: no command calls Save yet, it exists so a caller holding a
// Profile can write it back through IRepository.
func TestSaveProfileRoundTrip(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\n")

	p := profile.Profile{
		Name: "demo",
		KVs: []profile.KV{
			{Key: "FOO", Value: "bar"},
			{Key: "BAZ", Value: "qux"},
		},
	}

	if err := SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	env := profileEnv(t, "demo")

	for _, kv := range p.KVs {
		if env[kv.Key] != kv.Value {
			t.Errorf("%s = %q, want %q", kv.Key, env[kv.Key], kv.Value)
		}
	}

	if env["profile_name"] != "demo" {
		t.Errorf("profile_name = %q, want %q", env["profile_name"], "demo")
	}
}

// TestSaveProfileReplaces is what makes Save a save rather than the append
// AddProfile does: a key dropped from the profile is gone from the file.
func TestSaveProfileReplaces(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\nFOO: bar\nBAZ: qux\n")

	err := SaveProfile(profile.Profile{
		Name: "demo",
		KVs:  []profile.KV{{Key: "FOO", Value: "bar"}},
	})
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	vars, err := ShowProfile("demo")
	if err != nil {
		t.Fatalf("ShowProfile(%q): %v", "demo", err)
	}

	slices.Sort(vars)
	want := []string{"FOO", "profile_name"}

	if !slices.Equal(vars, want) {
		t.Errorf("after SaveProfile, keys = %v, want %v", vars, want)
	}
}

// TestSaveProfileNameFromKV covers the other way a name arrives. GetProfile
// lifts profile_name out of the map and into Profile.Name, but a profile built
// straight from a map still carries it in the KVs, so Save accepts both.
func TestSaveProfileNameFromKV(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\n")

	err := SaveProfile(profile.Profile{
		KVs: []profile.KV{
			{Key: "profile_name", Value: "other"},
			{Key: "FOO", Value: "bar"},
		},
	})
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	if _, err := os.Stat(profilePath(config.Get(), "other")); err != nil {
		t.Errorf("SaveProfile did not write the profile named in the KVs: %v", err)
	}
}

// TestSaveProfileNoName refuses the one profile there is nowhere to put: with
// no name in either place there is no path to write to.
func TestSaveProfileNoName(t *testing.T) {
	setupProfileDir(t, "demo", "profile_name: demo\n")

	err := SaveProfile(profile.Profile{
		KVs: []profile.KV{{Key: "FOO", Value: "bar"}},
	})
	if err == nil {
		t.Error("SaveProfile with no name returned no error")
	}
}
