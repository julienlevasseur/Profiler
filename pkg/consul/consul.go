package consul

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

func stringToByteSlice(value string) ([]byte, error) {
	r := strings.NewReader(value)
	b, err := io.ReadAll(r)
	if err != nil {
		return []byte{}, err
	}

	return b, err
}

func newConsulAPIClient() (*api.Client, error) {
	cfg := config.Get()
	client, err := api.NewClient(&api.Config{
		Address:   cfg.ConsulAddress,
		TokenFile: cfg.ConsulTokenFile,
		Token:     cfg.ConsulToken,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

func getProfilePath(profileName string) string {
	cfg := config.Get()

	return fmt.Sprintf(
		"%s/%s",
		cfg.ConsulProfilesPath,
		profileName,
	)
}

func profileExists(profileName string) (bool, error) {
	kvs, err := getKVPairs("profiler/" + profileName)
	if err != nil {
		return false, err
	}

	for _, kv := range kvs {
		if kv.Key == "profiler/"+profileName {
			return true, nil
		}
	}

	return false, nil
}

/*ListProfiles return the name of the Consul profiles as []string*/
func ListProfiles() ([]string, error) {
	kvs, err := getKVPairs("profiler/")
	if err != nil {
		return []string{}, err
	}

	// Consul list will return the `profiler` folder as a KV, removing it from
	// the slice because it doesn't need to be displayed:
	kvs = kvs[1:]

	var profiles []string
	for _, kv := range kvs {
		// Keys are named `profiler/Key`, removing the `profiler/` part for visibility:
		profiles = append(profiles, strings.Split(kv.Key, "/")[1])
	}
	return profiles, nil
}

func getKVPairs(path string) (api.KVPairs, error) {
	consul, err := newConsulAPIClient()
	if err != nil {
		return api.KVPairs{}, err
	}

	kvs, _, err := consul.KV().List(path, nil)
	if err != nil {
		return api.KVPairs{}, err
	}

	return kvs, nil
}

/* GetKVPair retrieve a single KV from Consul */
func GetKVPair(key string) (api.KVPair, error) {
	consul, err := newConsulAPIClient()
	if err != nil {
		return api.KVPair{}, err
	}

	kv, _, err := consul.KV().Get(key, nil)
	if err != nil {
		return api.KVPair{}, err
	}

	return *kv, nil
}

/* GetKVPairAsProfile retrn a Consul KVPair as a Profile */
func GetKVPairAsProfile(path string) (profile.Profile, error) {

	consul, err := newConsulAPIClient()
	if err != nil {
		return profile.Profile{}, err
	}

	kv, _, err := consul.KV().Get(fmt.Sprintf(path), nil)
	if err != nil {
		return profile.Profile{}, err
	}

	var kvs []profile.KV

	pairs := strings.Split(string(kv.Value), "\n")
	for _, pair := range pairs {
		// For now, only yaml data is supported for Consul KV:
		kv := strings.Split(pair, ": ")
		// ignore empty lines:
		if len(kv) < 2 {
			continue
		}
		kvs = append(kvs, profile.KV{
			Key:   kv[0],
			Value: kv[1],
		})
	}

	p := profile.Profile{
		Name: kv.Key,
		KVs:  kvs,
	}

	return p, nil
}

/*CreateProfilerFolder create the `/profiler` KV folder as the profiles placeholder in Consul*/
func CreateProfilerFolder() error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	profile := &api.KVPair{
		Key: "profiler/",
	}
	_, err = consul.KV().Put(profile, nil)
	if err != nil {
		return err
	}

	return nil
}

/*AddKVPair add one or more KV pairs to the given profile identified by profileName*/
func AddKVPair(profileName string, KVs []string) error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	if len(KVs) == 0 {
		// Create an empty profile:
		b, err := stringToByteSlice(
			fmt.Sprintf("profile_name: %s\n", profileName),
		)
		if err != nil {
			return err
		}

		profile := &api.KVPair{
			Key:   fmt.Sprintf("profiler/%s\n", profileName),
			Value: b,
		}
		consul.KV().Put(profile, nil)
	} else {
		var b []byte
		var actualKVPair api.KVPair

		exist, err := profileExists(profileName)
		if err != nil {
			return err
		}

		if exist {
			actualKVPair, err = GetKVPair(
				fmt.Sprintf("/profiler/%s", profileName),
			)
			if err != nil {
				return err
			}
		}

		for i := 0; i <= len(KVs)/2; i++ {
			var k string
			var v string

			k, KVs = KVs[i], KVs[1:]

			if len(KVs)/2 > 1 {
				v, KVs = KVs[i], KVs[1:]
			} else {
				v = KVs[i]
			}

			// Checking if provided argument is already present in KVPair:
			lines := strings.Split(string(actualKVPair.Value), "\n")
			for _, line := range lines {
				var key string
				var value string

				// Ignoring empty line:
				if len(line) == 0 {
					continue
				} else {
					kv := strings.Split(line, ": ")
					key = kv[0]
					value = kv[1]
				}

				if k == key && v == value {
					continue
				} else if k == key {
					// if the Key already exist but the value is different, it
					// will be handled as a new KV.
					continue
				} else {
					alreadyPresentKV, err := stringToByteSlice(
						fmt.Sprintf("%s: %s\n", key, value),
					)
					if err != nil {
						return err
					}

					b = append(b, alreadyPresentKV...)
				}
			}

			if i%2 == 0 {
				KV, err := stringToByteSlice(
					fmt.Sprintf("%s: %s\n", k, v),
				)
				if err != nil {
					return err
				}

				b = append(b, KV...)
			}

			if i == 1 {
				break
			}
		}

		profile := &api.KVPair{
			Key:   fmt.Sprintf("profiler/%s", profileName),
			Value: b,
		}
		err = DeleteKey(fmt.Sprintf("profiler/%s", profileName))
		if err != nil {
			return err
		}
		consul.KV().Put(profile, nil)
	}

	return nil
}

func GetProfile(profileName string) (profile.Profile, error) {
	cfg := config.Get()

	consul, err := newConsulAPIClient()
	if err != nil {
		return profile.Profile{}, err
	}

	kv, _, err := consul.KV().Get(fmt.Sprintf(
		"%s/%s",
		cfg.ConsulProfilesPath,
		profileName,
	), nil)
	if err != nil {
		return profile.Profile{}, err
	}

	var kvs []profile.KV

	pairs := strings.Split(string(kv.Value), "\n")
	for _, pair := range pairs {
		// For now, only yaml data is supported for Consul KV:
		kv := strings.Split(pair, ": ")
		// ignore empty lines:
		if len(kv) < 2 {
			continue
		}
		kvs = append(kvs, profile.KV{
			Key:   kv[0],
			Value: kv[1],
		})
	}

	p := profile.Profile{
		Name: kv.Key,
		KVs:  kvs,
	}

	return p, nil
}

/*ShowProfile return the list of keys for a profile*/
func ShowProfile(profileName string) ([]string, error) {
	var keys []string

	kv, err := GetKVPair("profiler/" + profileName)
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(string(kv.Value), "\n")
	for _, line := range lines {
		// Ignoring empty line:
		if len(line) == 0 {
			continue
		} else {
			kv := strings.Split(line, ": ")
			keys = append(keys, kv[0])
		}
	}

	return keys, nil
}

/* DeleteKey delete a Consul Key */
func DeleteKey(key string) error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	_, err = consul.KV().Delete(key, nil)
	if err != nil {
		return err
	}

	return nil
}

func profileToKVPair(p profile.Profile) (api.KVPair, error) {
	var kvp api.KVPair
	var b []byte

	kvp.Key = getProfilePath(p.Name)
	for _, kv := range p.KVs {
		c, err := stringToByteSlice(
			fmt.Sprintf("%s: %s\n", kv.Key, kv.Value),
		)
		if err != nil {
			return api.KVPair{}, err
		}

		b = append(b, c...)
	}
	kvp.Value = b

	return kvp, nil
}

func AddProfile(args []string) error {
	cfg := config.Get()

	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	profilerFolder, _, err := consul.KV().Get(cfg.ConsulProfilesPath, nil)
	if err != nil {
		return err
	}

	if profilerFolder == nil {
		f := &api.KVPair{
			Key: "profiler/",
		}
		_, err = consul.KV().Put(f, nil)
		if err != nil {
			return err
		}
	}

	if len(args) > 1 {
		if len(args) < 3 { // If no value provided:
			return errors.New("Please provide a value for the variable")
		}

		// If the number of args is even (profile name + an odd number of
		// arguments), this mean that a value is missing for its key:
		if len(args)%2 == 0 {
			return errors.New("Missing argument")
		}

		// Convert args to profile.KVs
		profileName, args := args[0], args[1:]
		var kvs []profile.KV
		for i := 0; i < len(args); i++ {
			kvs = append(
				kvs,
				profile.KV{
					Key:   args[i],
					Value: args[i+1],
				},
			)
			i++
		}

		// Does the profile already exists ?
		var actualProfile profile.Profile
		exists, err := profileExists(profileName)
		if err != nil {
			return err
		}
		if exists {
			actualProfile, err = GetKVPairAsProfile(getProfilePath(profileName))
			if err != nil {
				return err
			}

			kvs = append(kvs, actualProfile.KVs...)
		} else {
			kvs = append(kvs, profile.KV{Key: "profile_name", Value: profileName})
		}

		// Set the Profile
		var p = profile.Profile{
			Name: profileName,
			KVs:  kvs,
		}
		// Save the profile
		err = SaveProfile(p)
		if err != nil {
			return err
		}
	} else {
		// Just the name of profile has been provided, let's create an empty profile:
		var p = profile.Profile{
			Name: args[0],
		}
		// err = consul.SaveProfile(p)
		err = SaveProfile(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func SaveProfile(p profile.Profile) error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	kvPair, err := profileToKVPair(p)
	if err != nil {
		return err
	}

	_, err = consul.KV().Put(&kvPair, nil)
	if err != nil {
		return err
	}

	return nil
}

func RemoveProfile(args []string) error {
	pName := args[0]
	keys := args[1:]

	// Check if the profile exists
	exists, err := profileExists(pName)
	if err != nil {
		return err
	}

	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("The profile does not exist")
	} else {
		if len(args) == 0 {
			_, err := consul.KV().Delete(getProfilePath(pName), nil)
			if err != nil {
				return err
			}
		} else {
			// Retrieve the actual profile:
			p, err := GetKVPairAsProfile(getProfilePath(pName))
			if err != nil {
				return err
			}

			var updatedKVs []profile.KV

			for _, key := range keys {
				// the profile_name key cannot be removed (just remove the whole profile then)
				if key == "profile_name" {
					continue
				}

				for _, kv := range p.KVs {
					if key != kv.Key {
						updatedKVs = append(
							updatedKVs,
							profile.KV{
								Key:   kv.Key,
								Value: kv.Value,
							},
						)
					}
				}
			}

			prof := profile.Profile{
				Name: pName,
				KVs:  updatedKVs,
			}

			SaveProfile(prof)
		}
	}

	return nil
}
