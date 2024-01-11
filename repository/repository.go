package repository

import "github.com/julienlevasseur/profiler/profile"

type IRepository interface {
	Save(p profile.Profile) error
	Get() (profile.Profile, error)
	List() []string
}
