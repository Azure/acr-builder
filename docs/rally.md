## Structure of `rally.toml`:

A Rally config consists of steps. Each step is tailored to an independent container and describes how the pipeline can interact with the container.

```
# An example Step
## A Step describes how to interact with a container. 
## It consists of the following properties:

id = string (required)
args = [string, string, ...]
workDir = string (optional)
entryPoint = string (optional)
envs = [string, string, ...] (optional)
secretEnvs = [string, string, ...] (optional)
timeout = string (Go Duration format) (optional)
whenAll = [string, string, ...] (optional)
exitedWith = [int, int, ...] (optional)
exitedWithout = [int, int, ...](optional)
```

For details on each specific property in a Step, follow these links:
  - [id](#id)
  - [image](#image)
  - [args](#args)
  - [workDir](#workdir)
  - [entryPoint](#entrypoint)
  - [envs](#envs)
  - [secretEnvs](#secretenvs)
  - [timeout](#timeout)
  - [whenAll](#whenall)
  - [exitedWith](#exitedwith)
  - [exitedWithout](#exitedwithout)

## Context:

Rally can freely flow and manipulate context between Steps. It does this by creating a default workspace for each build request and storing artifacts in the workspace after each Step. This means a Step can access any artifact produced from an early Step. Note: given the parallel nature of Rally, it is up to users to plan their own concurrency model if using parallelism.

## Putting the pieces together:

Rally is able to chain steps together to allow parallel and sequential execution from a `rally.toml` template. It does this by creating a consistent DAG based on all Steps' `whenAll` property. Unless `whenAll` is specified as `-` it will not execute steps in parallel, it will assume all steps should proceed sequentially. In order for a Step B to block for a Step A in a sequential matter, it should use `whenAll: ['A']`. Rally reproduces the same DAG in a deterministic manner.

```
# rally.toml
stepTimeout = int (optional)
totalTimeout = int (optional)
push = [string, string, ...]

[[step]]
id = "..."

[[step]]

...

[[secrets]]
akv = string (optional)
secretEnv = string (optional)
```

For details on specific properties in `rally.toml`, review the following properties:
  - [secrets](#secrets)
  - [stepTimeout](#steptimeout)
  - [totalTimeout](#totaltimeout)
  - [push](#push)

## Examples:

### Using Context and Secrets:

```
totalTimeout = 600 # All steps must complete within 600 seconds.
stepTimeout = 400 # Individual steps have a default of 400 seconds to complete.

# Push the resulting images I made to ACR.
push = ["eric:{{- .Build.ID }}", "eric:latest"]

[[step]]
id = "clone"
args = ["clone", "--depth=1", "branch=={{ .Build.BranchName }}", "https://github.com/ehotinger/fsrus"]
timeout = "60s" # This step must complete within 60 seconds.

# Curl some webpage in parallel while we're cloning a git repository.
[[step]]
id = "curling"
args = ["curl", "https://www.ehotinger.com/recordmetrics"]
whenAll = ["-"] # - means to run curl immediately in parallel.

# Build the repository that I cloned earlier.
[[step]]
id = "buildit"
args = ["build", "-t", "eric:{{ .Build.ID }}", "-t", "latest", "-f", "Dockerfile", "."]
workingDir = "fsrus" # Use the previously cloned repository as the context for our build.
whenAll = ["clone"]
  
# Deploy the latest image to my pre-prod cluster.
[[step]]
id = "k8sci"
args = ["set", "image", "deployment/my-deployment", "eric:{{ .Build.ID }}"]
envs = ["MY_REGION=eastus"]
secretEnvs = ["MY_SECRET"]

[[secrets]]
akv = "ehotingerakv"
secretEnv = "MY_SECRET"
```

### Caching and Error Handling:

```
[[step]]
# Start off by pulling the image all of my apps depend on
id = "puller"
args = ["pull", "ubuntu"]
  
# Execute a bunch of builds in parallel using the previously pulled image as a cache
[[step]]
id = "build-foo"
args = ["build", "-f", "Dockerfile", "https://github.com/ehotinger/foo", "--cache-from=ubuntu"]
whenAll = ["-"]

[[step]]
id = "build-bar"
args = ["build", "-f", "Dockerfile", "https://github.com/ehotinger/bar", "--cache-from=ubuntu"]
whenAll = ["-"]

[[step]]
id = "build-qaz"
args = ["build", "-f", "Dockerfile", "https://github.com/ehotinger/qaz", "--cache-from=ubuntu"]
whenAll = ["-"]

[[step]]
id = "build-qux"
args = ["build", "-f", "Dockerfile", "https://github.com/ehotinger/qux", "--cache-from=ubuntu"]
whenAll = ["-"]

[[step]]
id = "panic"
args = ["write-log", "{{ .Build.ID }} failed"]
whenAll = ['build-foo", "build-bar", "build-qaz", "build-qux"]
exitedWith = [1, 2, 3]
```

## Step properties

### id
The `id` property is a unique identifier to reference the step throughout the pipeline.

### image

The `image` property of a step specifies which image to use when running the operation.

### args

The `args` property of a step specifies the command to run in the container as well as any arguments.

### workDir

`workDir` can be used to set a working directory when executing a step. By default, Rally will produce a default root directory as the working directory. However, if your build has more than one step, you can share the artifacts created from previous steps.

### entryPoint

`entryPoint` overrides the entry point of a step's container.

### envs

`envs` sets environment variables for a step.

### secretEnvs

`secretEnvs` is a list of environment variables which are encrypted using Azure Key Vault. These values are decrypted using the [secrets](#secrets) property.

### timeout

`timeout` is the maximum duration for a step to execute.

### whenAll

`whenAll` is used to block a Step's execution until one or more other Steps have been completed.

### exitedWith

`exitedWith` can be used to trigger a task when previous steps exited with one or more of the specified exit codes.

### exitedWithout
`exitedWithout` can be used to trigger a task when previous steps exited without one or more of the specified exit codes.

## `rally.toml` Properties

### secrets

`secrets` defines secrets to decrypt using Azure Key Vault. The decrypted value is set as the field specified in `secretEnv` which can be reference in scripts via `secretEnvs`

### stepTimeout

`stepTimeout` can be used to set the maximum time a step has to execute. This property can be overridden by a particular Step's individual `timeout` property.

### totalTimeout

`totalTimeout` can be used to set the maximum time all steps must execute within.

### push

`push` is an optional list of images and tags to push after the build has completed. This is a shortcut to creating multiple `push` commands after `build`s.