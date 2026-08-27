package profile

import (
	"maps"
	"os"
	"slices"
	"strings"
)

// NoName is the name a profile gets when nothing names it -- a profile
// assembled from `.env` files that carry no profile_name of their own.
const NoName = "No-Name-Profile"

// StackSeparator joins the names of the profiles a composed profile was built
// from, so `profiler status` reports the whole stack rather than only the
// profile applied last.
const StackSeparator = "+"

// NameVar is the environment variable naming the profile in use. Every
// profile carries it, and `profiler status` reads it.
const NameVar = "profile_name"

// KeysVar is the environment variable SetEnvironment exports beside a
// profile's own variables, naming every key that profile owns.
//
// It is what makes stacking possible. The values of the profile in use are
// already in the process environment -- SetEnvironment ends in syscall.Exec,
// so the shell it lands in inherits them -- but nothing in that environment
// says which of its hundreds of variables came from a profile. KeysVar says
// exactly that, and ActiveProfile reads it back.
const KeysVar = "profile_keys"

// name returns the profile's name, preferring the profile_name variable it
// carries over the Name field. The two agree for local and SSM profiles; for
// Consul they do not, because Name there is the full KV path
// (`profiler/staging`) while profile_name is the profile's own name
// (`staging`). profile_name is the marker every profile sets, so it wins.
func (p Profile) name() string {
	for _, kv := range p.KVs {
		if kv.Key == NameVar && kv.Value != "" {
			return kv.Value
		}
	}

	return p.Name
}

// ComposeProfile merges profiles into one, left to right: a key carried by
// more than one of them takes its value from the last profile that carries
// it. `profiler use B`, run while profile A is in use, composes
// []Profile{A, B}, so B's values land over A's and everything A holds that B
// does not is kept.
//
// The composed profile is named after every profile it was built from, joined
// with StackSeparator -- `A+B` -- with duplicates dropped, so re-applying a
// profile already in the stack does not lengthen the name. A name that is
// itself a stack is split back into its parts first, which is what makes
// composing repeatedly idempotent.
//
// It is a function over []Profile rather than a method on Profile on purpose:
// the merge knows nothing about where its inputs came from, so stacking a
// local profile over an SSM one costs nothing extra.
func ComposeProfile(profiles []Profile) Profile {
	values := make(map[string]string)
	var names []string

	for _, p := range profiles {
		for _, part := range strings.Split(p.name(), StackSeparator) {
			if part == "" || part == NoName || slices.Contains(names, part) {
				continue
			}

			names = append(names, part)
		}

		for _, kv := range p.KVs {
			// Both markers are rewritten below from the composed profile, so
			// the inputs' copies are dropped rather than merged.
			if kv.Key == NameVar || kv.Key == KeysVar {
				continue
			}

			values[kv.Key] = kv.Value
		}
	}

	name := strings.Join(names, StackSeparator)
	if name == "" {
		name = NoName
	}
	values[NameVar] = name

	composed := Profile{Name: name}
	// Map iteration order is random and these KVs are written to the profiler
	// file in order, so sort to keep that file stable between runs:
	for _, key := range slices.Sorted(maps.Keys(values)) {
		composed.KVs = append(composed.KVs, KV{Key: key, Value: values[key]})
	}

	return composed
}

// ActiveProfile returns the profile currently in use, rebuilt from the
// environment SetEnvironment exported, and reports whether there is one.
//
// A key listed in KeysVar but no longer in the environment is dropped rather
// than rebuilt as empty, so `unset FOO` really does take FOO out of the
// profile instead of stacking an empty value onto the next one.
func ActiveProfile() (Profile, bool) {
	keys, ok := os.LookupEnv(KeysVar)
	if !ok || keys == "" {
		return Profile{}, false
	}

	p := Profile{Name: os.Getenv(NameVar)}

	for _, key := range strings.Split(keys, ",") {
		if key == "" || key == KeysVar {
			continue
		}

		if value, ok := os.LookupEnv(key); ok {
			p.KVs = append(p.KVs, KV{Key: key, Value: value})
		}
	}

	return p, true
}

// StackEnvironment sets the environment for p composed over the profile
// already in use. It is what every `use` command calls: stacking is the
// meaning of `use`, not an option on it, and the way back out is to leave the
// shell that `use` spawned.
//
// With no profile in use it is SetEnvironment with the composed profile's
// bookkeeping -- a sorted key order and a profile_name that agrees with the
// name -- so the first profile of a stack is written exactly like the second.
func StackEnvironment(p Profile) error {
	profiles := []Profile{p}

	if active, ok := ActiveProfile(); ok {
		profiles = []Profile{active, p}
	}

	return SetEnvironment(ComposeProfile(profiles))
}

// keysMarker returns the KeysVar entry for kvs: every key the profile owns,
// comma separated, in the order they are written.
func keysMarker(kvs []KV) KV {
	keys := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		if kv.Key != KeysVar {
			keys = append(keys, kv.Key)
		}
	}

	return KV{Key: KeysVar, Value: strings.Join(keys, ",")}
}

// withKeysMarker returns the profile's variables with KeysVar appended,
// replacing any stale copy read back from a profiler file.
func withKeysMarker(kvs []KV) []KV {
	marked := make([]KV, 0, len(kvs)+1)
	for _, kv := range kvs {
		if kv.Key != KeysVar {
			marked = append(marked, kv)
		}
	}

	return append(marked, keysMarker(kvs))
}
