package consul

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/spf13/viper"
)

// kvEntry is one entry of the fake store, in the shape the Consul KV endpoint
// returns it. Value is []byte so encoding/json base64s it, as Consul does.
type kvEntry struct {
	LockIndex   int    `json:"LockIndex"`
	Key         string `json:"Key"`
	Flags       int    `json:"Flags"`
	Value       []byte `json:"Value"`
	CreateIndex int    `json:"CreateIndex"`
	ModifyIndex int    `json:"ModifyIndex"`
}

// fakeConsul stands up something that answers /v1/kv the way a real agent
// does, so this package can be exercised without a Consul on the box. It is
// returned alongside the store it serves, so a test can seed it and then read
// back what a write actually put there.
//
// A recursive read of a prefix nothing matches answers 404, as Consul does:
// that is the empty-store path, and it reaches profiler as an empty listing
// rather than as an error.
type fakeConsul struct {
	mu    sync.Mutex
	store map[string][]byte
}

func newFakeConsul(t *testing.T, seed map[string]string) *fakeConsul {
	t.Helper()

	c := &fakeConsul{store: map[string][]byte{}}
	for k, v := range seed {
		c.store[k] = []byte(v)
	}

	srv := httptest.NewServer(http.HandlerFunc(c.serve))
	t.Cleanup(srv.Close)

	setConsulCfg(t, srv.URL)

	return c
}

func (c *fakeConsul) serve(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/v1/kv/")

	c.mu.Lock()
	defer c.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		// Consul sends these on every read, and the client parses them
		// before it looks at the body:
		w.Header().Set("X-Consul-Index", "1")
		w.Header().Set("X-Consul-LastContact", "0")
		w.Header().Set("X-Consul-KnownLeader", "true")

		var matching []kvEntry
		if _, recurse := r.URL.Query()["recurse"]; recurse {
			for k := range c.store {
				if strings.HasPrefix(k, key) {
					matching = append(matching, c.entry(k))
				}
			}
		} else if _, ok := c.store[key]; ok {
			matching = append(matching, c.entry(key))
		}

		if len(matching) == 0 {
			http.Error(w, "", http.StatusNotFound)
			return
		}

		// Consul answers in key order, and ListProfiles passes that order
		// straight through to `profiler list`:
		sort.Slice(matching, func(i, j int) bool {
			return matching[i].Key < matching[j].Key
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(matching)

	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c.store[key] = body
		fmt.Fprint(w, "true")

	case http.MethodDelete:
		delete(c.store, key)
		fmt.Fprint(w, "true")

	default:
		http.Error(w, "unexpected method "+r.Method, http.StatusBadRequest)
	}
}

func (c *fakeConsul) entry(key string) kvEntry {
	return kvEntry{Key: key, Value: c.store[key]}
}

// get reads a key back out of the fake store, which is how the write paths are
// checked: what profiler PUT is what a later `profiler consul list` would see.
func (c *fakeConsul) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.store[key]

	return string(v), ok
}

func (c *fakeConsul) keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]string, 0, len(c.store))
	for k := range c.store {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// setConsulCfg points profiler at address for the duration of one test.
// InitCfg never runs here, so consulProfilesPath is set explicitly rather than
// left to its default.
func setConsulCfg(t *testing.T, address string) {
	t.Helper()

	for k, v := range map[string]string{
		"consulAddress":      address,
		"consulProfilesPath": "profiler",
		"consulToken":        "",
		"consulTokenFile":    "",
	} {
		viper.Set(k, v)
		t.Cleanup(func() { viper.Set(k, "") })
	}
}

// demoStore is one profile of two variables plus a second profile, under the
// `profiler/` folder Consul returns as a KV pair of its own.
var demoStore = map[string]string{
	"profiler/":      "",
	"profiler/demo":  "profile_name: demo\nFOO: bar\n",
	"profiler/other": "profile_name: other\n",
}

func TestStringToByteSlice(t *testing.T) {
	b, err := stringToByteSlice("FOO: bar\n")
	if err != nil {
		t.Fatalf("stringToByteSlice() error: %v", err)
	}

	if string(b) != "FOO: bar\n" {
		t.Errorf("stringToByteSlice() = %q, want %q", b, "FOO: bar\n")
	}
}

// TestGetProfilePath: profiles live under consulProfilesPath, so a Consul
// shared with something else can keep them out of the way of everything else.
func TestGetProfilePath(t *testing.T) {
	viper.Set("consulProfilesPath", "profiler")
	t.Cleanup(func() { viper.Set("consulProfilesPath", "") })

	if got := getProfilePath("demo"); got != "profiler/demo" {
		t.Errorf("getProfilePath(demo) = %q, want %q", got, "profiler/demo")
	}

	viper.Set("consulProfilesPath", "teams/platform/profiler")

	if got := getProfilePath("demo"); got != "teams/platform/profiler/demo" {
		t.Errorf("getProfilePath(demo) = %q, want it under the configured path", got)
	}
}

// TestProfileToKVPair: a profile is stored as one KV pair holding a line per
// variable, which is the format GetProfile reads back.
func TestProfileToKVPair(t *testing.T) {
	viper.Set("consulProfilesPath", "profiler")
	t.Cleanup(func() { viper.Set("consulProfilesPath", "") })

	kvp, err := profileToKVPair(profile.Profile{
		Name: "demo",
		KVs: []profile.KV{
			{Key: "profile_name", Value: "demo"},
			{Key: "FOO", Value: "bar"},
		},
	})
	if err != nil {
		t.Fatalf("profileToKVPair() error: %v", err)
	}

	if kvp.Key != "profiler/demo" {
		t.Errorf("Key = %q, want %q", kvp.Key, "profiler/demo")
	}

	want := "profile_name: demo\nFOO: bar\n"
	if string(kvp.Value) != want {
		t.Errorf("Value = %q, want %q", kvp.Value, want)
	}
}

// TestProfileToKVPairEmpty: a profile with no variables is an empty value, not
// an error -- `profiler consul add demo` creates one.
func TestProfileToKVPairEmpty(t *testing.T) {
	viper.Set("consulProfilesPath", "profiler")
	t.Cleanup(func() { viper.Set("consulProfilesPath", "") })

	kvp, err := profileToKVPair(profile.Profile{Name: "demo"})
	if err != nil {
		t.Fatalf("profileToKVPair() error: %v", err)
	}

	if len(kvp.Value) != 0 {
		t.Errorf("Value = %q, want nothing", kvp.Value)
	}
}

// TestListProfiles is what reaches `profiler list`: every profile named once,
// and the `profiler/` folder -- which Consul returns as a KV pair but which is
// not a profile -- left out.
func TestListProfiles(t *testing.T) {
	newFakeConsul(t, demoStore)

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error: %v", err)
	}

	want := []string{"demo", "other"}
	if len(profiles) != len(want) {
		t.Fatalf("ListProfiles() = %v, want %v", profiles, want)
	}

	for i := range want {
		if profiles[i] != want[i] {
			t.Errorf("ListProfiles()[%d] = %q, want %q", i, profiles[i], want[i])
		}
	}
}

// TestListProfilesEmpty: a Consul with no profiles at all answers 404 to the
// recursive read, which is an empty listing rather than an error.
func TestListProfilesEmpty(t *testing.T) {
	newFakeConsul(t, nil)

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error: %v", err)
	}

	if len(profiles) != 0 {
		t.Errorf("ListProfiles() = %v, want no profiles", profiles)
	}
}

// TestListProfilesFolderOnly: the folder exists but holds nothing, which is
// what CreateProfilerFolder leaves behind on a fresh Consul.
func TestListProfilesFolderOnly(t *testing.T) {
	newFakeConsul(t, map[string]string{"profiler/": ""})

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error: %v", err)
	}

	if len(profiles) != 0 {
		t.Errorf("ListProfiles() = %v, want no profiles", profiles)
	}
}

// TestGetProfile is the `profiler consul use` path: the profile comes back
// with its values, which is what SetEnvironment exports.
func TestGetProfile(t *testing.T) {
	newFakeConsul(t, demoStore)

	p, err := GetProfile("demo")
	if err != nil {
		t.Fatalf("GetProfile() error: %v", err)
	}

	got := map[string]string{}
	for _, kv := range p.KVs {
		got[kv.Key] = kv.Value
	}

	want := map[string]string{"profile_name": "demo", "FOO": "bar"}
	if len(got) != len(want) {
		t.Fatalf("GetProfile().KVs = %v, want %v", p.KVs, want)
	}

	for k, v := range want {
		if got[k] != v {
			t.Errorf("GetProfile() %s = %q, want %q", k, got[k], v)
		}
	}
}

// TestGetKVPairAsProfile: the same parse, reached by full key rather than by
// profile name. A line carrying no `key: value` is not a variable -- the
// trailing newline of every stored profile produces one.
func TestGetKVPairAsProfile(t *testing.T) {
	newFakeConsul(t, map[string]string{
		"profiler/demo": "profile_name: demo\nFOO: bar\n\nmalformed\n",
	})

	p, err := GetKVPairAsProfile("profiler/demo")
	if err != nil {
		t.Fatalf("GetKVPairAsProfile() error: %v", err)
	}

	if len(p.KVs) != 2 {
		t.Fatalf("KVs = %v, want the two variables of demo", p.KVs)
	}
}

// TestShowProfile is `profiler consul show demo`: the variable names, without
// their values.
func TestShowProfile(t *testing.T) {
	newFakeConsul(t, demoStore)

	keys, err := ShowProfile("demo")
	if err != nil {
		t.Fatalf("ShowProfile() error: %v", err)
	}

	want := []string{"profile_name", "FOO"}
	if len(keys) != len(want) {
		t.Fatalf("ShowProfile() = %v, want %v", keys, want)
	}

	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("ShowProfile()[%d] = %q, want %q", i, keys[i], want[i])
		}
	}
}

func TestGetKVPair(t *testing.T) {
	newFakeConsul(t, demoStore)

	kv, err := GetKVPair("profiler/demo")
	if err != nil {
		t.Fatalf("GetKVPair() error: %v", err)
	}

	if kv.Key != "profiler/demo" {
		t.Errorf("Key = %q, want %q", kv.Key, "profiler/demo")
	}

	if string(kv.Value) != demoStore["profiler/demo"] {
		t.Errorf("Value = %q, want %q", kv.Value, demoStore["profiler/demo"])
	}
}

// TestSaveProfile checks what actually lands in Consul: a later GetProfile has
// to read back what this wrote.
func TestSaveProfile(t *testing.T) {
	c := newFakeConsul(t, nil)

	err := SaveProfile(profile.Profile{
		Name: "demo",
		KVs: []profile.KV{
			{Key: "profile_name", Value: "demo"},
			{Key: "FOO", Value: "bar"},
		},
	})
	if err != nil {
		t.Fatalf("SaveProfile() error: %v", err)
	}

	got, ok := c.get("profiler/demo")
	if !ok {
		t.Fatalf("SaveProfile() stored nothing at profiler/demo, store holds %v", c.keys())
	}

	want := "profile_name: demo\nFOO: bar\n"
	if got != want {
		t.Errorf("stored %q, want %q", got, want)
	}

	// The round trip is the point: what was written is a profile again.
	p, err := GetProfile("demo")
	if err != nil {
		t.Fatalf("GetProfile() after SaveProfile() error: %v", err)
	}

	if len(p.KVs) != 2 {
		t.Errorf("GetProfile().KVs = %v, want the two variables written", p.KVs)
	}
}

func TestCreateProfilerFolder(t *testing.T) {
	c := newFakeConsul(t, nil)

	if err := CreateProfilerFolder(); err != nil {
		t.Fatalf("CreateProfilerFolder() error: %v", err)
	}

	if _, ok := c.get("profiler/"); !ok {
		t.Errorf("CreateProfilerFolder() left %v, want the profiler/ folder", c.keys())
	}
}

func TestDeleteKey(t *testing.T) {
	c := newFakeConsul(t, demoStore)

	if err := DeleteKey("profiler/demo"); err != nil {
		t.Fatalf("DeleteKey() error: %v", err)
	}

	if _, ok := c.get("profiler/demo"); ok {
		t.Error("DeleteKey() left profiler/demo behind")
	}

	if _, ok := c.get("profiler/other"); !ok {
		t.Error("DeleteKey() removed profiler/other, which it was not asked about")
	}
}

// TestAddProfileNameOnly: `profiler consul add demo` creates an empty profile
// rather than requiring a variable up front.
func TestAddProfileNameOnly(t *testing.T) {
	c := newFakeConsul(t, nil)

	if err := AddProfile([]string{"demo"}); err != nil {
		t.Fatalf("AddProfile(demo) error: %v", err)
	}

	if _, ok := c.get("profiler/demo"); !ok {
		t.Errorf("AddProfile(demo) left %v, want profiler/demo", c.keys())
	}
}

// TestAddProfileNewProfile: a profile that does not exist yet is created with
// the variable asked for, plus the profile_name that names it.
func TestAddProfileNewProfile(t *testing.T) {
	c := newFakeConsul(t, map[string]string{"profiler/": ""})

	if err := AddProfile([]string{"demo", "FOO", "bar"}); err != nil {
		t.Fatalf("AddProfile(demo FOO bar) error: %v", err)
	}

	got, ok := c.get("profiler/demo")
	if !ok {
		t.Fatalf("AddProfile() left %v, want profiler/demo", c.keys())
	}

	for _, want := range []string{"FOO: bar\n", "profile_name: demo\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("stored %q, want it to hold %q", got, want)
		}
	}
}

// TestAddProfileKeepsExistingVariables: adding a variable to a profile that
// already has some keeps the ones already there -- an add is not a replace.
func TestAddProfileKeepsExistingVariables(t *testing.T) {
	c := newFakeConsul(t, map[string]string{
		"profiler/":     "",
		"profiler/demo": "profile_name: demo\nFOO: bar\n",
	})

	if err := AddProfile([]string{"demo", "BAZ", "qux"}); err != nil {
		t.Fatalf("AddProfile(demo BAZ qux) error: %v", err)
	}

	got, _ := c.get("profiler/demo")
	for _, want := range []string{"BAZ: qux\n", "FOO: bar\n", "profile_name: demo\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("stored %q, want it to hold %q", got, want)
		}
	}
}

// TestAddProfileMissingValue: a key with no value never reaches Consul.
func TestAddProfileMissingValue(t *testing.T) {
	newFakeConsul(t, demoStore)

	if err := AddProfile([]string{"demo", "FOO"}); err == nil {
		t.Error("AddProfile(demo FOO) = nil, want an error naming the missing value")
	}
}

// TestAddProfileMissingArgument: variables come in pairs, so an even number of
// arguments after the profile name means one is missing its other half.
func TestAddProfileMissingArgument(t *testing.T) {
	c := newFakeConsul(t, demoStore)

	if err := AddProfile([]string{"demo", "FOO", "bar", "BAZ"}); err == nil {
		t.Error("AddProfile(demo FOO bar BAZ) = nil, want an error naming the missing argument")
	}

	if got, _ := c.get("profiler/demo"); got != demoStore["profiler/demo"] {
		t.Errorf("profiler/demo = %q, want it untouched by a rejected add", got)
	}
}

// TestAddProfileCreatesFolder: a Consul that has never held a profile gets the
// `profiler/` folder before the first profile is written to it.
func TestAddProfileCreatesFolder(t *testing.T) {
	c := newFakeConsul(t, nil)

	if err := AddProfile([]string{"demo", "FOO", "bar"}); err != nil {
		t.Fatalf("AddProfile() error: %v", err)
	}

	if _, ok := c.get("profiler/"); !ok {
		t.Errorf("AddProfile() left %v, want the profiler/ folder created", c.keys())
	}
}

// TestRemoveProfileUnknown: removing something that is not there is an error,
// not a silent success.
func TestRemoveProfileUnknown(t *testing.T) {
	newFakeConsul(t, demoStore)

	if err := RemoveProfile([]string{"nope", "FOO"}); err == nil {
		t.Error("RemoveProfile(nope) = nil, want an error saying the profile does not exist")
	}
}

// TestRemoveProfileVariable: `profiler consul remove demo FOO` drops that
// variable and leaves the rest of the profile alone.
func TestRemoveProfileVariable(t *testing.T) {
	c := newFakeConsul(t, map[string]string{
		"profiler/":     "",
		"profiler/demo": "profile_name: demo\nFOO: bar\n",
	})

	if err := RemoveProfile([]string{"demo", "FOO"}); err != nil {
		t.Fatalf("RemoveProfile(demo FOO) error: %v", err)
	}

	got, _ := c.get("profiler/demo")

	if strings.Contains(got, "FOO") {
		t.Errorf("profiler/demo = %q, want FOO removed", got)
	}

	if !strings.Contains(got, "profile_name: demo") {
		t.Errorf("profiler/demo = %q, want the profile_name kept", got)
	}
}

// TestRemoveProfileVariables: several variables can go at once. Each key was
// filtered against the whole profile on its own, so every variable survived
// the key that was not its own -- two keys removed neither, and wrote the ones
// they kept once per key.
func TestRemoveProfileVariables(t *testing.T) {
	c := newFakeConsul(t, map[string]string{
		"profiler/":     "",
		"profiler/demo": "profile_name: demo\nFOO: bar\nBAZ: qux\nKEEP: me\n",
	})

	if err := RemoveProfile([]string{"demo", "FOO", "BAZ"}); err != nil {
		t.Fatalf("RemoveProfile(demo FOO BAZ) error: %v", err)
	}

	got, _ := c.get("profiler/demo")

	for _, gone := range []string{"FOO", "BAZ"} {
		if strings.Contains(got, gone) {
			t.Errorf("profiler/demo = %q, want %s removed", got, gone)
		}
	}

	want := "profile_name: demo\nKEEP: me\n"
	if got != want {
		t.Errorf("profiler/demo = %q, want %q", got, want)
	}
}

// TestRemoveProfileKeepsProfileName: profile_name is what names the profile,
// so it is not removable on its own -- removing the whole profile is the way.
func TestRemoveProfileKeepsProfileName(t *testing.T) {
	c := newFakeConsul(t, map[string]string{
		"profiler/":     "",
		"profiler/demo": "profile_name: demo\nFOO: bar\n",
	})

	if err := RemoveProfile([]string{"demo", "profile_name"}); err != nil {
		t.Fatalf("RemoveProfile(demo profile_name) error: %v", err)
	}

	if got, _ := c.get("profiler/demo"); !strings.Contains(got, "profile_name: demo") {
		t.Errorf("profiler/demo = %q, want the profile_name kept", got)
	}
}

// TestProfileExists covers both answers, since AddProfile branches on it:
// found means merge into what is there, not found means create.
func TestProfileExists(t *testing.T) {
	newFakeConsul(t, demoStore)

	exists, err := profileExists("demo")
	if err != nil {
		t.Fatalf("profileExists(demo) error: %v", err)
	}

	if !exists {
		t.Error("profileExists(demo) = false, want true")
	}

	exists, err = profileExists("nope")
	if err != nil {
		t.Fatalf("profileExists(nope) error: %v", err)
	}

	if exists {
		t.Error("profileExists(nope) = true, want false")
	}
}

// TestGetKVPairsUnreachable: nothing listening is an error rather than an
// empty listing, so `profiler list` can tell a Consul with no profiles from a
// Consul that is not there.
func TestGetKVPairsUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedAddr := srv.URL
	srv.Close()

	setConsulCfg(t, closedAddr)

	if _, err := ListProfiles(); err == nil {
		t.Error("ListProfiles() = nil error with nothing listening at consulAddress")
	}
}
