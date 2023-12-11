package repository

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/viper"
)

type Profile struct {
	Name        string `json:"name"`
	ProfileType string `json:"profile_type"`
}

type Repository struct {
	Path     string
	Profiles []Profile
}

type IRepository interface {
	List() ([]Profile, error)
	Get(name string) (Profile, error)
	Add(p Profile) error
	Del(name string) error
}

func readJsonFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return b, err
}

func (r Repository) List() ([]Profile, error) {
	b, err := readJsonFile(r.Path)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	json.Unmarshal(b, &profiles)

	return profiles, nil
}

func (r Repository) Get(name string) (Profile, error) {
	b, err := readJsonFile(r.Path)
	if err != nil {
		return Profile{}, err
	}

	var profiles []Profile
	var profile Profile

	json.Unmarshal(b, &profiles)

	for _, p := range profiles {
		if p.Name == name {
			profile = p
		}
	}

	return profile, nil
}

func (r Repository) Add(p Profile) error {
	actualRepository, err := r.List()
	if err != nil {
		return err
	}

	var alreadyRegistered bool
	for _, i := range actualRepository {
		if i.Name == p.Name {
			alreadyRegistered = true
		}
	}
	if !alreadyRegistered {
		actualRepository = append(actualRepository, p)

		repo, err := json.Marshal(actualRepository)
		if err != nil {
			return err
		}
		err = os.WriteFile(viper.GetString("repositoryPath"), repo, 0644)
	}

	return nil
}

func (r Repository) Del(name string) error {
	return nil
}
