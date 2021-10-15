package ssm

import (
	"strings"

	awsssm "github.com/PaddleHQ/go-aws-ssm"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/viper"
)

func Get() (string, error) {

	pmstore, err := awsssm.NewParameterStore(
		&aws.Config{

			Region: aws.String(viper.GetString("ssmRegion")),
		},
	)
	if err != nil {
		return "", err
	}
	//Requesting the base path
	params, err := pmstore.GetAllParametersByPath("/profiler/test_ssm_profile/", true)
	if err != nil {
		return "", err
	}

	//And getting a specific value
	return params.GetValueByName("FOO"), nil
}

type Parameter struct {
	Name  string
	Value string
}

func getParameters(path string) ([]*ssm.Parameter, error) {
	mySession := session.Must(session.NewSession())

	// Create a SSM client from just a session.
	svc := ssm.New(
		mySession, aws.NewConfig().WithRegion(
			viper.GetString("ssmRegion"),
		),
	)

	var input = &ssm.GetParametersByPathInput{}
	input.SetPath(path)
	input.SetRecursive(true)

	getParametersByPathOutput, err := svc.GetParametersByPath(input)
	if err != nil {
		panic(err)
	}

	return getParametersByPathOutput.Parameters, nil
}

func profileAlreadyListed(profiles []string, searchedProfile string) bool {
	for _, i := range profiles {
		if i == searchedProfile {
			return true
		}
	}

	return false
}

/*ListProfiles return the name of the SSM profiles as []string*/
func ListProfiles() ([]string, error) {
	params, err := getParameters("/profiler/")
	if err != nil {
		return []string{}, err
	}

	var ssmProfiles []string
	for _, p := range params {
		profileName := strings.Split(*p.Name, "/")[2]
		/** With SSM folders management, a profile would be listed as many time as vars
		it contains. Since ListProfile is meant to list once every found profiles, we
		check here if the profile has already been listed or not:*/
		if len(ssmProfiles) == 0 || !profileAlreadyListed(ssmProfiles, profileName) {
			ssmProfiles = append(
				ssmProfiles,
				profileName,
			)
		}
	}

	return ssmProfiles, nil
}

func ShowProfile(profileName string) ([]string, error) {
	params, err := getParameters("/profiler/" + profileName)
	if err != nil {
		return []string{}, err
	}

	var vars []string
	for _, p := range params {
		varName := strings.Split(*p.Name, "/")[3]
		vars = append(vars, varName)
	}

	return vars, nil
}

/*GetProfile retrive the given profile from AWS SSM*/
func GetProfile(profileName string) (map[string]string, error) {
	params, err := getParameters("/profiler/" + profileName)
	if err != nil {
		return map[string]string{}, err
	}

	vars := make(map[string]string)

	for _, p := range params {
		vars[strings.Split(*p.Name, "/")[3]] = *p.Value
	}

	return vars, nil
}
