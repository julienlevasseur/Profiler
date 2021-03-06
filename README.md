# Profiler

![Tests status](https://github.com/julienlevasseur/Profiler/workflows/Test/badge.svg)
![GoReleaser](https://github.com/julienlevasseur/Profiler/workflows/goreleaser/badge.svg)

Profiler is simple tool that allow you to manage your environment variables via profiles and distributed files.

## Usage

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

Those profile files have to be located in `` and named like `.FooBar.yml`.

### The config file

The config file is located by default in `~/.profiler_cfg.yml`.
You can override this value by setting the `PROFILER_CFG` env var:

```bash
export PROFILER_CFG="/my/prefered/path"
```

> **Note**
> 
> If no configuration file is found, a default configuration file will be created poiting the `profilerFolder` attribute to `$HOME/.profiles`.

#### Configuration options

##### shell

The profile file may contain the `shell` attribute. This attribute will never be exported as env variable. Its used to specify in which shell you want to spawn your profile.

You can also set a `shell` in the configuration file. This can be helpful if you want to use a diferent shell than your current one when you use a profile.

> **Note**
>
> The default shell is the current shell.

> **Note**
>
> Profiler hsa been tested only with bash and zsh.
> Any contribution to validate other shells are welcomed.

##### preserveProfile

This option allow you to decide if you want to preserve the `.profiler` file where you have used a profile or remove it once the profile is exported.
With this option you can decide if you prefer to keep the `.profiler` files, so you can re-use a profile later (adding it to your global `.gitignore` is strongly recommended) or simply decide that you want to generate it every time.


###### Example of a configuration file

```yml
profilerFolder: /My/Home/.profiles
shell: bash               # Optional (current shell by default)
preserveProfile: true # Optional (true by default)
```

### The profile definition

If you want to set env vars or profiles, you can create as many profile files as you want into the `profilerFolder`.

> **Note**
>
> **Be carefull !** The profiles files have to be hidden (so prefixed by a `.`)
>
> e.g:
> ```bash
>/home/$USER/.profiles/
>└── .example-aws-us-east-1.yml
>```
> 

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

### The profiler command

* `profiler` - Search for env files and source them if they exists.
* `profiler` `list` - list the available profiles.
* `profiler` `use` - Search for .profiler file and env files and export the generated profile from them.
* `profiler` `use` `${profile_name}` - Actually use the specified profile.
* `profiler` `aws_mfa` `${MFA Token}` - Need an already exported AWS profile. Authenticate to AWS with MFA Token. (Surcharge the current profile with Secret Key, Access Key Id and Token from MFA auth.)
* `profiler` `help` - Display the help message.

## Tips

> **Note**
> I strongly recommand, for convenience, for you to add `.profiler` field to your global `.gitignore`.

## How I'm using it

Basically I like the idea to run a command that summerize all the env vars I need rather than sourcing a file in a folder that I will have to figure out the location almost every time I need it.

So, I define my cloud providers env vars per profile :

* Work AWS
* Work OpenStack
* Personnal AWS
...

In each of these, I have something like :

```yaml
profile_name: aws_dev
AWS_ACCESS_KEY_ID: 
AWS_SECRET_ACCESS_KEY: 
AWS_DEFAULT_REGION: us-east-1
CONSUL_HTTP_TOKEN: 
```
I'm using the `profile_name` var in my PS1 to display on my shell which profile is currently in use :

![usage_demo.png](https://github.com/julienlevasseur/profiler/raw/master/images/usage_demo.png)

> **Note**
> I use [Powerline-shell](https://github.com/b-ryan/powerline-shell) to customize my PS1.
> If you use it too and want take advantage of it to display you *env profile* you can find the segment [here](https://github.com/julienlevasseur/powerline-shell/blob/master/powerline_shell/segments/cloud_profile.py) and a support of Terraform workspaces [here](https://github.com/julienlevasseur/powerline-shell/blob/master/powerline_shell/segments/terraform_workspace.py).

And I set project's specific vars via the `.env.yml` file. For example, if I have a project that contain the code to provision a Kubernetes cluster with [KOPS](https://github.com/kubernetes/kops) I will set the `my_project/.env.yml` as :

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
