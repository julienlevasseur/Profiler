package repository

import "github.com/julienlevasseur/profiler/profile"

type Repository struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type IRepository interface {
	Save(p profile.Profile) error
	Get() (profile.Profile, error)
	List() []string
}

func New() Repository {
	return Repository{}
}

func (r Repository) Save(p profile.Profile) error {
	return nil
}

func (r Repository) Get() (profile.Profile, error) {
	return profile.Profile{}, nil
}

func (r Repository) List() []string {
	return nil
}
