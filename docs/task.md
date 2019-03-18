# Task Schema

## Task Properties

| Property | Type | Required | Nullable |
|----------|------|----------|----------|
| [steps](#steps) | `step[]` | Required  | No |
| [stepTimeout](#steptimeout) | `int` | Optional  | Yes |
| [totalTimeout](#totaltimeout) | `int` | Optional | Yes |
| [secrets](#secrets) | `secret[]` | Optional | Yes |
| [networks](#networks) | `network[]` | Optional | Yes |
| [envs](#envs) | `string[]` | Optional | Yes |
| [workingDirectory](#workingdirectory) | `string` | Optional | Yes |
| [version](#version) | `string` | Optional | Yes |

## steps

An array of [step](#step) objects.

* Required
* Type: `step[]`

## stepTimeout

A [step's](#step) maximum execution time in seconds. This property defaults all [steps'](#steps) [timeout](#timeout) properties. A [step](#step) can override this property via [timeout](#timeout).

* Optional
* Type: `int`

## totalTimeout

The total execution execution time of the [task](#task) in seconds.

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

## envs

Default environment variables which are applied to every [step](#step) in the format of `VARIABLE=value`.

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

| Property | Type | Required | Nullable |
|----------|------|----------|----------|
| [id](#id) | `string` | Optional | Yes |
| [cmd](#cmd) | `string` | Optional | Yes |
| [build](#build) | `string` | Optional | Yes |
| [workingDirectory](workingdirectory) | `string` | Optional | Yes |
| [entryPoint](#entrypoint) | `string` | Optional | Yes |
| [user](#user) | `string` | Optional | Yes |
| [network](#network) | `string` | Optional | Yes |
| [isolation](#isolation) | `string` | Optional | Yes |
| [push](#push) | `string[]` | Optional | Yes |
| [env](#env) | `string[]` | Optional | Yes |
| [expose](#expose) | `string[]` | Optional | Yes |
| [ports](#ports) | `string[]` | Optional | Yes |
| [when](#when) | `string[]` | Optional | Yes |
| [timeout](#timeout) | `int` | Optional | Yes |
| [startDelay](#startdelay) | `int` | Optional | Yes |
| [retryDelay](#retrydelay) | `int` | Optional | Yes |
| [retries](#retries) | `int` | Optional | Yes |
| [repeat](#repeat) | `int` | Optional | Yes |
| [keep](#keep) | `bool` | Optional | Yes |
| [detach](#detach) | `bool` | Optional | Yes |
| [privileged](#privileged) | `bool` | Optional | Yes |
| [ignoreErrors](#ignoreerrors) | `bool` | Optional | Yes |
| [disableWorkingDirectoryOverride](#disableworkingdirectoryoverride) | `bool` | Optional | Yes |
| [pull](#pull) | `bool` | Optional | Yes |

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

The number of retries to attempt if a container fails its execution.

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

| Property | Type | Required | Nullable |
|----------|------|----------|----------|
| `id` | `string` | Optional | Yes |
| [akv](#akv) | `string` | Optional | Yes |
| [clientID](#clientid) | `string` | Optional | Yes |

#### akv

The Azure Key Vault (AKV) Secret URL.

* Optional
* Type: `string`

#### clientID

The MSI user assigned identity client ID.

* Optional
* Type: `string`

### network

An object with the following properties:

| Property | Type | Required | Nullable |
|----------|------|----------|----------|
| [name](#name) | `string` | Required | Yes |
| [driver](#driver) | `string` | Optional | Yes |
| [ipv6](#ipv6) | `bool` | Optional | Yes |
| [skipCreation](#skipcreation) | `bool` | Optional | Yes |
| [isDefault](#isdefault) | `bool` | Optional | Yes |

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