package ssm

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

// profilesPath is the Parameter Store hierarchy profiler owns. Every profile
// is a folder under it, and every variable a parameter in that folder:
//
//	/profiler/<profile>/profile_name
//	/profiler/<profile>/<KEY>
const profilesPath = "/profiler"

// probeTimeout bounds the credential lookup IsConfigured makes. It is short on
// purpose: `profiler list` runs it on every invocation, and the credential
// chain ends at the EC2 metadata endpoint, which is unreachable rather than
// absent on a machine that is not an EC2 instance.
const probeTimeout = 2 * time.Second

// awsConfig is the SDK configuration both the SSM client and the credential
// probe are built from, so the two never disagree about which credentials the
// calls would be made with.
//
// Credentials are left unset unless profiler was configured with a pair of
// its own: unset is what lets the SDK's own chain answer -- the AWS_*
// environment variables, the shared credentials file, an instance role.
func awsConfig() *aws.Config {
	cfg := config.Get()

	awsCfg := aws.NewConfig().WithRegion(cfg.SSMRegion)

	// An empty ssmEndpoint means the AWS endpoint for ssmRegion. It is set
	// to reach an SSM that is not AWS's own -- localstack, a VPC endpoint,
	// a FIPS endpoint.
	if cfg.SSMEndpoint != "" {
		awsCfg = awsCfg.WithEndpoint(cfg.SSMEndpoint)
	}

	if cfg.AWS_ACCESS_KEY_ID != "" && cfg.AWS_SECRET_ACCESS_KEY != "" {
		awsCfg = awsCfg.WithCredentials(credentials.NewStaticCredentials(
			cfg.AWS_ACCESS_KEY_ID,
			cfg.AWS_SECRET_ACCESS_KEY,
			cfg.AWS_SESSION_TOKEN,
		))
	}

	return awsCfg
}

func newSSMService() (*ssm.SSM, error) {
	sess, err := session.NewSession(awsConfig())
	if err != nil {
		return nil, err
	}

	return ssm.New(sess), nil
}

// profilePath is the folder holding one profile's parameters.
func profilePath(profileName string) string {
	return fmt.Sprintf("%s/%s", profilesPath, profileName)
}

// parameterPath is the full name of one variable of one profile.
func parameterPath(profileName, key string) string {
	return fmt.Sprintf("%s/%s", profilePath(profileName), key)
}

// splitParameterPath breaks a parameter name into the profile it belongs to
// and the variable it holds. Either can come back empty: listing a path
// recursively also returns the folder parameters themselves, which name a
// profile but no variable.
func splitParameterPath(name string) (profileName, key string) {
	parts := strings.Split(
		strings.TrimPrefix(name, profilesPath+"/"),
		"/",
	)

	profileName = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}

	return profileName, key
}

// getParameters returns every parameter under path. GetParametersByPath is
// paginated -- ten parameters a page by default -- so this walks every page:
// a profile set larger than one page would otherwise be silently truncated.
func getParameters(path string) ([]*ssm.Parameter, error) {
	svc, err := newSSMService()
	if err != nil {
		return nil, err
	}

	var params []*ssm.Parameter

	err = svc.GetParametersByPathPages(
		&ssm.GetParametersByPathInput{
			Path:      aws.String(path),
			Recursive: aws.Bool(true),
		},
		func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
			params = append(params, page.Parameters...)
			return !lastPage
		},
	)
	if err != nil {
		return nil, err
	}

	return params, nil
}

// putParameter writes one variable of one profile. SSM refuses Tags and
// Overwrite in the same call, so this tags on creation and overwrites on
// update: a parameter profiler already owns is already tagged.
func putParameter(name, value string) error {
	svc, err := newSSMService()
	if err != nil {
		return err
	}

	input := &ssm.PutParameterInput{
		Name:  aws.String(name),
		Type:  aws.String(ssm.ParameterTypeString),
		Value: aws.String(value),
		Tier:  aws.String(config.Get().SSMParameterTier),
		Tags: []*ssm.Tag{{
			Key:   aws.String("profiler"),
			Value: aws.String("true"),
		}},
	}

	_, err = svc.PutParameter(input)

	var awsErr awserr.Error
	if errors.As(err, &awsErr) && awsErr.Code() == ssm.ErrCodeParameterAlreadyExists {
		input.Tags = nil
		input.Overwrite = aws.Bool(true)
		_, err = svc.PutParameter(input)
	}

	return err
}

func deleteParameter(name string) error {
	svc, err := newSSMService()
	if err != nil {
		return err
	}

	_, err = svc.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(name),
	})

	return err
}

// IsConfigured reports whether this machine has what it takes to reach the SSM
// Parameter Store: a region, and credentials the AWS chain can resolve -- from
// the environment, the shared credentials file, or an instance role.
//
// It resolves the credentials rather than calling SSM, so the answer costs
// nothing on the common paths, and it bounds the one path that can reach out
// (the metadata endpoint) so `profiler list` cannot hang on a machine with no
// AWS setup at all. Answering false is what keeps `profiler list` usable
// there: cmd/list skips a repository that is not configured.
func IsConfigured() bool {
	sess, err := session.NewSession(
		awsConfig().
			WithHTTPClient(&http.Client{Timeout: probeTimeout}).
			WithMaxRetries(0),
	)
	if err != nil {
		return false
	}

	// SSM cannot be reached without a region, and every call would fail
	// with MissingRegion. NewSession fills the region in from AWS_REGION
	// when ssmRegion is unset, so this reads the resolved session rather
	// than the configuration.
	if aws.StringValue(sess.Config.Region) == "" {
		return false
	}

	_, err = sess.Config.Credentials.Get()

	return err == nil
}

/*ProfileExist return a boolean representation of the given profile existence*/
func ProfileExist(profileName string) (bool, error) {
	profiles, err := ListProfiles()
	if err != nil {
		return false, err
	}

	return slices.Contains(profiles, profileName), nil
}

// profileNames is the profile half of a listing of the whole /profiler path.
func profileNames(params []*ssm.Parameter) []string {
	var profiles []string

	for _, p := range params {
		profileName, _ := splitParameterPath(aws.StringValue(p.Name))

		// A profile holding several variables is returned once per
		// variable, and a listing names each profile once:
		if profileName == "" || slices.Contains(profiles, profileName) {
			continue
		}

		profiles = append(profiles, profileName)
	}

	return profiles
}

/*ListProfiles return the name of the SSM profiles as []string*/
func ListProfiles() ([]string, error) {
	params, err := getParameters(profilesPath + "/")
	if err != nil {
		return []string{}, err
	}

	return profileNames(params), nil
}

// parametersToProfile turns a listing of one profile's folder into the profile
// it holds.
func parametersToProfile(profileName string, params []*ssm.Parameter) profile.Profile {
	var kvs []profile.KV

	for _, p := range params {
		_, key := splitParameterPath(aws.StringValue(p.Name))
		// The folder parameter names the profile but holds no variable:
		if key == "" {
			continue
		}

		kvs = append(kvs, profile.KV{
			Key:   key,
			Value: aws.StringValue(p.Value),
		})
	}

	return profile.Profile{
		Name: profileName,
		KVs:  kvs,
	}
}

/*GetProfile retrieve the given profile from AWS SSM*/
func GetProfile(profileName string) (profile.Profile, error) {
	params, err := getParameters(profilePath(profileName))
	if err != nil {
		return profile.Profile{}, err
	}

	return parametersToProfile(profileName, params), nil
}

/*ShowProfile list the Env vars stored in a profile*/
func ShowProfile(profileName string) ([]string, error) {
	p, err := GetProfile(profileName)
	if err != nil {
		return []string{}, err
	}

	var keys []string
	for _, kv := range p.KVs {
		keys = append(keys, kv.Key)
	}

	return keys, nil
}

// AddProfile create the given profile, and add the given variables to it. args
// is the profile name followed by an even number of key/value arguments, which
// is the same shape the consul and vault repositories take.
func AddProfile(args []string) error {
	if len(args) == 0 {
		return errors.New("Please provide a profile name")
	}

	profileName, kvs := args[0], args[1:]

	// A key with no value, or a value with no key:
	if len(kvs)%2 != 0 {
		return errors.New("Please provide a value for the variable")
	}

	exists, err := ProfileExist(profileName)
	if err != nil {
		return err
	}

	if !exists {
		err = putParameter(
			parameterPath(profileName, "profile_name"),
			profileName,
		)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(kvs); i += 2 {
		err = putParameter(parameterPath(profileName, kvs[i]), kvs[i+1])
		if err != nil {
			return err
		}
	}

	return nil
}

/*SaveProfile write every variable of the given profile to AWS SSM*/
func SaveProfile(p profile.Profile) error {
	for _, kv := range p.KVs {
		if err := putParameter(parameterPath(p.Name, kv.Key), kv.Value); err != nil {
			return err
		}
	}

	return nil
}

// RemoveProfile delete the named variables from the given profile, or the
// whole profile when no variable is named.
func RemoveProfile(args []string) error {
	if len(args) == 0 {
		return errors.New("Please provide a profile name")
	}

	profileName, keys := args[0], args[1:]

	exists, err := ProfileExist(profileName)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("The provided Profile does not exist")
	}

	// No variable named: remove the profile, which in SSM means removing
	// every parameter under its folder.
	if len(keys) == 0 {
		keys, err = ShowProfile(profileName)
		if err != nil {
			return err
		}
	} else {
		// profile_name is what makes the profile a profile: it goes with
		// the whole profile, never on its own. Same rule as consul and
		// vault. Deleting from a copy, so args is not rewritten under
		// the caller.
		keys = slices.DeleteFunc(slices.Clone(keys), func(key string) bool {
			return key == "profile_name"
		})
	}

	for _, key := range keys {
		if err := deleteParameter(parameterPath(profileName, key)); err != nil {
			return err
		}
	}

	return nil
}
