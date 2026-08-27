<div align="center">

<img src="images/logo.png" alt="Profiler" width="320">

# Profiler

**Save your shell environment as named profiles. Switch between them with one command.**

[![Tests status](https://github.com/julienlevasseur/Profiler/workflows/Test/badge.svg)](https://github.com/julienlevasseur/Profiler/actions)
[![GoReleaser](https://github.com/julienlevasseur/Profiler/workflows/goreleaser/badge.svg)](https://github.com/julienlevasseur/Profiler/actions)

[What is this?](#what-is-this) ·
[Install](#install) ·
[5-minute tour](#5-minute-tour) ·
[How it works](#how-it-works) ·
[Commands](#command-reference) ·
[Configuration](#configuration-reference)

</div>

---

## What is this?

> **In one sentence:** Profiler is a bookmark manager for your terminal's environment variables.

If you have never dealt with this problem, here is what it looks like.

Your shell carries a bag of **environment variables** — `AWS_ACCESS_KEY_ID`,
`KUBECONFIG`, `DATABASE_URL`, and so on. Command-line tools read that bag to
decide *which account*, *which cluster*, *which database* they talk to. Change
the bag and the same `terraform apply` hits a different cloud account.

So people end up doing this, several times a day:

```bash
# "let me switch to the production account…"
export AWS_ACCESS_KEY_ID=AKIA................
export AWS_SECRET_ACCESS_KEY=wJalr...........
export AWS_DEFAULT_REGION=eu-west-1
export CONSUL_HTTP_TOKEN=3d4a9009-eef0-4444-92c4-322e6a853385
export KUBECONFIG=~/.kube/prod.yaml
# …and then forget to unset half of them
```

Copy-pasted from a wiki. Or from a `notes.txt`. Or from a chunk of `.bashrc`
that is commented out and uncommented by hand. It is tedious, it is easy to get
wrong, and getting it wrong means running the right command against the **wrong
account**.

**Profiler replaces all of that with:**

```bash
$ profiler use prod
```

<div align="center">

```mermaid
flowchart LR
    subgraph BEFORE["❌ Without Profiler"]
        direction TB
        B1["Open a wiki page"] --> B2["Copy 6 export lines"]
        B2 --> B3["Paste into the shell"]
        B3 --> B4["Hope nothing is stale"]
        B4 --> B5["Forget to unset them"]
    end

    subgraph AFTER["✅ With Profiler"]
        direction TB
        A1["profiler use prod"] --> A2["Environment is set"]
        A2 --> A3["exit"]
        A3 --> A4["Environment is gone"]
    end

    BEFORE ~~~ AFTER
```

</div>

### Is Profiler for me?

| You... | Profiler helps? |
|---|---|
| Juggle several AWS / GCP / OpenStack accounts | ✅ Yes — one profile per account |
| Switch between Kubernetes clusters and namespaces | ✅ Yes — think "universal [kubectx](https://github.com/ahmetb/kubectx)" |
| Work on projects that each need their own variables | ✅ Yes — like [direnv](https://direnv.net/), and it reads direnv's files too |
| Want your team to share one set of environments | ✅ Yes — profiles can live in Consul, AWS SSM or Vault |
| Want to configure an application you are shipping | ❌ No — use [viper](https://github.com/spf13/viper) or your framework's config system |

Profiler is a **developer convenience tool for your own terminal**. It is not a
secrets manager and not an application configuration framework.

---

## Install

<details open>
<summary><b>Download a release binary</b> (recommended)</summary>

Grab the archive for your OS and architecture from the
[Releases page](https://github.com/julienlevasseur/Profiler/releases), then:

```bash
tar -xzf profiler_Linux_x86_64.tar.gz
sudo mv profiler /usr/local/bin/
profiler help
```

Linux and macOS builds are published for `amd64`, `arm64`, `arm` and `386`.

</details>

<details>
<summary><b>With Go</b></summary>

```bash
go install github.com/julienlevasseur/profiler@latest
```

</details>

<details>
<summary><b>From source</b></summary>

```bash
git clone https://github.com/julienlevasseur/Profiler.git
cd Profiler
make linux      # or: make darwin
./bin/profiler help
```

</details>

---

## 5-minute tour

### 1. Create a profile

```bash
$ profiler add aws_dev
$ profiler add aws_dev AWS_ACCESS_KEY_ID AKIAIOSFODNN7EXAMPLE
$ profiler add aws_dev AWS_DEFAULT_REGION us-east-1
```

That wrote a plain YAML file at `~/.profiles/.aws_dev.yml`:

```yaml
profile_name: aws_dev
AWS_ACCESS_KEY_ID: AKIAIOSFODNN7EXAMPLE
AWS_DEFAULT_REGION: us-east-1
```

You can equally well create that file in your editor — there is nothing magic
about it. (Note the leading dot in the **filename**: profile files must be
hidden files.)

### 2. See what you have

```bash
$ profiler list
aws_dev
aws_prod
personal
```

### 3. Use it

```bash
$ profiler use aws_dev
```

You are now in a shell where those variables are set. Check it:

```bash
$ echo $AWS_DEFAULT_REGION
us-east-1
$ profiler status
aws_dev
```

### 4. Stop using it

```bash
$ exit
```

That is it. `exit` puts you back exactly where you were, with the variables
gone. There is no "unset" command to remember, and no way to leave a stale
credential lying around in your shell.

<div align="center">
<img src="images/usage_demo.png" alt="profiler list, then profiler use, with the profile name in the prompt" width="700">
<br>
<em>Here the active profile is shown right in the prompt — see <a href="#show-the-active-profile-in-your-prompt">Tips</a>.</em>
</div>

---

## How it works

This is the one mechanism worth understanding, because everything else follows
from it.

### Why a new shell?

A program **cannot** change the environment of the shell that started it. That
is a hard rule of Unix processes: a child gets a *copy* of its parent's
environment, and changes to the copy die with the child. It is the reason
`direnv`, `nvm` and friends have to install a shell hook.

Profiler takes the other route: instead of trying to modify your shell, it
**starts a new one** with the environment already in place.

```mermaid
flowchart LR
    A["Your shell<br/><i>no profile</i>"] -->|"profiler use aws_dev"| B["A new shell<br/><i>AWS_* variables set</i>"]
    B -->|"exit"| A
```

That single design choice buys the two properties that define the tool:

| Consequence | Why it matters |
|---|---|
| 🎁 **`exit` is the "off switch"** | You can never forget to clean up. Leaving the shell *is* deactivating the profile. |
| 🧅 **Profiles nest** | Each `use` is another shell level, so profiles stack and unstack like layers. |
| ⚠️ **Shell history is per-level** | The new shell has its own history. See [Tips](#share-shell-history-across-levels) for the two-line fix. |

### The full picture

When you run `profiler use aws_dev`, this happens:

```mermaid
flowchart TD
    START(["profiler use aws_dev"]) --> FIND

    subgraph FIND["1 · Find the profile"]
        direction LR
        L["📁 Local file<br/>~/.profiles/.aws_dev.yml"]
        C["🌐 Consul KV"]
        S["🌐 AWS SSM"]
        V["🌐 Vault"]
    end

    FIND --> DIR

    subgraph DIR["2 · Add this directory's variables"]
        direction LR
        D1[".profiler"] --> D2["*.env"] --> D3[".env.yml"] --> D4[".envrc"]
    end

    DIR --> STACK["3 · Merge with any profile already in use"]
    STACK --> K8S{"K8S_NAMESPACE set?"}
    K8S -->|yes| KUBE["kubectl config set-context --current --namespace=…"]
    K8S -->|no| SPAWN
    KUBE --> SPAWN["4 · Write .profiler, then exec a new shell"]
    SPAWN --> DONE(["🎉 You are in the profile"])
```

## The three things Profiler reads

Profiler builds your environment out of up to three ingredients. You can use
just the first one and ignore the rest.

```mermaid
flowchart TB
    P["<b>1 · The profile</b><br/>~/.profiles/.aws_dev.yml<br/><i>the same everywhere you go</i>"]
    E["<b>2 · The directory's env files</b><br/>.env.yml / .envrc / *.env<br/><i>specific to this project</i>"]
    S["<b>3 · The profile already in use</b><br/><i>what you had before this command</i>"]
    P --> M["🍲 One merged environment"]
    E --> M
    S --> M
    M --> SH["A new shell"]
```

### 1 · Profiles — "which account am I?"

A profile is a YAML file: one key per environment variable.

```yaml
# ~/.profiles/.aws_dev.yml
profile_name: aws_dev                        # required, names the profile
shell: /usr/bin/zsh                          # optional, and never exported
AWS_ACCESS_KEY_ID: xxxxxxxxXXXXXxxxxxxxx
AWS_SECRET_ACCESS_KEY: xxxxxxxxXXXXXxxxxxxxx
AWS_DEFAULT_REGION: us-east-1
CONSUL_HTTP_TOKEN: xxxxxxxxXXXXXxxxxxxxx
TF_VAR_CONSUL_HTTP_TOKEN: xxxxxxxxXXXXXxxxxxxxx
```

Two keys are special and are **not** exported as variables:

- **`profile_name`** — the name Profiler and `profiler status` use.
- **`shell`** — which shell to spawn for this profile.

> [!IMPORTANT]
> Profile files live in `profilesFolder` (default `~/.profiles`) and their
> **filenames must start with a dot**:
>
> ```
> ~/.profiles/
> ├── .aws_dev.yml
> ├── .aws_prod.yml
> └── .personal.yml
> ```
>
> A file named `aws_dev.yml`, with no leading dot, is not seen.

### 2 · Directory env files — "which project am I in?"

Inspired by (and compatible with) [direnv](https://direnv.net/): drop a file in
a project directory and Profiler folds it in whenever you run `profiler` there.

<table>
<tr><th>File</th><th>Format</th><th>Example</th></tr>
<tr><td><code>.env.yml</code></td><td>YAML</td><td>

```yaml
AWS_DEFAULT_REGION: us-east-2
TF_VAR_project_name: my_awesome_project
```

</td></tr>
<tr><td><code>.envrc</code></td><td>shell</td><td>

```bash
export FOO=bar
```

</td></tr>
<tr><td><code>*.env</code></td><td>shell</td><td>

```bash
export FOO=bar
```

</td></tr>
</table>

This is what lets one profile hold your AWS credentials while each repository
adds its own region, Terraform variables or cluster name.

**Precedence — later wins:**

```mermaid
flowchart LR
    A["Profile<br/>~/.profiles/.aws_dev.yml"]
    B[".profiler"] --> C["*.env"] --> D[".env.yml"] --> E[".envrc"]
    A --> B
    E --> F["🏁 Final value"]
    style A fill:#1f6feb,color:#fff
    style F fill:#238636,color:#fff
```

So if the profile says `AWS_DEFAULT_REGION: us-east-1` and the directory's
`.env.yml` says `us-east-2`, you get **us-east-2**.

> [!TIP]
> Running bare `profiler` (no subcommand) loads *only* the directory files, no
> named profile. Handy in a project that needs no cloud credentials.

Some projects ship a template env file with fake values — `example.env`,
`sample.env`. Those are ignored by default; extend the list with
[`ignoredFiles`](#ignoredfiles).

### 3 · Stacking — layering one profile on another

`profiler use` **adds to** what you already have rather than replacing it.

```bash
$ profiler use aws_dev      # exports AWS_PROFILE, AWS_REGION
$ profiler use kube_dev     # exports KUBECONFIG, K8S_NAMESPACE, AWS_REGION
$ profiler status
aws_dev+kube_dev
```

`AWS_PROFILE` is still exported, `KUBECONFIG` is now exported too, and on
`AWS_REGION` — which both profiles set — the one you applied last wins.

```mermaid
flowchart TB
    subgraph L2["shell level 2 — aws_dev + kube_dev"]
        direction LR
        V2["AWS_PROFILE=dev · KUBECONFIG=… · K8S_NAMESPACE=… · <b>AWS_REGION=kube_dev's</b>"]
    end
    subgraph L1["shell level 1 — aws_dev"]
        direction LR
        V1["AWS_PROFILE=dev · AWS_REGION=aws_dev's"]
    end
    subgraph L0["shell level 0 — no profile"]
        direction LR
        V0["your normal environment"]
    end
    L0 -->|"profiler use aws_dev"| L1
    L1 -->|"profiler use kube_dev"| L2
    L2 -.->|"exit"| L1
    L1 -.->|"exit"| L0
```

**To unstack, leave the shell:**

```bash
$ exit    # back to aws_dev
$ exit    # back to no profile
```

Stacking works across backends too — a local profile stacks onto a Consul or
SSM one just the same.

<details>
<summary>How Profiler keeps track (the <code>profile_keys</code> variable)</summary>

Profiler exports one variable of its own, `profile_keys`, listing the variables
the current profile owns. It is what lets the next `profiler use` tell your
profile's variables apart from the rest of your environment.

A useful side effect: `unset SOME_VAR` takes that variable out of the profile,
so it is not carried into the next one.

</details>

---

## Where profiles can live

By default profiles are files on your machine. They can also come from a shared
backend, so a whole team works from one set of environments — or so your laptop
and your workstation stay in sync.

```mermaid
flowchart LR
    U["👤 you"] --> CLI["profiler"]
    CLI --> R{{"repository layer"}}
    R --> LOC["📁 <b>local</b><br/>~/.profiles/*.yml"]
    R --> CON["🟣 <b>consul</b><br/>KV store"]
    R --> SSM["🟠 <b>ssm</b><br/>AWS Parameter Store"]
    R --> VLT["⚫ <b>vault</b><br/>HashiCorp Vault"]
```

Every backend supports the same five verbs, so a command means the same thing
everywhere — only the store behind it changes:

| Verb | `local` | `consul` | `ssm` | `vault` |
|---|---|---|---|---|
| `list` | profile files in `profilesFolder` | keys under `consulProfilesPath` | profiles under the SSM path | secrets under `vaultProfilesPath` |
| `show` | variable names in the file | variable names in the KV entry | variable names of the parameters | variable names in the secret |
| `use` | export the file's vars | export the KV entry's vars | export the parameters | export the secret's vars |
| `add` | append to the file | write to the KV | put as SSM parameters | write to the secret |
| `remove` | delete the file, or one var | delete the KV entry, or one var | delete the parameters, or one | delete the secret, or one var |

Local is the default and its verbs are top level (`profiler add`,
`profiler show`). Remote backends namespace them (`profiler ssm add`,
`profiler consul show`), and `use` also accepts `profiler use <backend> <name>`.

`profiler list` shows **everything, from every backend at once** — and skips
any backend that does not answer, so a laptop with no Consul reachable simply
lists the local profiles.

<details>
<summary><b>Consul</b> — how a profile is stored</summary>

One KV key per profile, with a YAML value.

Key: `profiler/example_consul_profile`
Value:

```yaml
profile_name: test_consul
FOO: BAR
```

Configure with `consulAddress`, and `consulToken` **or** `consulTokenFile` if
your Consul uses ACLs. Profiler decides whether to use Consul by *asking* it: if
something answers at `consulAddress`, its profiles are listed; if nothing does,
Consul is skipped.

</details>

<details>
<summary><b>AWS SSM</b> — how a profile is stored</summary>

One parameter per variable, plus one for the profile name, all tagged
`profiler: true`.

| Name | Type | Value | Tags |
|---|---|---|---|
| `/profiler/ProfileName/profile_name` | String | `$ProfileName` | `profiler: true` |
| `/profiler/ProfileName/Key` | String | `$Value` | `profiler: true` |

Credentials come from the standard
[AWS SDK credential chain](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials)
— environment, shared credentials file, or instance role. Profiler decides
whether to use SSM by *resolving* those credentials rather than by calling AWS:
if the chain answers, `/profiler` parameters show up in `profiler list`; if not,
SSM is skipped.

> On a machine that has AWS credentials but no interest in SSM profiles, drop
> `ssm` from [`supportedRepositories`](#general-options) so `profiler list`
> stops asking.

</details>

<details>
<summary><b>Vault</b> — how a profile is stored</summary>

One secret per profile under `vaultProfilesPath`. Configure with
`vaultAddress`, `vaultToken` and optionally `vaultProfilesPath`.

</details>

---

## Command reference

| Command | What it does |
|---|---|
| `profiler` | Load the current directory's env files (`.profiler`, `*.env`, `.env.yml`, `.envrc`) — no named profile |
| `profiler list` | List every available profile, from every backend |
| `profiler show <profile>` | List the variable names a profile sets |
| `profiler add <profile>` | Create a profile that sets only its own name |
| `profiler add <profile> <KEY> <VALUE>` | Add a variable to a profile, creating the profile if needed |
| `profiler remove <profile>` | Delete a profile |
| `profiler remove <profile> <KEY>...` | Drop one or more variables from a profile |
| `profiler use <profile>` | Enter a new shell with the profile applied (stacks onto any profile in use) |
| `profiler use` | Same, from the `.profiler` file and env files in this directory |
| `profiler use <backend> <profile>` | Use a profile from `consul`, `ssm` or `vault` |
| `profiler status` | Print the profile currently in use — `A+B` when stacked |
| `profiler aws_mfa <token>` | Re-authenticate the active AWS profile with an MFA token |
| `profiler ssm <subcommand>` | `add`, `list`, `remove`, `show`, `use` against AWS SSM |
| `profiler consul <subcommand>` | The same, against Consul |
| `profiler vault <subcommand>` | The same, against Vault |
| `profiler help` | Show help |

### AWS MFA

`profiler aws_mfa <token>` takes the AWS credentials in the profile you are
already using and swaps them for MFA-backed session credentials — access key,
secret key and session token.

<div align="center">
<img src="images/aws_mfa_demo.png" alt="profiler aws_mfa in action" width="700">
</div>

> [!NOTE]
> It needs `AWS_MFA_USERNAME` set in your profile.

### Kubernetes namespaces

If a profile sets `K8S_NAMESPACE`, Profiler switches `kubectl` to that namespace
when you use the profile. Turn it off with
[`k8sSwitchNamespace: False`](#general-options).

---

## Configuration reference

The config file lives at `~/.profiler_cfg.yml`. A minimal, complete example:

```yaml
profilesFolder: /home/me/.profiles
shell: bash                   # optional — your current shell by default
preserveProfile: False        # optional — true by default
k8sSwitchNamespace: False     # optional — true by default
showRepoSourceInList: True    # optional — false by default
```

<details>
<summary><b>Where the config file is found</b></summary>

Override the location with `PROFILER_CFG`, which accepts either a file or a
directory to search:

```bash
export PROFILER_CFG="/my/preferred/path/my_cfg.yml"   # the file itself
export PROFILER_CFG="/my/preferred/path"              # a dir holding .profiler_cfg.yml
```

Profiler treats the value as a directory only if a directory already exists at
that path; otherwise it is the config file.

If no configuration file is found, a default one is created with
`profilesFolder` pointing at `$HOME/.profiles`. That applies to the default
`~/.profiler_cfg.yml` and to a `PROFILER_CFG` naming a file. When `PROFILER_CFG`
names an existing directory, Profiler reads from it but does not create a config
file there.

</details>

### General options

| Option | Default | What it does |
|---|---|---|
| `profilesFolder` | `$HOME/.profiles` | Where local profile files live |
| `shell` | your current `$SHELL` | Which shell to spawn. A profile's own `shell:` key wins over this |
| `preserveProfile` | `true` | Keep the `.profiler` file after use, or delete it |
| `profilerFileName` | `.profiler` | Name of the generated file |
| `k8sSwitchNamespace` | `true` | Run `kubectl` to switch namespace when a profile sets `K8S_NAMESPACE` |
| `showRepoSourceInList` | `false` | Suffix listed profiles with the backend they came from |
| `supportedRepositories` | `local, consul, ssm, vault` | Which backends to consult at all |
| `ignoredFiles` | `example.env, sample.env` | Directory env files to skip |
| `localProfiles` | *(unset)* | Narrow `profiler list` to these local profiles |
| `consulProfiles` | *(unset)* | Narrow `profiler list` to these Consul profiles |
| `ssmProfiles` | *(unset)* | Narrow `profiler list` to these SSM profiles |

### Backend options

| Option | Default | Example |
|---|---|---|
| `consulAddress` | `http://127.0.0.1:8500` | `http://consul.example.com:8500` |
| `consulToken` | *(none)* | `3d4a9009-eef0-4444-92c4-322e6a853385` |
| `consulTokenFile` | *(none)* | `/home/user/.consul_token` |
| `consulProfilesPath` | `profiler` | `teams/platform/profiler` |
| `ssmRegion` | `us-east-1` | `eu-west-1` |
| `ssmParameterTier` | `Standard` | `Advanced` |
| `ssmEndpoint` | *(the AWS endpoint for `ssmRegion`)* | `http://localhost:4566` |
| `vaultAddress` | `http://127.0.0.1:8200` | `https://vault.example.com:8200` |
| `vaultToken` | *(none)* | `5a78a463-1f9b-44cd-8ddd-8e03f3704772` |
| `vaultProfilesPath` | `secret/metadata/profiler` | `secret/metadata/team-profiles` |

`consulToken` and `consulTokenFile` are alternatives — use one or the other, or
neither if your Consul has no ACLs. `ssmEndpoint` points Profiler at an SSM that
is not AWS's own: localstack, a VPC endpoint, a FIPS endpoint.

### A few options worth explaining

#### `preserveProfile`

Every `profiler use` writes a `.profiler` file in the current directory
recording the environment it built. Keeping it (the default) means you can
re-enter the same environment later with a bare `profiler use`. Setting
`preserveProfile: False` deletes it once the profile is exported.

> [!TIP]
> Add `.profiler` to your **global** `.gitignore`.

#### `showRepoSourceInList`

Off by default, so profiles list by name alone:

```bash
$ profiler list
demo
staging
```

On, every profile held by a remote backend is suffixed with that backend's name
— useful when the same profile name exists in more than one. Local profiles are
never suffixed:

```bash
$ profiler list
demo
staging (consul)
staging (vault)
```

#### `localProfiles`, `consulProfiles`, `ssmProfiles`

These narrow `profiler list` to a chosen set per backend. Unset (or empty) means
no limit — every profile the backend holds is listed.

They are useful when a backend holds more profiles than a given machine cares
about: a Consul shared between several teams, or an AWS account shared between
stacks.

```yaml
consulProfiles:
  - staging
  - prod
```

```bash
# Consul KV holds staging, prod, team-b-dev and team-b-qa
$ profiler list
staging
prod
```

Each option narrows only its own backend, so the three are independent. A name
listed here that the backend does not hold is simply not shown, with no warning
— which is what lets one config file be shared across machines that each hold a
different subset of the profiles.

> [!NOTE]
> Vault has no equivalent option, so Vault profiles are never narrowed. And
> these options filter `profiler list` only: `profiler use` can still reach a
> profile the list does not show.

#### `ignoredFiles`

Projects often ship a template env file listing the variables an app supports,
filled with fake or blank values. Sourcing it would build a broken profile, so
`example.env` and `sample.env` are ignored by default. Add your own:

```yaml
ignoredFiles:
  - example.env
  - sample.env
  - NotThisOne.yml
  - NotEvenThisOne.env.yml
```

---

## Tips

### Show the active profile in your prompt

`profile_name` is exported, so any prompt that can read an environment variable
can display it. That is how the screenshots above show the profile in the
prompt bar.

With [powerline-shell](https://github.com/b-ryan/powerline-shell), there is a
ready-made segment:
[`cloud_profile.py`](https://github.com/julienlevasseur/powerline-shell/blob/master/powerline_shell/segments/cloud_profile.py)
(and a [Terraform workspace segment](https://github.com/julienlevasseur/powerline-shell/blob/master/powerline_shell/segments/terraform_workspace.py)
to go with it).

### Share shell history across levels

Because `profiler use` spawns a nested shell, that shell starts with its own
history. In Bash, this makes history follow you up and down the levels — add to
`.bashrc`:

```bash
# append to the history file, don't overwrite it
shopt -s histappend
# Bash examines PROMPT_COMMAND before printing each primary prompt:
#   history -a  append this session's history to the history file
#   history -c  clear the entries currently in memory
#   history -r  reload the history file into memory
PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND$'\n'}history -a; history -c; history -r"
```

Zsh has equivalent options (`setopt SHARE_HISTORY`). Check your shell's
documentation.

> [!NOTE]
> Profiler has been tested with **bash** and **zsh**. Contributions validating
> other shells are very welcome.

---

## A worked example

One profile per cloud account and stack — work AWS, work OpenStack, personal
AWS, a personal Nomad/Consul cluster:

```bash
$ profiler list
work_aws
work_openstack
perso_aws
perso_nomad
```

Then, inside a repository that provisions a Kubernetes cluster with
[KOPS](https://github.com/kubernetes/kops), a `.env.yml` holds what is specific
to *that project*:

```yaml
KOPS_STATE_STORE: s3://my-project-kubernetes-aws-state
KUBE_CTX_CLUSTER: my-project-k8s.my.domain.me
KUBE_USER: admin
KUBE_PASSWORD: ***************
```

`profiler use work_aws` in that directory gives you the account credentials from
the profile **and** the KOPS variables from the directory, in one shell you can
`exit` out of when you are done.

---

## Project layout

For contributors — see [Design.md](Design.md) for the full rationale.

```mermaid
flowchart LR
    CMD["<b>cmd/</b><br/>cobra commands"] --> REPO["<b>repository/</b><br/>IRepository interface<br/>+ factory"]
    REPO --> PKG["<b>pkg/{local,consul,ssm,vault}/</b><br/>one implementation each"]
    PKG --> PROF["<b>pkg/profile/</b><br/>the Profile type,<br/>merging and exec"]
```

Every command reaches a backend through the same `IRepository`, implemented once
per backend. All four implement all five verbs; none is stubbed.

- [CHANGELOG.md](CHANGELOG.md) — what changed, per release
- [Design.md](Design.md) — the command/backend matrix and the recorded decisions

## Related tools

| Tool | Overlap |
|---|---|
| [direnv](https://direnv.net/) | Per-directory variables. Profiler reads direnv's `.envrc`, and adds named, portable profiles on top |
| [kubectx](https://github.com/ahmetb/kubectx) / kubens | Kubernetes context switching. Profiler does the same for *any* tool driven by env vars |
| [aws-vault](https://github.com/99designs/aws-vault) | AWS credentials specifically, with OS keychain storage. Profiler is multi-cloud but not a secure store |
| [viper](https://github.com/spf13/viper) | Configuration *for an application you are writing* — a different job entirely |
