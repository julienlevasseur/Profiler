package local

import (
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/local"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

type localRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
	// Profile localProfile `mapstructure:"profiles,omitempty" yaml:"profile,omitempty"`
}

func NewLocalRepository() *localRepository {
	return &localRepository{
		Name: "local",
	}
}

func (r *localRepository) Add(args []string) error {
	return local.AddProfile(args)
}

func (r *localRepository) IsConfigured() bool {
	// The local repository is always configured:
	return true
}

func (r *localRepository) GetName() string {
	return r.Name
}

func (r *localRepository) List() ([]string, error) {
	cfg := config.Get()
	return local.ListProfiles(cfg.ProfilesFolder)
}

func (r *localRepository) Get(name string) (profile.Profile, error) {
	return local.GetProfile(name)
}

func (r *localRepository) Save(p profile.Profile) error {
	return nil
}

func (r *localRepository) Remove(args []string) error {
	return nil
}

func (r *localRepository) Show(name string) ([]string, error) {
	return []string{}, nil
}
