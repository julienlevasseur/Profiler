package ssm

import "github.com/julienlevasseur/profiler/pkg/profile"

type ssmRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
}

func NewSSMRepository() *ssmRepository {
	return &ssmRepository{
		Name: "ssm",
	}
}

func (r *ssmRepository) Add(args []string) error {
	return nil
}

func (r *ssmRepository) IsConfigured() bool {
	return false
}

func (r *ssmRepository) GetName() string {
	return r.Name
}

func (r *ssmRepository) List() ([]string, error) {
	return []string{}, nil
}

func (r *ssmRepository) Get(name string) (profile.Profile, error) {
	return profile.Profile{}, nil
}

func (r *ssmRepository) Remove(args []string) error {
	return nil
}

func (r *ssmRepository) Save(p profile.Profile) error {
	return nil
}

func (r *ssmRepository) Show(name string) ([]string, error) {
	return []string{}, nil
}
