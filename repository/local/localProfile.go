package local

import (
	"github.com/julienlevasseur/profiler/pkg/profile"
)

type localProfile struct {
	Name       string       `json:"name"`
	KVs        []profile.KV `json:"kvs"`
	Repository string       `json:"repository"`
}

//type KeyValueMap map[string]string

func NewLocalProfile() *localProfile {
	return &localProfile{}
}

// func apendProfileNameToFile(filepath, profileName string) error {
// 	err := appendToFile(
// 		filepath,
// 		profileName,
// 		"profile_name",
// 		profileName,
// 	)
// 	return err
// }

// func (p *localProfile) Add(repository string, kvs []profile.KV) error {
// 	// 1. Does only a profile name was provided (create an empty profile) ?
// 	//    1.1 Does the profile already exists ?
// 	// 2. Does the KVs provided already exists in the profile ?

// 	// No KVs provided, let's create an empty profile (with just `profile_name`):
// 	if len(kvs) < 1 {
// 		apendProfileNameToFile(
// 			fmt.Sprintf(
// 				"%s/.%v.yml",
// 				viper.GetString("profileFolder"),
// 				&p.Name,
// 			),
// 			p.Name,
// 		)
// 	} else {
// 		// At least one KV as been provided, append them to the file:
// 		for _, kv := range p.KVs {
// 			profileNameExists, _, err := foundInfFile(
// 				fmt.Sprintf(
// 					"%s/.%v.yml",
// 					viper.GetString("profileFolder"),
// 					&p.Name,
// 				),
// 				"profile_name",
// 			)

// 			if err != nil {
// 				return err
// 			}

// 			// If profile_name is not present in the file, add it:
// 			if !profileNameExists {
// 				apendProfileNameToFile(
// 					fmt.Sprintf(
// 						"%s/.%v.yml",
// 						viper.GetString("profileFolder"),
// 						&p.Name,
// 					),
// 					p.Name,
// 				)
// 			}

// 			alreadyExist, _, err := foundInfFile(
// 				fmt.Sprintf(
// 					"%s/.%v.yml",
// 					viper.GetString("profileFolder"),
// 					&p.Name,
// 				),
// 				kv.Key,
// 			)
// 			if err != nil {
// 				return err
// 			}

// 			if alreadyExist {
// 				fmt.Fprintf(
// 					os.Stderr,
// 					fmt.Sprintf("The provided variable already exist in %s", p.Name),
// 					"\n",
// 				)
// 				os.Exit(1)
// 			}

// 			err = appendToFile(
// 				viper.GetString("profileFolder"),
// 				p.Name,
// 				kv.Key,
// 				kv.Value,
// 			)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return nil
// }

func (p *localProfile) Remove() error {
	return nil
}

func (p *localProfile) Show() error {
	// var vars []string

	// kvs, err := parseYaml(
	// 	fmt.Sprintf(
	// 		"%s/.%v.yml",
	// 		viper.GetString("profileFolder"),
	// 		&p.Name,
	// 	),
	// )
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
	// for k := range kvs {
	// 	vars = append(vars, k)
	// }

	// fmt.Printf("%s:\n", p.Name)
	// // Display each Profile's env var name:
	// for _, v := range vars {
	// 	fmt.Printf("- %s\n", v)
	// }

	return nil
}

func (p *localProfile) Use() error {
	return nil
}
