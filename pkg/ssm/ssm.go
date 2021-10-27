package ssm

import (
	"fmt"
	"os"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return getParametersByPathOutput.Parameters, nil
}

/*ProfileExist return a boolean representation of the given profile existence*/
func ProfileExist(profileName string) (bool, error) {
	profiles, err := ListProfiles()
	if err != nil {
		return false, err
	}

	for _, profile := range profiles {
		if profile == profileName {
			return true, nil
		}
	}

	return false, nil
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

/*ShowProfile list the Env vars stored in a profile*/
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

/*AddParameter is used to create either Profile or Env var in SSM*/
func AddParameter(paramName string, paramValue string) error {
	mySession := session.Must(session.NewSession())

	// Create a SSM client from just a session.
	svc := ssm.New(
		mySession, aws.NewConfig().WithRegion(
			viper.GetString("ssmRegion"),
		),
	)

	var tags []*ssm.Tag
	tag := &ssm.Tag{
		Key:   aws.String("profiler"),
		Value: aws.String("true"),
	}
	tags = append(tags, tag)

	/* If only paramName is provided, Profiler assumes that it needs to create
	a profile and not an Env var*/
	if paramValue == "" {
		paramName = paramName + "/"
	}

	var input = &ssm.PutParameterInput{}
	input.SetName("/profiler/" + paramName)
	input.SetType("String")
	input.SetTags(tags)
	input.SetTier(viper.GetString("ssmParameterTier"))
	input.SetValue(paramValue)

	_, err := svc.PutParameter(input)
	if err != nil {
		return err
	}

	return nil
}

/*RemoveParameter is used to delete a Profile or Env var from SSM*/
func RemoveParameter(paramName string) error {
	mySession := session.Must(session.NewSession())

	// Create a SSM client from just a session.
	svc := ssm.New(
		mySession, aws.NewConfig().WithRegion(
			viper.GetString("ssmRegion"),
		),
	)

	var input = &ssm.DeleteParameterInput{}
	input.SetName(paramName)

	_, err := svc.DeleteParameter(input)
	if err != nil {
		return err
	}

	return nil
}
