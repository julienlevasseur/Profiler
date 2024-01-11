package repository

import "github.com/julienlevasseur/profiler/pkg/profile"

type IRepository interface {
	Save(p profile.Profile) error
	Get() (profile.Profile, error)
	List() []string
}
