package profile

import (
	"errors"

	"github.com/julienlevasseur/profiler/pkg/consul"
	"github.com/julienlevasseur/profiler/pkg/local"
	"github.com/julienlevasseur/profiler/pkg/repository"
	"github.com/spf13/viper"
)

type Profile struct {
	Name string
	KVs  map[string]string
	Type string
}

type IProfile interface {
	Add(profileType string, args []string) error
	Exists() (bool, error)
	Remove(p Profile) error
	Show() error
	Use()
}

func (p Profile) Add(profileType string, args []string) error {
	kvs, err := argsToKVs(args)
	if err != nil {
		return err
	}

	if p.Type == "local" {
		lp := local.LocalProfile{
			Name: p.Name,
			KVs:  kvs,
		}

		err := lp.Add()
		if err != nil {
			return err
		}
	} else if p.Type == "consul" {
		cp := consul.ConsulProfile{
			Name: p.Name,
			KVs:  kvs,
		}

		err := cp.Add()
		if err != nil {
			return err
		}
	} else if p.Type == "ssm" {
		return errors.New("[TODO]")
	} else {
		return errors.New("Unknown profile type")
	}

	return nil
}

// Create is the profile creation entrypoint. It is used by the `add` commands
// to manage the steps around profile creation.
func Create(p Profile, args []string) error {
	_, args = args[0], args[1:] // remove profile_name from args list

	err := p.Add(p.Type, args)
	if err != nil {
		return err
	}

	// Register the newly added profile to the repository to keep track
	// of the profile type:
	repo := repository.Repository{
		Path: viper.GetString("repositoryPath"),
	}
	rp := repository.Profile{
		Name:        p.Name,
		ProfileType: p.Type,
	}
	err = repo.Add(rp)
	if err != nil {
		return err
	}

	return nil
}

func argsToKVs(args []string) (map[string]string, error) {
	kvs := make(map[string]string)

	// If the number of args is even (profile name + an odd number of
	// arguments), this mean that a value is missing for its key,
	// because of the explanation above (profile_name + N(k:v) => Odd
	// number of arguments):
	if len(args)%2 != 0 {
		return kvs, errors.New("Missing argument")
	} else {
		for i := 0; i < len(args); i++ {
			kvs[args[i]] = args[i+1]
			i++ // reiterate over i because we parse the slice by pairs.
		}
	}

	return kvs, nil
}
