package repository

import (
	"github.com/julienlevasseur/profiler/pkg/profile"
)

type IRepository interface {
	Add(args []string) error
	Get(name string) (profile.Profile, error)
	GetName() string
	IsConfigured() bool
	List() ([]string, error)
	Remove([]string) error
	Save(p profile.Profile) error
	Show(name string) ([]string, error)
}
