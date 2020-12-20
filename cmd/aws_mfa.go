package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/julienlevasseur/profiler/pkg/profile"
)

var awsMFACmd = &cobra.Command{
	Use:   "aws_mfa",
	Short: "aws_mfa [mfa token code]",
	Long: `Override current profile with credentials obtained via MFA
	connection (need an already used profile to get MFA details from IAM.)`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] == "help" {
			cmd.Help()
			os.Exit(0)
		}

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
		})

		if err != nil {
			panic(err)
		}

		// Create a IAM service client:
		svc := iam.New(sess)

		mfaDevices, err := svc.ListMFADevices(
			&iam.ListMFADevicesInput{
				UserName: aws.String(os.Getenv("AWS_MFA_USERNAME")),
			},
		)

		if err != nil {
			panic(err)
		}

		var mfaDeviceSn string
		for _, i := range mfaDevices.MFADevices {
			mfaDeviceSn = aws.StringValue(i.SerialNumber)
			break
		}

		// Create a STS service client"
		stsSvc := sts.New(sess)
		awsCreds, err := stsSvc.GetSessionToken(
			&sts.GetSessionTokenInput{
				SerialNumber: aws.String(mfaDeviceSn),
				TokenCode:    aws.String(args[0]),
			},
		)

		envVars := make(map[string]string)
		envVars["profile_name"] = fmt.Sprintf(
			"%s-MFA",
			os.Getenv("profile_name"),
		)
		envVars["AWS_ACCESS_KEY_ID"] = aws.StringValue(
			awsCreds.Credentials.AccessKeyId,
		)
		envVars["AWS_SECRET_ACCESS_KEY"] = aws.StringValue(
			awsCreds.Credentials.SecretAccessKey,
		)
		envVars["AWS_SESSION_TOKEN"] = aws.StringValue(
			awsCreds.Credentials.SessionToken,
		)

		profile.SetEnvironment(envVars)
	},
}

func init() {
	RootCmd.AddCommand(awsMFACmd)
}
