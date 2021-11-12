package consul

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

func stringToByteSlice(value string) ([]byte, error) {
	r := strings.NewReader(value)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return []byte{}, err
	}

	return b, err
}

func newConsulAPIClient() (*api.Client, error) {
	client, err := api.NewClient(&api.Config{
		Address:   viper.GetString("consulAddress"),
		TokenFile: viper.GetString("consulTokenFile"),
		Token:     viper.GetString("consulToken"),
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

/*ProfileExist return a boolean representation of the given profile existence*/
func ProfileExist(profileName string) (bool, error) {
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

/*GetKVPair retrieve a single KV from Consul*/
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

/*CreateProfilerFolder create the `/profiler` KV folder as the profiles placeholder in Consul*/
func CreateProfilerFolder() error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	profile := &api.KVPair{
		Key: "profiler/",
	}
	consul.KV().Put(profile, nil)

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

		exist, err := ProfileExist(profileName)
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

/*DeleteKey delete a Consul Key*/
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
