# Profiler

![Tests status](https://github.com/julienlevasseur/Profiler/workflows/Test/badge.svg)
![GoReleaser](https://github.com/julienlevasseur/Profiler/workflows/goreleaser/badge.svg)

Profiler is a CLI tool to organize your environment variables as profiles.
It is not intended to manage software configuration (look at [viper](github.com/spf13/viper) for this), it focus on developer environment to set temporary specific environments.

> **NOTE**
> Due to the scope of exported variable in shells, Profiler needs to spawn a new shell instance
> when using a profile.
> This has the advantage to allow you to simply exit the spawned instance to disable the profile
> activation, but, it limits the access to the shell history. You can refer to your shell
> documentation to find the best way to share history between your shell levels.

## Usage

One of the reasons Profiler has been created is to easily switch between cloud providers
or Hashicorp stack environments/clusters.
If you're a Kubernetes user, you can see Profiler as "universal" [kubens](https://github.com/ahmetb/kubectx).
But is also does more !

Profiler allow you to regroup environement variable in profiles but it will also reads local files (.env.yml, .envrc, .env) to expand your profile based on your local directory (as [direnv](https://github.com/direnv/direnv) (see below)).

So basically, Profiler allows you to:

* Load cloud provider/stacks credentials/configs to switch between accounts/environments
* Switch accross Kubernetes namespaces
* Create per-project isolated development environments
* Load secrets/configs for deployment
* Share profiles accross teams (with SSM & Consul profiles remote storage)

### The profile file

A profile file is a simple YAML file that represent the vars you need for this profile :

```yaml
profile_name: aws_dev
shell: /usr/bin/zsh
AWS_ACCESS_KEY_ID: xxxxxxxxXXXXXxxxxxxxx
AWS_SECRET_ACCESS_KEY: xxxxxxxxXXXXXxxxxxxxx
AWS_DEFAULT_REGION: us-east-1
CONSUL_HTTP_TOKEN: xxxxxxxxXXXXXxxxxxxxx
TF_VAR_CONSUL_HTTP_TOKEN: xxxxxxxxXXXXXxxxxxxxx
```

These profile files have to be located in `profilesFolder` and named like `.FooBar.yml`.

Profiler support external sources for profiles.
This is useful if you share environment variable in your team or if you want to use a specific set of of env vars on multiple computers.

### The SSM profile

A profile stored in SSM will be split in multiple parameters:

- the profile name (created by default when the profile is created with `profiler ssm add`)
- one parameter per variable contained in the profile

Example:

|  Name | Type  |  Value | Tags |
|-------|-------|--------|------|
| /profiler/ProfileName/profile_name | String | $ProfileName | `profiler: true` |
| /profiler/ProfileName/Key | String | $Value | `profiler: true` |

### The Consul profile

A profile stored in Consul will be in a `profiler` KV folder with a Key per
profile and YAML Value.

Example:

Key: `/profiler/example_consul_profile`
Value:

```yaml
profile_name: test_consul
FOO: BAR
```

### The config file

The config file is located by default in `~/.profiler_cfg.yml`.

You can override this value by setting the `PROFILER_CFG` env var:

```bash
export PROFILER_CFG="/my/prefered/path"
```

> **Note**
> 
> If no configuration file is found, a default configuration file will be created poiting the `profilesFolder` attribute to `$HOME/.profiles`.

#### Configuration options

##### shell

The profile file may contain the `shell` attribute. This attribute will never be exported as env variable. Its used to specify in which shell you want to spawn your profile.

You can also set a `shell` in the configuration file. This can be helpful if you want to use a different shell than your current one when you use a profile.

> **Note**
>
> The default shell is the current shell.

> **Note**
>
> Profiler has been tested only with bash and zsh.
> Any contribution to validate other shells are welcome.

#### preserveProfile



This option allows you to decide if you want to preserve the `.profiler` file where you have used a profile or remove it once the profile is exported.
With this option you can decide if you prefer to keep the `.profiler` files, so you can re-use a profile later (adding it to your global `.gitignore` is strongly recommended) or simply decide that you want to generate it every time.

Reusing an already exported profile from a directory is done as simply as: `profiler use`.

#### k8sSwitchNamespace

This option allows you to toggle the auto Kubernetes namespace switch.
When enabled (by default), if the `K8S_NAMESPACE` is set in a profile, Profiler will switch to this namespace using the `kubectl` command.

##### Example of a configuration file

```yml
profilesFolder: /My/Home/.profiles
shell: bash               # Optional (current shell by default)
preserveProfile: False    # Optional (true by default)
k8sSwitchNamespace: False # Optional (true by default)
```

### The profile definition

#### Via the profiler command

The `profiler add` command allow you to create profiles and add variable to them:

```bash
profiler add MyProfile
```

will create a profile that only export its name (`profile_name` var)

```bash
profiler add MyProfile Key Value
```

will add the Key=Value env var to MyProfile (if the profile does not exists it will be created).

#### Manually

If you want to set env vars or profiles, you can create as many profile files as you want into the `profilesFolder`.

> **Note**
>
> **Be careful !** The profiles files have to be hidden (so prefixed by a `.`)
>
> e.g:
>
> ```bash
>/home/$USER/.profiles/
>└── .example-aws-us-east-1.yml
>```

### The yaml env file

Inspired by [direnv](https://direnv.net/), this feature allow you to create a `.env.yml` file into a folder that will be sourced when you call this tool inside this folder.

When you call `profiler` without arguments, the program will look on the current working directory for a file called `.env.yml` and source it if found.

When you call `profiler use ${profile}`, the program will also look for `.env/yml` and append its content to the specified profile.

This feature is usefull if you want to have immutable set a vars for a cloud provider (the profile) but specific vars for a specific project, repo, branch, etc
For example, you can want to set your AWS keys in the `my_aws_account` profile, but having different `AWS_DEFAULT_REGION` or dedicated Terraform vars in several projects.
To accomplish that, just create a `/etc/profiler/.my_aws_account.yml` profile file and in your repos, a .env.yml per repo with the dedicated set of vars inside.

Example of a `.env.yml` file:

```yaml
AWS_DEFAULT_REGION: us-east-2
TF_VAR_project_name: my_awesome_project
```

> **Note:** The `.env.yml` file override the double env vars that it can find in the profile.

### .envrc support

To go deeper into the `direnv` inspiration/compatibility, profiler now support `.envrc` files.

Example of a `.envrc` file:

```bash
export FOO=bar
```

### .env support

To complete the non `yaml` files support, the 1.3.0 version has seen the addition of `*.env` files support.

Example of `.env` file:

```bash
export FOO=bar
```

### Remote storage

From version 3.4.0, it's now possible to store profiles remotely.
The two supported provider (so far) are:

#### AWS SSM

##### Credentials

To access profiles stored in the AWS SSM Parameters Store, Profiler requires AWS
credentials.
To configure the AWS credentials, you can refer to the AWS SDK documentation: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials

#### Consul

To access profiles stored in the Consul KV Store, Consul credentials must be provided via profiler_cfg.

Supported Consul configuration options:

|  Name | Value example |
|-------|-------|
| consulAddress | http://W.X.Y.Z:8500 |
| consulToken (optional) | 3d4a9009-eef0-4444-92c4-322e6a853385 |
| consulTokenFile (optional) | /home/user/.consul_token |

> **Note:**
> 
> The consulToken and consulTokenFile configurations are optional. You can choose to use one or the other. And of course, if your Consul instance does not use ACLs, they're not required.

### The profiler command

* `profiler` - Search for env files and source them if they exists.
* `profiler` `list` - list the available profiles.
* `profiler` `add` `${profile_name}` `${key}` `${value}` - create the given profile and or add the given env var to the profile.
* `profiler` `remove` `${profile_name}` `${key}` - remove the given profile or the variable matching the $key from the given profile.
* `profiler` `use` `${profile_name}` - Actually use the specified profile, if no profile name specified, search for .profiler file and env files and export the generated profile from them.
* `profiler` `aws_mfa` `${MFA Token}` - Need an already exported AWS profile. Authenticate to AWS with MFA Token. (Surcharge the current profile with Secret Key, Access Key Id and Token from MFA auth.)
* `profiler` `ssm` - Interact with remote profiles stored in AWS SSM.
* `profiler` `help` - Display the help message.

## Tips

> **Note**
> It's strongly recommend, for convenience, for you to add `.profiler` to your global `.gitignore`.

## Use case example

Here's a use case example with several profiles: cloud providers and stacks env vars.

- Work AWS
- Work OpenStack
- Personal AWS
- Personal Nomad/Consul Cluster
...

For conveniance the `profile_name` var can be used in the PS1 to display which profile is currently in use :

![usage_demo.png](https://github.com/julienlevasseur/profiler/raw/master/images/usage_demo.png)

> **Note**
> [Powerline-shell](https://github.com/b-ryan/powerline-shell) is the tool used in the example to customize my PS1.
> If you use it too and want take advantage of it to display your *env profile* you can find the segment [here](https://github.com/julienlevasseur/powerline-shell/blob/master/powerline_shell/segments/cloud_profile.py) and a support of Terraform workspaces [here](https://github.com/julienlevasseur/powerline-shell/blob/master/powerline_shell/segments/terraform_workspace.py).

Project's specific vars can be set via the `.env.yml` file. For instance, here's an example of an `.env.yml` file inside the folder for provisionning a Kubernetes cluster with [KOPS](https://github.com/kubernetes/kops):

```yaml
KOPS_STATE_STORE: s3://my-project-kubernetes-aws-state
KUBE_CTX_CLUSTER: my-project-k8s.my.domain.me
KUBE_USER: admin
KUBE_PASSWORD: ***************
```

### AWS MFA

![aws_mfa_demo.png](https://github.com/julienlevasseur/profiler/raw/master/images/aws_mfa_demo.png)

> **Note**
> The AWS MFA feature rely on the `AWS_MFA_USERNAME` env var. Be sure to have it set in your profile prior to use aws_mfa option.

## Concept summary

![concept_summary.png](https://github.com/julienlevasseur/profiler/raw/master/images/concept_summary.png)
