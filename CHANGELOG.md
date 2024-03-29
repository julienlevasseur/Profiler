# 3.5.1

- Remove default consul address value from config to avoid error on `profiler list` if Consul is not used.

# 3.5.0

- Add Kubernetes Namespace switch support
- Rename profilerFolder to profilesFolder and create the folde by default
- Various fixes and improvements

# 3.4.8

- Update release go version & README

# 3.4.7

- Extends list command

# 3.4.6

- Bump github.com/aws/aws-sdk-go from 1.33.0 to 1.34.0

# 3.4.5

- Add Go version 1.17 and 1.18 in the test pipeline
- Bump github.com/aws/aws-sdk-go from 1.25.41 to 1.33.0
- Linting & typos in README

# 3.4.4

- Security fix:

  gopkg.in/yaml.v3 Version< 3.0.0 | Upgrade to~> 3.0.0
CVE-2022-28948 Moderate severity

# 3.4.3

- Update dependecies

# 3.4.2

- Add support for .yaml files (in addition to .yml) for local profiles.

# 3.4.1

- Add Consul remote profiles storage support.

# 3.4.0

- Add `add` command to append variables to profiles or creating empty new profiles via the `profiler` CLI.
- Add remote profiles support.
- Add AWS SSM Parameter Store as remote profiles repository.

# 3.3.0

- Add a `show` argument to display exported profiles variable names.

# 3.2.6

- Set default `profileFolder` to `~/.profiles` rather than `~/.profiler` to avid conflict with the `.profiler` file in home dir.

# 3.2.5

- Removes helpers pkg (there should be no helpers package:
    "[A little] duplication is far cheaper than the wrong abstraction."
  )
- Improve tests (for MacOS target)

# 3.2.4

- CI improvement (Release only on new tags)

# 3.2.3

- README Improvement & images cleanup

# 3.2.2

- Removes FreeBSD support

# 3.2.1

- Migrate to Github
- Improve testing
- Adds Goreleaser support
- Migrate pipeline to Github Actions

# 3.2.0

- Implement `.profiler` file preservation option

# 3.1.0

- Implement Cobra framework as CLI library

# 3.0.1

- Add a test for setConfigFile function
- Add `src/` & `bin/` to gitignore
- Add go.mod

# 3.0.0

- Add a configFile that support:
    - Path to the profile files (`cloudProfileFolder`)
    - Alternate shell that apply to every profile that don't provide a shell

- Set the default shell to $SHELL instead of bash

- Refactor tests with Ginkgo

- Simplify the pipeline

# 2.3.0

- Add support of tier shell (not only bash), via the `shell` attribute.

# 2.2.3

- Add a statement if no profile is provided to 'use' option, then print help.

# 2.2.1

- Correct a big with listFiles function (25bbccbd)

# 2.2.0

- Refactor the pipeline (c4dd0873, 6c1c483f, 7e7fb5fe, 1f3e9e8f)

# 2.1.0

- Implement AWS MFA support (de7af637)

# 2.0.0

- Refactor main function and other functions / clean the code (c55f16e3)

# 1.3.0

- Implement support of *.env files (a9495c1f)
- Various refactoring and linting adjustments (039dc120, 892e79da, 319a70b1, c0a0dda6,  9ff786fa,  f590b62c)

# 1.2.0

- Correct parseEnvrc which was adding a " to .cloud_profile values (e76d9109)
- Reorganise import according to gofmt recommendation (0c186101)
- Implement usage of existing .cloud_profile (169c0a23)
