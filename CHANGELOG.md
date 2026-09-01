# 4.0.2

- `profiler list` asks every repository at once instead of one after another, so it waits for the slowest backend rather than for the sum of all of them. The output keeps `supportedRepositories` order whichever backend answers first, and a failure now names the repository it came from.
- SSM calls are bounded: five seconds a request, fifteen for a whole paginated listing, and two retries at most. `profiler list` could previously hang on a network that drops packets rather than refusing them.
- `config.Get` is safe to call from several goroutines at once.

# 4.0.1

- fail fast on SSM IsConfigured if no AWS env var are set
- update pipeline actions

# 4.0.0

- The SSM repository is implemented rather than stubbed: `profiler list` shows SSM profiles when AWS credentials resolve, `profiler ssm use` sets an environment from an SSM profile, and every `profiler ssm` subcommand goes through the repository interface.
- Add the optional `ssmEndpoint` configuration, to reach an SSM that is not AWS's own (localstack, a VPC endpoint, a FIPS endpoint).
- SSM profile listings are paginated, so more than ten parameters are no longer silently truncated.
- `profiler remove` and `profiler show` go through the local repository rather than calling `pkg/profile` directly, and the local repository's `Remove`, `Save` and `Show` are implemented rather than stubbed.
- `profiler show` lists a profile's variables in a stable order.
- `profiler remove` accepts several variables at once, reports a profile that does not exist instead of failing silently, and refuses to remove `profile_name` on its own.
- Remove the unused top-level `profile` package. `pkg/profile` is the single `Profile` type; `Design.md` records why.
- Remove dead scaffolding left over from the redesign: `repository/local/localProfile.go`, the unused `repository.Repository` struct and its `New()`, and the never-implemented `ComposeProfile` and its two helpers.
- `profiler use` stacks profiles instead of replacing them: a profile applied while another is in use exports both profiles' variables, the newly applied one winning on any variable they share. `profiler status` reports the whole stack (`A+B`), and leaving the shell `use` spawned unstacks. `Design.md` records the decision.
- `profiler use` exports `profile_keys`, naming the variables the profile in use owns. It is what lets the next `use` stack onto them, and it makes the `.profiler` file a full record of the environment in use rather than only of the profile applied last.

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
