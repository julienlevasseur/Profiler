package ssm

import (
	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/julienlevasseur/profiler/pkg/ssm"
)

type ssmRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
}

func NewSSMRepository() *ssmRepository {
	return &ssmRepository{
		Name: "ssm",
	}
}

func (r *ssmRepository) Add(args []string) error {
	return ssm.AddProfile(args)
}

func (r *ssmRepository) Get(name string) (profile.Profile, error) {
	return ssm.GetProfile(name)
}

func (r *ssmRepository) GetName() string {
	return r.Name
}

// IsConfigured reports whether AWS is set up on this machine -- see
// pkg/ssm.IsConfigured for what that means and what it costs.
func (r *ssmRepository) IsConfigured() bool {
	return ssm.IsConfigured()
}

func (r *ssmRepository) List() ([]string, error) {
	return ssm.ListProfiles()
}

func (r *ssmRepository) Remove(args []string) error {
	return ssm.RemoveProfile(args)
}

func (r *ssmRepository) Save(p profile.Profile) error {
	return ssm.SaveProfile(p)
}

func (r *ssmRepository) Show(name string) ([]string, error) {
	return ssm.ShowProfile(name)
}
