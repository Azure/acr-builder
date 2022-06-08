# Task Schema

## Current Version

`v1.0.0`

## Task

| Property | Type | Required | Default Value |
|----------|------|----------|---------------|
| [steps](#steps) | `step[]` | Required | N/A |
| [stepTimeout](#steptimeout) | `int` | Optional | 600 |
| [secrets](#secrets) | `secret[]` | Optional | N/A |
| [networks](#networks) | `network[]` | Optional | N/A |
| [env](#env) | `string[]` | Optional | N/A |
| [workingDirectory](#workingdirectory) | `string` | Optional | `$HOME` |
| [version](#version) | `string` | Optional | Yes | v1.0.0 |

## steps

An array of [step](#step) objects.

* Required
* Type: `step[]`

## stepTimeout

A [step's](#step) maximum execution time in seconds. This property defaults all [steps'](#steps) [timeout](#timeout) properties. A [step](#step) can override this property via [timeout](#timeout).

* Optional
* Type: `int`

## secrets

An array of [secret](#secret) objects.

* Optional
* Type: `secret[]`

## networks

An array of [network](#network) objects.

* Optional
* Type: `network[]`

## env

If specified on a [task](#task), these environment variables are applied to every [step](#step) in the format of `VARIABLE=value`.
If specified on a [step](#step), it will override any environment variables inherited from the [task](#task). In other words, `env` is always scoped to a [step](#step).

* Optional
* Type: `string[]`

## workingDirectory

Specifies the working directory of the container during runtime. If specified on a [task](#task), it defaults all of the [steps'](#step) `workingDirectory` properties. If specified on a [step](#step), it will override the property provided by the [task](#task).

* Optional
* Type: `string`

## version

The version of the [task](#task). If unspecified, defaults to the latest version.

* Optional
* Type: `string`

### step

An object with the following properties:

| Property | Type | Required | Default Value |
|----------|------|----------|---------------|
| [id](#id) | `string` | Optional | `acb_step_%d`, where `%d` is the 0-based index of the step top-down in the yaml |
| [cmd](#cmd) | `string` | Optional | N/A |
| [build](#build) | `string` | Optional | N/A |
| [workingDirectory](workingdirectory) | `string` | Optional | `$HOME` |
| [entryPoint](#entrypoint) | `string` | Optional | N/A |
| [user](#user) | `string` | Optional | N/A |
| [network](#network) | `string` | Optional | N/A |
| [isolation](#isolation) | `string` | Optional | `default` |
| [push](#push) | `string[]` | Optional | N/A |
| [env](#env) | `string[]` | Optional | N/A |
| [expose](#expose) | `string[]` | Optional | N/A |
| [ports](#ports) | `string[]` | Optional | N/A |
| [when](#when) | `string[]` | Optional | N/A |
| [timeout](#timeout) | `int` | Optional | 600 |
| [startDelay](#startdelay) | `int` | Optional | 0 |
| [retryDelay](#retrydelay) | `int` | Optional | 0 |
| [retries](#retries) | `int` | Optional | 0 |
| [downloadRetries](#downloadRetries) | `int` | Optional | 0 |
| [downloadRetryDelay](#downloadRetryDelay) | `int` | Optional | 0 |
| [repeat](#repeat) | `int` | Optional | 0 |
| [keep](#keep) | `bool` | Optional | false |
| [detach](#detach) | `bool` | Optional | false |
| [privileged](#privileged) | `bool` | Optional | false |
| [ignoreErrors](#ignoreerrors) | `bool` | Optional | false |
| [disableWorkingDirectoryOverride](#disableworkingdirectoryoverride) | `bool` | Optional | false |
| [pull](#pull) | `bool` | Optional | false |

* A [step](#step) must define either a [cmd](#cmd), [build](#build), or a [push](#push) property. It may not define more than one of the aforementioned properties.

#### id

The unique identifier for the [step](#step). Used as the name of the running container. Can be referenced across containers using this identifier as the tcp host and used in [when](#when).

* Optional
* Type: `string`
* Cannot contain spaces.

#### cmd

Defines which container to run along with any additional command line arguments.

Example:

```yaml
cmd: bash echo "hello world"
```

This runs a container called `bash` and tells it to run the `echo` command with `"Hello World"` as a parameter.

* Optional
* Type: `string`

#### build

Allows building containers.

Example:

```yaml
build: -f Dockerfile -t acr-builder:v1 https://github.com/Azure/acr-builder.git
```

* Optional
* Type: `string`

#### entryPoint

Sets the entry point of a container.

* Optional
* Type: `string`

#### user

Sets the username or UID of a container.

* Optional
* Type: `string`

#### isolation

Sets the isolation level of a container.

* Optional
* Type: `string`

#### push

Pushes the specified images to a container registry.

Example:

```yaml
push: ["example.azurecr.io/acb:v1"]
```

* Optional
* Type: `string[]`

#### env

Sets environment variables for the container during execution.

Example:

```yaml
env: ["K1=V1", "K2=V2"]
```

* Optional
* Type: `string[]`

#### expose

Exposes port(s) from the container.

* Optional
* Type: `string[]`

#### ports

Publishes port(s) from the container to the host.

* Optional
* Type: `string[]`

#### when

Describes when the step should get executed relative to other steps.
If unspecified, steps will be ran sequentially.

If `when: ["-"]` is specified, the container will immediately begin execution without any dependency on parent steps' execution.

Examples:

```yaml:
# build is ran, followed by cmd sequentially.
steps:
  - build: -f Dockerfile . -t example
  - cmd: example
```

```yaml:
# step 1 and 2 are both executed simultaneously.
steps:
  - id: step_1
    cmd: bash echo "1"

  - id: step_2
    cmd: bash echo "2"
    when: ["-"]
```

```yaml:
# step 1 and 2 are both executed simultaneously. Step 3 is executed after step 1 is completed.
steps:
  - id: step_1
    cmd: bash echo "1"

  - id: step_2
    cmd: bash echo "2"
    when: ["-"]

  - id: step_3
    cmd: bash echo "3"
    when: ["step_1"]
```

* Optional
* Type: `string[]`

#### timeout

The maximum execution time of a [step](#step) in seconds.

* Optional
* Type: `int`

#### startDelay

Defines the number of seconds to wait before executing the container.

* Optional
* Type: `int`

#### retries

The number of retries to attempt if a container fails its execution. A retry is only attempted if a container's exit code is non-zero.

* Optional
* Type: `int`

#### cmdDownloadRetries

The number of retries to attempt if downloading a container fails in a single cmd step.

* Optional
* Type: `int`

#### cmdDownloadRetryDelay

The delay in seconds between download retries in a cmd step.

* Optional
* Type: `int`

#### retryDelay

The delay in seconds between retries.

* Optional
* Type: `int`

#### repeat

The number of times to repeat the execution of a container.

* Optional
* Type: `int`

#### keep

Determines whether or not the container should be kept (all containers are removed by default) after it has been executed.

* Optional
* Type: `bool`

#### detach

Runs the container as a background process to avoid blocking.

* Optional
* Type: `bool`

#### privileged

Runs the container in privileged mode.

* Optional
* Type: `bool`

#### ignoreErrors

Ignores any errors during container execution and always marks the step as successful.

* Optional
* Type: `bool`

#### disableWorkingDirectoryOverride

Disables all `workingDirectory` override functionality. Use this in combination with [workingDirectory](#workingdirectory) to have complete control over the container's working directory.

#### pull

Forces a pull of the container before executing it to prevent any caching behavior.

* Optional
* Type: `bool`

### secret

An object with the following properties:

| Property | Type | Required | Default Value |
|----------|------|----------|---------------|
| `id` | `string` | Required | N/A |
| [keyvault](#keyvault) | `string` | Optional | N/A |
| [clientID](#clientid) | `string` | Optional | N/A |

#### keyvault

The key vault URL.

* Optional
* Type: `string`

#### clientID

The MSI user assigned identity client ID.

* Optional
* Type: `string`

### network

An object with the following properties:

| Property | Type | Required | Default Value |
|----------|------|----------|---------------|
| [name](#name) | `string` | Required | N/A |
| [driver](#driver) | `string` | Optional | N/A |
| [ipv6](#ipv6) | `bool` | Optional | false |
| [skipCreation](#skipcreation) | `bool` | Optional | false |
| [isDefault](#isdefault) | `bool` | Optional | false |

#### name

The name of the network.

* Required
* Type: `string`

#### driver

The driver to manage the network.

* Default: `bridge`
* Optional
* Type: `string`

#### ipv6

Enables ipv6 networking.

* Optional
* Type: `bool`

#### skipCreation

Skips creation of the network.

* Optional
* Type: `bool`

#### isDefault

Specifies whether or not the network is a default one provided by Azure Container Registry.

* Optional
* Type: `bool`