package consul

import (
	"io"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/consul"
	"github.com/julienlevasseur/profiler/profile"
)

type consulRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
}

func NewConsulRepository() *consulRepository {
	return &consulRepository{
		Name: "consul",
	}
}

func newConsulAPIClient() (*api.Client, error) {
	cfg := config.Get()

	client, err := api.NewClient(&api.Config{
		Address:   cfg.ConsulAddress,
		TokenFile: cfg.ConsulTokenFile,
		Token:     cfg.ConsulToken,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

func stringToByteSlice(value string) ([]byte, error) {
	r := strings.NewReader(value)
	b, err := io.ReadAll(r)
	if err != nil {
		return []byte{}, err
	}

	return b, err
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

func (r *consulRepository) Add(args []string) error {
	err := consul.AddProfile(args)

	return err
}

func (r *consulRepository) IsConfigured() bool {
	cfg := config.Get()

	if cfg.ConsulAddress != "" && cfg.ConsulToken != "" || cfg.ConsulTokenFile != "" {
		return true
	}

	return false
}

func (r *consulRepository) GetName() string {
	return r.Name
}

func (r *consulRepository) List() ([]string, error) {
	list, err := consul.ListProfiles()

	return list, err
}

func (r *consulRepository) Get(name string) (profile.Profile, error) {
	p, err := consul.GetProfile(name)

	return p, err
}

func (r *consulRepository) Remove(args []string) error {
	return consul.RemoveProfile(args)
}

func (r *consulRepository) Save(p profile.Profile) error {
	return consul.SaveProfile(p)
}

func (r *consulRepository) Show(name string) ([]string, error) {
	return consul.ShowProfile(name)
}
