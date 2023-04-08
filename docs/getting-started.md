# Getting Started

Welcome to Terramate documentation! This page will give you an introduction to
the 80% of Terramate concepts that you will use on a daily basis.

> You will learn:
> 
> - Setup the project
> - How to create stacks
> - Manage globals
> - Generate Terraform code
> - Generate custom files
> - Orchestrate your stack's execution
> - How change detector works

No cloud account will be needed for this tutorial as we will play with pure
Terramate.

## Project Setup

If you don't have Terramate installed, then first head to the 
[installation](./installation.md) page and follow the steps there.

If you are new to Terramate or if you are creating a new project using it,
make sure you have the latest version installed. The command `terramate version` 
will inform you if your installed version is not the latest.

```shell
$ terramate version
0.2.16

Your version of Terramate is out of date! The latest version
is 0.2.17 (released on Mon Apr 3 00:00:00 UTC 2023).
You can update by downloading from https://github.com/mineiros-io/terramate/releases/tag/v0.2.17
```

Terramate has some features for enhancing the [git](https://git-scm.com/) workflow
of infrastructe as code (IaC) projects, then because of that it's better if
you start setting up a complete git repository, otherwise some features explained
here won't work.

### Setting up the repository

Terramate comes with sensible defaults for repositories created in 
[GitHub](https://github.com), [Gitlab](https://gitlab.com) or [Bitbucket](https://bitbucket.com) version control hostings, then the easiest way for getting started
is just create a repository in any of them, and clone the repository on your
machine.

If your company has a dedicated git server, there's a good chance that everything
will work smoothly as well, the exception being repositories that have a default
remote branch different than `origin/main`, and if that's the case you will need
additional configuration explained in the [project configuration](project-config.md)
page.

It's important that you start with a *cloned* repository instead of a locally
initialized git repository because we need a fully functioning repository, ie
default branch must have an initial commit, remote/upstreams must be set and
working, etc.

Let's say you cloned the repository into a `my-iac` directory:

```shell
$ git clone <url> my-iac
```

Then if you `cd` into that directory and execute Terramate commands, it will
detect it as a valid _Terramate Project_. In other words, any git project is a
valid _Terramate Project_. The Terramate tool behaves nicely with other language
files in the same repository and it adds **no constraints** to the organization
of your directories and files.

# Creating and listing Stacks

When working with Infrastructure as Code it's considered to be a best practice
to split up and organize your IaC into several smaller and isolated stacks.

The stacks are independent configurations that create resources when executed.
Sometimes it's not easy to figure if two resources must be kept in the same
stack or separated but asking the questions below to yourself could help:

- TBD: help please
- Are they related to the same cloud resource?
- Is it acceptable that changes to one resource could affect the other?
- Do they have similar lifecycles? Eg.: destroying one always imply destroying
  the other?

For more information about them, have a look at the [stack](./stack.md)
documentation page.

A stack is just a directory in the repository (even the root of the repository
could be a valid stack directory).
Stacks can have child stacks and stacks can have relationships (explained later
in the [orchestration](#orchestration) section).

Let's create two stacks for deploying a local [NGINX](https://nginx.org/) 
and a [PostgreSQL](https://postgresql.org) containers using the Terraform 
[docker provider](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs).

But first, let's create a git feature branch for the `nginx` service:

> <img src="https://cdn-icons-png.flaticon.com/512/427/427735.png" width="24px" />
> 
> This is an important step for understanding the Terramate change detection 
> feature.
> The _default branch_ (commonly `main`, but some teams uses another branch like
> `production` or `default`) represents the _production_ deployed infrastructure.
> At this point, the _default branch_ has no resources defined, but later on,
> Terramate will be able to compare your _work_ branch (_feature_ or _fix_)
> against the _default branch_ and identify which stacks changed, ie which
> stacks requires a `terraform apply`. 

```
$ git checkout -b nginx-service
Switched to a new branch 'nginx-service'
```

Terramate comes with a handy `terramate create` command to easily create stacks.

```shell
$ terramate create nginx
Created stack /nginx
```

This command creates the `nginx` directory containing a `stack.tm.hcl` file
similar to the one below:

```hcl
stack {
  name        = "nginx"
  description = "nginx"
  id          = "8b9c6e39-5145-40f1-90f1-67d022b6a6e9"
}
```

The `stack.name` and `stack.description` can be customized with strings that
better document the stack purpose. The `stack.id` is a randomly generated UUID
that must uniquely identify the stack in this repository.
For a complete list of the stack attributes, see the [stack](./stack.md)
documentation page.

Now if you execute `terramate list` you must see your brand new stack listed
in the output:

```shell
$ terramate list
2023-04-08T02:45:01+01:00 WRN repository has untracked files
nginx
```

As the project now has untracked files, Terramate is very picky and warns about
it because you should never deploy infrastructure from code that's not committed
and pushed to the remote git server. This behavior can be customized with the 
`terramate.config.git` config object, see [here](./project-config.md).

In a real world IaC project, only the CI/CD should deploy infrastructure, then
those safeguards are in place to avoid infrastructure being deployed with
temporary, uncommitted, unreviewed files.

For the purpose of this tutorial, let's disable those safeguards locally by
creating a `.gitignored` file called `disable_git_safeguards.tm.hcl` with
content below:

```hcl
terramate {
  config {
    git {
      check_untracked = false
      check_uncommitted = false
    }
  }
}
```

Please, don't forget to `gitignore` this file because those checks must always
be `enabled` in the CI/CD:

```
# .gitignore
disable_git_safeguards.tm.hcl
```

Now the `terramate list` returns:

```shell
$ terramate list
nginx
```

# Managing resources

Now let's create docker resources with Terraform.
Drop the file below into the `nginx/main.tf` file:

```hcl
terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0.1"
    }
  }
}

provider "docker" {}

resource "docker_image" "nginx" {
  name         = "nginx:latest"
  keep_locally = false
}

resource "docker_container" "nginx" {
  image = docker_image.nginx.image_id
  name  = "terramate-tutorial-nginx"
  ports {
    internal = 80
    external = 8000
  }
}
```

The Terraform configuration above creates two resources, the `docker_image` and 
the `docker_container` for running a `nginx` service exposed on host port `8000`.

> <img src="https://cdn-icons-png.flaticon.com/512/427/427735.png" width="24px" />
> 
> If your docker daemon is running on a custom port or you use Windows, then the
> "docker" provider need an additional `host` attribute for daemon address.
> On Windows, the config below is commonly needed:
> 
> ```hcl
> provider "docker" {
>   host    = "npipe:////.//pipe//docker_engine"
> }
> ```


From the root directory, run:

```shell
$ terramate run -- terraform init
```

The command above will execute `terraform init` in all Terramate stacks (just `nginx` stack at this point).

> <img src="https://cdn-icons-png.flaticon.com/512/427/427735.png" width="24px" />
> 
> You can think of `terramate run -- cmd` as a more robust version of the shell
> script below:
> 
>   ```shell
>   for stack in $(terramate list); do
>     cd $stack;
>     cmd;
>   done
>   ```
>
> But the `terramate run` also pulls `wanted`, computes the correct stack 
> execution order, detect changed stacks, run safeguards, etc.

The Terraform initialization will create the directory `nginx/.terraform` and
the file `nginx/.terraform.lock.hcl`. These files must never be committed to
the version control and it's recommended to be added to the `.gitignore` as well.
Additionally, you should also ignore the `terraform.tfstate` file as it contains
sensitive information.

Example:

```
# .gitignore

# Terramate files
disable_git_safeguards.tm.hcl

# Terraform files
terraform.tfstate*
.terraform
.terraform.lock.hcl
```

Executing `terramate run -- terraform apply` will create the resources.

```shell
$ terramate run -- terraform apply

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # docker_container.nginx will be created
  + resource "docker_container" "nginx" {
      + attach                                      = false
      + bridge                                      = (known after apply)
      + command                                     = (known after apply)
      + container_logs                              = (known after apply)
      + container_read_refresh_timeout_milliseconds = 15000
      + entrypoint                                  = (known after apply)
      + env                                         = (known after apply)
      + exit_code                                   = (known after apply)
      + hostname                                    = (known after apply)
      + id                                          = (known after apply)
      + image                                       = (known after apply)
      + init                                        = (known after apply)
      + ipc_mode                                    = (known after apply)
      + log_driver                                  = (known after apply)
      + logs                                        = false
      + must_run                                    = true
      + name                                        = "terramate-tutorial-nginx"
      + network_data                                = (known after apply)
      + read_only                                   = false
      + remove_volumes                              = true
      + restart                                     = "no"
      + rm                                          = false
      + runtime                                     = (known after apply)
      + security_opts                               = (known after apply)
      + shm_size                                    = (known after apply)
      + start                                       = true
      + stdin_open                                  = false
      + stop_signal                                 = (known after apply)
      + stop_timeout                                = (known after apply)
      + tty                                         = false
      + wait                                        = false
      + wait_timeout                                = 60

      + healthcheck {
          + interval     = (known after apply)
          + retries      = (known after apply)
          + start_period = (known after apply)
          + test         = (known after apply)
          + timeout      = (known after apply)
        }

      + labels {
          + label = (known after apply)
          + value = (known after apply)
        }

      + ports {
          + external = 8000
          + internal = 80
          + ip       = "0.0.0.0"
          + protocol = "tcp"
        }
    }

  # docker_image.nginx will be created
  + resource "docker_image" "nginx" {
      + id           = (known after apply)
      + image_id     = (known after apply)
      + keep_locally = false
      + name         = "nginx:latest"
      + repo_digest  = (known after apply)
    }

Plan: 2 to add, 0 to change, 0 to destroy.

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value: yes

docker_image.nginx: Creating...
docker_image.nginx: Still creating... [10s elapsed]
docker_image.nginx: Creation complete after 10s [id=sha256:080ed0ed8312deca92e9a769b518cdfa20f5278359bd156f3469dd8fa532db6bnginx:latest]
docker_container.nginx: Creating...
docker_container.nginx: Creation complete after 1s [id=0854270a600861cb72f36d3a78084c240826170601171416bfe3a8e0f1a4547c]

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.
```

Then now the `nginx` service should be running, you can check this with the
`docker ps` command:

```shell
$ docker ps
CONTAINER ID   IMAGE          COMMAND                  CREATED         STATUS         PORTS                  NAMES
0854270a6008   080ed0ed8312   "/docker-entrypoint.…"   2 minutes ago   Up 2 minutes   0.0.0.0:8000->80/tcp   terramate-tutorial-nginx
```

or opening [http://localhost:8000/](https://localhost:8000/) in the browser.

You just deployed something locally. Yay!!!

> <img src="https://cdn-icons-png.flaticon.com/512/1680/1680012.png" width="24px" />
>
> When using real world cloud infrastructure (`aws` or `gcloud` providers) you 
> should use a _development_ or _testing cloud_ account when invoking 
> `terraform apply` from your own machine.

Now that you tested and the resource is working, you can _destroy_ it and prep
for making it into _production_.

Terramate does not prevent you from invoking `terraform` directly as other similar
tools, then let's go ahead and call `terraform destroy` from the `nginx` directory:

```shell
$ cd nginx
$ terraform destroy
docker_image.nginx: Refreshing state... [id=sha256:080ed0ed8312deca92e9a769b518cdfa20f5278359bd156f3469dd8fa532db6bnginx:latest]
docker_container.nginx: Refreshing state... [id=0854270a600861cb72f36d3a78084c240826170601171416bfe3a8e0f1a4547c]

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  - destroy

Terraform will perform the following actions:

  # docker_container.nginx will be destroyed
  - resource "docker_container" "nginx" {
      - attach                                      = false -> null
      - command                                     = [
          - "nginx",
          - "-g",
          - "daemon off;",
        ] -> null
      - container_read_refresh_timeout_milliseconds = 15000 -> null
      - cpu_shares                                  = 0 -> null
      - dns                                         = [] -> null
      - dns_opts                                    = [] -> null
      - dns_search                                  = [] -> null
      - entrypoint                                  = [
          - "/docker-entrypoint.sh",
        ] -> null
      - env                                         = [] -> null
      - group_add                                   = [] -> null
      - hostname                                    = "0854270a6008" -> null
      - id                                          = "0854270a600861cb72f36d3a78084c240826170601171416bfe3a8e0f1a4547c" -> null
      - image                                       = "sha256:080ed0ed8312deca92e9a769b518cdfa20f5278359bd156f3469dd8fa532db6b" -> null
      - init                                        = false -> null
      - ipc_mode                                    = "private" -> null
      - log_driver                                  = "json-file" -> null
      - log_opts                                    = {} -> null
      - logs                                        = false -> null
      - max_retry_count                             = 0 -> null
      - memory                                      = 0 -> null
      - memory_swap                                 = 0 -> null
      - must_run                                    = true -> null
      - name                                        = "terramate-tutorial-nginx" -> null
      - network_data                                = [
          - {
              - gateway                   = "172.17.0.1"
              - global_ipv6_address       = ""
              - global_ipv6_prefix_length = 0
              - ip_address                = "172.17.0.2"
              - ip_prefix_length          = 16
              - ipv6_gateway              = ""
              - mac_address               = "02:42:ac:11:00:02"
              - network_name              = "bridge"
            },
        ] -> null
      - network_mode                                = "default" -> null
      - privileged                                  = false -> null
      - publish_all_ports                           = false -> null
      - read_only                                   = false -> null
      - remove_volumes                              = true -> null
      - restart                                     = "no" -> null
      - rm                                          = false -> null
      - runtime                                     = "runc" -> null
      - security_opts                               = [] -> null
      - shm_size                                    = 64 -> null
      - start                                       = true -> null
      - stdin_open                                  = false -> null
      - stop_signal                                 = "SIGQUIT" -> null
      - stop_timeout                                = 0 -> null
      - storage_opts                                = {} -> null
      - sysctls                                     = {} -> null
      - tmpfs                                       = {} -> null
      - tty                                         = false -> null
      - wait                                        = false -> null
      - wait_timeout                                = 60 -> null

      - ports {
          - external = 8000 -> null
          - internal = 80 -> null
          - ip       = "0.0.0.0" -> null
          - protocol = "tcp" -> null
        }
    }

  # docker_image.nginx will be destroyed
  - resource "docker_image" "nginx" {
      - id           = "sha256:080ed0ed8312deca92e9a769b518cdfa20f5278359bd156f3469dd8fa532db6bnginx:latest" -> null
      - image_id     = "sha256:080ed0ed8312deca92e9a769b518cdfa20f5278359bd156f3469dd8fa532db6b" -> null
      - keep_locally = false -> null
      - name         = "nginx:latest" -> null
      - repo_digest  = "nginx@sha256:2ab30d6ac53580a6db8b657abf0f68d75360ff5cc1670a85acb5bd85ba1b19c0" -> null
    }

Plan: 0 to add, 0 to change, 2 to destroy.

Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value: yes

docker_container.nginx: Destroying... [id=0854270a600861cb72f36d3a78084c240826170601171416bfe3a8e0f1a4547c]
docker_container.nginx: Destruction complete after 1s
docker_image.nginx: Destroying... [id=sha256:080ed0ed8312deca92e9a769b518cdfa20f5278359bd156f3469dd8fa532db6bnginx:latest]
docker_image.nginx: Destruction complete after 0s

Destroy complete! Resources: 2 destroyed.
```

> <img src="https://cdn-icons-png.flaticon.com/512/1680/1680012.png" width="24px" />
> 
> When you need to explicitly _destroy_ stacks, it's better to `cd` into the
> specific stack and invoke `terraform destroy` directly because `terramate run`
> will execute in all stacks by default.

Done! You're now again in a clean slate.

# Change detection

So the changes you did in this branch works, then now it's time to commit
everything and follow your git workflow to get this merged into production.

```shell
$ git add nginx
$ git commit -m "feat: create the NGINX service"
[nginx-service 8969b0c] feat: create the NGINX service
 1 file changed, 2 insertions(+)
```

But first, let's check what the `--changed` option of Terramate tells us:

```shell
$ terramate list --changed
nginx
```

The `--changed` option compares the current commit against the _default branch_
latest commit and computes which stacks has differences (changes to be applied).
It's always wise to check the changed stacks before going forward, so you know
exactly what's going to be applied in production.

Now the process to get this merged and applied into _production_ depends on your
company's standards, policy, coding culture, etc, but usually it involves pushing 
your branch to the default git upstream (commonly `origin`) and create a request 
for code review (a _Pull Request_ in Github or a _Merge Request_ in GitLab).

Eventually, your contribution is going to be accepted and merged into the
_default branch_ and then the CI/CD can kick in and deploy the changes in the
infrastructure. 

Let's mimick here in simple terms what would happen in a CI/CD pipeline for applying changes to the _default branch_.

> Note: every CI is different and advanced git features are used in most CIs to
> ensure low network bandwitch, low storage utilization, faster merges, and so on.
> The steps described below are useful for you understand the concepts behind
> a CI pipeline running on the `main` branch.

So let's move to the `main` branch and merge your _feature branch_ into it:

```shell
$ git checkout main
$ git merge --no-ff nginx-service
Merge made by the 'ort' strategy.
 .gitignore         |  9 +++++++++
 nginx/main.tf      | 26 ++++++++++++++++++++++++++
 nginx/stack.tm.hcl |  5 +++++
 3 files changed, 41 insertions(+)
 create mode 100644 .gitignore
 create mode 100644 nginx/main.tf
 create mode 100644 nginx/stack.tm.hcl
```

See the `--no-ff` flag to `git merge`? It's the default in most (if not all)
git hostings (GitHub, GitLab, Bitbucket, etc) and it means that a _merge commit_
will always be created even when it's not needed. Terramate uses this _merge commit_
to figure what was the last merged code, ie what's the base revision used when
comparing differences. For more information about this process, see the
[change detection](./change-detection.md) documentation page.

Then executing `terramate list --changed` in the _default branch_ (`main` in 
this case) automatically computes introduced by the last merged _Pull/Merge
Request_.

```
$ git branch
* main
  nginx-service
$ terramate list --changed
nginx
```

Then your CI/CD pipeline for changes in the `main` branch can be simply:

- `terramate run --changed -- terraform init`
- `terramate run --changed -- terraform apply -input=false`

> <img src="https://cdn-icons-png.flaticon.com/512/427/427735.png" width="24px" />
> 
> It's a good practice to have some kind of automation in the users _Pull/Merge_
> _Request_ to execute a Terraform Plan of the introduced changes with the
> output submitted for reviewers (eg.: a link to the CI/CD log of the run can
> be submitted as a comment in the git interface).
> The plan should be against the changed stacks:
> ```shell
> terramate run --changed -- terraform plan -input=false
> ```
>
> Then when changes are merged into `main`, Terramate will just apply the same
> set of changes already reviewed and approved.
> Additionally, a Terraform plan file can be created with `-out=pr.tfplan` and
> saved as an artifact for later be used by the pipeline running on `main`.

# Code generation

TODO

Now let's create the _PostgreSQL_ stack.

```shell
$ terramate create postgresql
Created stack /postgresql
```

Running list now shows:

```shell
$ terramate list
nginx
postgresql
```

TODO: create postgres docker and make both stacks DRY

TODO: configure the www index page of the container using code generation.
```hcl
terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0.1"
    }
  }
}

variable "www_path" {
  description = "Path to the www directory to be mapped into NGINX container"
}

provider "docker" {}

resource "docker_image" "nginx" {
  name         = "nginx:latest"
  keep_locally = false
}

resource "docker_container" "nginx" {
  image = docker_image.nginx.image_id
  name  = "tutorial"
  ports {
    internal = 80
    external = 8000
  }

  volumes {
    host_path      = var.pwd
    container_path = "/usr/share/nginx/html"
  }
}
```