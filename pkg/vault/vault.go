package vault

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	// "github.com/hashicorp/vault/api"
	vault "github.com/hashicorp/vault-client-go"
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

func newVaultAPIClient() (*vault.Client, error) {
	cfg := config.Get()

	client, err := vault.New(
		vault.WithAddress(cfg.VaultAddress),
		vault.WithRequestTimeout(10*time.Second),
	)

	client.SetToken(cfg.VaultToken)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func getSecretPath(profileName string) string {
	cfg := config.Get()
	// return fmt.Sprintf("/secret/data/profiler/%s", profileName)
	return fmt.Sprintf("%s/%s", cfg.VaultProfilesPath, profileName)
}

func getSecret(path string) (map[string]interface{}, error) {
	// func getSecret(path string) (*api.KVSecret, error) {
	vault, err := newVaultAPIClient()
	if err != nil {
		return nil, err
	}

	secret, err := vault.Read(context.TODO(), path)
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}

func profileExists(profileName string) (bool, error) {
	secret, err := getSecret(getSecretPath(profileName))
	if err != nil {
		return false, err
	}

	if secret != nil {
		return true, nil
	}

	return false, nil
}

func getSecretAsProfile(profileName string) (profile.Profile, error) {
	s, err := getSecret(getSecretPath(profileName))
	if err != nil {
		return profile.Profile{}, err
	}

	var kvs []profile.KV

	for key, data := range s {
		if key == "data" {
			for k, v := range data.(map[string]interface{}) {
				kvs = append(kvs, profile.KV{
					Key:   k,
					Value: v.(string),
				})
			}
		}
	}

	p := profile.Profile{
		Name: profileName,
		KVs:  kvs,
	}

	return p, nil
}

func AddProfile(args []string) error {
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
			actualProfile, err = getSecretAsProfile(profileName)
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
	}

	return nil
}

func GetProfile(profileName string) (profile.Profile, error) {
	s, err := getSecret(getSecretPath(profileName))
	if err != nil {
		return profile.Profile{}, err
	}

	var p profile.Profile
	var kvs []profile.KV

	for key, data := range s {
		if key == "data" {
			for k, v := range data.(map[string]interface{}) {
				kvs = append(kvs, profile.KV{
					Key:   k,
					Value: v.(string),
				})
			}
		}
	}

	p = profile.Profile{
		Name: p.Name,
		KVs:  kvs,
	}

	return p, nil
}

/* ListProfiles return the name of the Vault profiles as []string */
func ListProfiles(path string) ([]string, error) {
	vault, err := newVaultAPIClient()
	if err != nil {
		return []string{}, err
	}

	data, err := vault.List(
		context.TODO(),
		// List works on metadata, replacing data to metadata in the path:
		strings.Replace(getSecretPath(""), "/data/", "/metadata/", 1),
	)
	if data == nil {
		return []string{}, fmt.Errorf("no keys found in \"%s\"", path)
	}

	if data.Data != nil {
		keys := []string{}

		k, ok := data.Data["keys"].([]interface{})
		if !ok {
			log.Fatalf("did not found any keys in %s", path)
		}

		for _, e := range k {
			keys = append(keys, fmt.Sprintf("%v", e))
		}

		return keys, nil
	}

	return []string{}, nil
}

func profileToMap(p profile.Profile) map[string]interface{} {
	data := make(map[string]interface{})

	for _, kv := range p.KVs {
		data[kv.Key] = kv.Value
	}

	return map[string]interface{}{"data": data}
}

func RemoveProfile(args []string) error {
	pName := args[0]
	keys := args[1:]

	// Check if the profile exists
	exists, err := profileExists(pName)
	if err != nil {
		return err
	}

	vault, err := newVaultAPIClient()
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("The profile does not exist")
	} else {
		if len(args) == 0 {
			_, err := vault.Delete(context.TODO(), getSecretPath(pName))
			if err != nil {
				return err
			}
		} else {
			p, err := getSecretAsProfile(pName)
			if err != nil {
				return err
			}

			// fmt.Println(getSecretPath(pName))
			// fmt.Println(p)

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

func SaveProfile(p profile.Profile) error {
	vault, err := newVaultAPIClient()
	if err != nil {
		return err
	}

	data := profileToMap(p)

	_, err = vault.Write(context.Background(), getSecretPath(p.Name), data)

	return err
}

func ShowProfile(profileName string) ([]string, error) {
	p, err := GetProfile(profileName)
	if err != nil {
		return []string{}, err
	}

	var keys []string
	for _, kv := range p.KVs {
		keys = append(keys, kv.Key)
	}

	return keys, nil
}
