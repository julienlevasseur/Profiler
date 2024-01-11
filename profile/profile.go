package profile

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Profile struct {
	Name     string `json:"name"`
	KVs      []KV   `json:"kvs"`
	Provider string `json:"provider"`
}

type IProfile interface {
	Add(provider string, kvs []KV) error
	Remove() error
	Show() error
	Use() error
}

func New() Profile {
	return Profile{}
}

func (p Profile) Add(provider string, kvs []KV) error {
	return nil
}

func (p Profile) Remove() error {
	return nil
}

func (p Profile) Show() error {
	return nil
}

func (p Profile) Use() error {
	return nil
}
