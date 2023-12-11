package consul

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/local"
	"github.com/spf13/viper"
)

type ConsulConfig struct {
	Enabled     bool
	ConsulAddr  string
	ConsulToken string
}

type ConsulProfile struct {
	Name string
	KVs  map[string]string
}

func (cp ConsulProfile) Add() error {
	err := addKVPair(cp.Name, cp.KVs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}

func (cp ConsulProfile) Exists() (bool, error) {
	kvs, err := getKVPairs("profiler/" + cp.Name)
	if err != nil {
		return false, err
	}

	for _, kv := range kvs {
		if kv.Key == "profiler/"+cp.Name {
			return true, nil
		}
	}

	return false, nil
}

func (cp ConsulProfile) Show() error {

	kv, err := getKVPair(
		fmt.Sprintf("/profiler/%s", cp.Name),
	)
	if err != nil {
		return err
	}

	lines := strings.Split(string(kv.Value), "\n")
	for _, line := range lines {
		// Ignoring empty line:
		if len(line) == 0 {
			continue
		} else {
			kv := strings.Split(line, ": ")
			fmt.Println(kv[0])
		}
	}

	return nil
}

func (cp ConsulProfile) Use() {
	envVars := make(map[string]string)

	p, err := getKVPair(cp.Name)
	if err != nil {
		failure.ExitOnError(err)
	}

	kvs := strings.Split(string(p.Value), "\n")
	for _, kv := range kvs {
		if kv != "" {
			entry := strings.Split(string(kv), ": ")
			envVars[entry[0]] = entry[1]
		}
	}

	local.SetEnvironment(envVars)
}

func NewConsulConfig() (ConsulConfig, error) {
	cc := ConsulConfig{}

	if viper.GetString("consulAddress") != "" {
		cc.Enabled = true
		cc.ConsulAddr = viper.GetString("consulAddress")
		if viper.GetString("consulToken") != "" {
			cc.ConsulToken = viper.GetString("consulToken")
		}

		// Ensure that the "profiler/" KV folder exists on Consul KV Store:
		ok, err := profileFolderExists()
		if err != nil {
			return ConsulConfig{}, err
		}

		if !ok {
			createProfilerFolder()
		}
	}

	return cc, nil
}

func stringToByteSlice(value string) ([]byte, error) {
	r := strings.NewReader(value)
	b, err := io.ReadAll(r)
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

// GetKVParis return a list of KVPair
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

// getKVPair retrieve a single KV from Consul
func getKVPair(key string) (api.KVPair, error) {
	consul, err := newConsulAPIClient()
	if err != nil {
		return api.KVPair{}, err
	}

	kv, _, err := consul.KV().Get(fmt.Sprintf("profiler/%s", key), nil)
	if err != nil {
		return api.KVPair{}, err
	}

	return *kv, nil
}

// profileFolderExists return a boolean representation of the profiler folder existence
func profileFolderExists() (bool, error) {
	kvs, err := getKVPairs("profiler/")
	if err != nil {
		return false, err
	}

	for _, kv := range kvs {
		if kv.Key == "profiler/" {
			return true, nil
		}
	}

	return false, nil
}

// createProfilerFolder create the `/profiler` KV folder as the profiles placeholder in Consul
func createProfilerFolder() error {
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

// addKVPair add one or more KV pairs to the given profile identified by profileName
func addKVPair(profileName string, KVs map[string]string) error {
	if len(KVs) == 0 {
		// Create an empty profile:
		return createEmptyProfile(profileName)
	}

	kvPair, err := getKVPair(profileName)
	if err != nil {
		return err
	}

	for k, v := range KVs {
		ok, _, err := foundInKV(profileName, k)
		if err != nil {
			return err
		}

		if !ok {
			// Does not exists yet in the KVPair, add it:
			err = appendToKVPair(&kvPair, k, v)
			if err != nil {
				return err
			}
		} else {
			// Already exists in the file, does it has the same value ?
			//values, err := manageExistingVariable(&kvPair, k, v)
			err := manageExistingVariable(&kvPair, k, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createEmptyProfile(profileName string) error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	b, err := stringToByteSlice(
		fmt.Sprintf("profile_name: %s", profileName),
	)
	if err != nil {
		return err
	}

	profile := &api.KVPair{
		Key:   fmt.Sprintf("profiler/%s", profileName),
		Value: b,
	}
	_, err = consul.KV().Put(profile, nil)
	if err != nil {
		return err
	}

	return nil
}

func foundInKV(KVName, match string) (bool, int, error) {
	kv, err := getKVPair(KVName)
	if err != nil {
		return false, 0, err
	}

	lines := strings.Split(string(kv.Value), "\n")
	for i, line := range lines {
		// Ignoring empty line:
		if len(line) == 0 {
			continue
		} else {
			if strings.Contains(line, match) && match != "" {
				return true, i, nil
			}
		}
	}

	return false, 0, nil
}

func appendToKVPair(kv *api.KVPair, key, value string) error {
	consul, err := newConsulAPIClient()
	if err != nil {
		return err
	}

	b, err := stringToByteSlice(
		string(kv.Value) + fmt.Sprintf("\n%v: %v", key, value),
	)
	if err != nil {
		return err
	}

	profile := &api.KVPair{
		Key:   kv.Key,
		Value: b,
	}
	_, err = consul.KV().Put(profile, nil)
	if err != nil {
		return err
	}

	return nil
}

// func manageExistingVariable(kv *api.KVPair, k, v string) ([]string, error) {
func manageExistingVariable(kv *api.KVPair, k, v string) error {
	// Search for "key: value" combination to check if it already exists.
	ok := strings.Contains(string(kv.Value), fmt.Sprintf("%v: %v", k, v))

	if ok {
		// It has the same value, no update required.
		return nil
	} else {
		// If not, update the entry
		value := string(kv.Value)

		var l string
		var output []string
		for _, line := range strings.Split(string(value), "\n") {
			if strings.Contains(line, fmt.Sprintf("%v:", k)) {
				l = fmt.Sprintf("%v: %v", k, v)
			} else {
				l = line
			}
			output = append(output, l)
		}

		//return output, nil
		consul, err := newConsulAPIClient()
		if err != nil {
			return err
		}

		var v string
		for _, value := range output {
			v += value + "\n"
		}

		b, err := stringToByteSlice(v)
		if err != nil {
			return err
		}

		profile := &api.KVPair{
			Key:   kv.Key,
			Value: b,
		}

		_, err = consul.KV().Put(profile, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

/*ShowProfile return the list of keys for a profile*/
// func ShowProfile(profileName string) ([]string, error) {
// 	var keys []string

// 	kv, err := GetKVPair("profiler/" + profileName)
// 	if err != nil {
// 		return []string{}, err
// 	}

// 	lines := strings.Split(string(kv.Value), "\n")
// 	for _, line := range lines {
// 		// Ignoring empty line:
// 		if len(line) == 0 {
// 			continue
// 		} else {
// 			kv := strings.Split(line, ": ")
// 			keys = append(keys, kv[0])
// 		}
// 	}

// 	return keys, nil
// }

// deleteKey delete a Consul Key
func deleteKey(key string) error {
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
