package vault

import (
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/pkg/vault"
)

type vaultRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
}

func NewVaultRepository() *vaultRepository {
	return &vaultRepository{
		Name: "vault",
	}
}

func (r *vaultRepository) Add(args []string) error {
	return vault.AddProfile(args)
}

func (r *vaultRepository) Get(name string) (profile.Profile, error) {
	return vault.GetProfile(name)
}

func (r *vaultRepository) GetName() string {
	return r.Name
}

func (r *vaultRepository) IsConfigured() bool {
	cfg := config.Get()

	if cfg.VaultAddress != "" && cfg.VaultToken != "" {
		return true
	}

	return false
}

func (r *vaultRepository) List() ([]string, error) {
	cfg := config.Get()
	secrets, err := vault.ListProfiles(cfg.VaultProfilesPath)
	if err != nil {
		return []string{}, err
	}

	var profiles []string
	profiles = append(profiles, secrets...)

	return profiles, nil
}

func (r *vaultRepository) Remove(args []string) error {
	return vault.RemoveProfile(args)
}

func (r *vaultRepository) Save(p profile.Profile) error {
	return vault.SaveProfile(p)
}

func (r *vaultRepository) Show(name string) ([]string, error) {
	return vault.ShowProfile(name)
}
