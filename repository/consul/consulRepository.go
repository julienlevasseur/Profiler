package consul

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/consul"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

type consulRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
}

func NewConsulRepository() *consulRepository {
	return &consulRepository{
		Name: "consul",
	}
}

// probeTimeout bounds the reachability check IsConfigured makes. It is short
// on purpose: `profiler list` runs it on every invocation.
const probeTimeout = 2 * time.Second

func newConsulAPIClient() (*api.Client, error) {
	cfg := config.Get()

	client, err := api.NewClient(&api.Config{
		Address:    cfg.ConsulAddress,
		TokenFile:  cfg.ConsulTokenFile,
		Token:      cfg.ConsulToken,
		HttpClient: &http.Client{Timeout: probeTimeout},
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

// IsConfigured reports whether there is a Consul this profiler can actually
// talk to. Consul needs no auth in dev mode, so both credentials are optional,
// and consulAddress carries a localhost default -- which leaves nothing in the
// configuration alone that separates "Consul is set up" from "Consul is not".
// The only honest answer is to ask, so this makes one short, bounded call.
//
// /v1/status/leader requires no ACL token, so the probe reports reachability
// whether or not Consul has ACLs enabled.
//
// Answering false is what keeps `profiler list` usable on a machine with no
// Consul: cmd/list skips a repository that is not configured, but exits
// non-zero if one that claimed to be configured then fails to list.
func (r *consulRepository) IsConfigured() bool {
	client, err := newConsulAPIClient()
	if err != nil {
		return false
	}

	_, err = client.Status().Leader()

	return err == nil
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
