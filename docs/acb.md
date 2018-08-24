# Structure

A task config consists of steps. Each step is tailored to an independent container and describes how the task can interact with the container.

```yaml
# Example task that builds and runs an image
stepTimeout: 60
steps:
  - build: -t myimage .
  - cmd: myimage
```

Properties can define the execution characteristics of the entire task or individual step.

* [Task Properties](#task-properties) apply to the entire task or all steps in a task, like timeouts.  
* [Step Properties](#step-properties) define the behavior of a single step in the entire task and how they interact with other steps.
* [Template Values](templates.md) can be used in a task config file.

## Context

Azure Container Builder can freely flow and manipulate context between Steps. It does this by creating a default workspace for each build request and storing artifacts in the workspace after each Step. This means a Step can access any artifact produced from an early Step.

## Composing steps together

Azure Container Builder is able to chain steps together to allow parallel and sequential execution from an `acb.yaml` template. It does this by creating a consistent DAG based on all Steps' `when` property. Unless `when` is specified as `-` it will not execute steps in parallel, it will assume all steps should proceed sequentially. In order for a Step B to block for a Step A in a sequential matter, it should use `when: ['A']`. Azure Container Builder reproduces the same DAG in a deterministic manner.

```yaml
stepTimeout: int (optional)
totalTimeout: int (optional)
version: string
workingDirectory: string

steps:
  - id: someID
```

## Task Properties

For details on specific properties in the `acb.yaml`, review the following properties:

* [version](#version)
* [stepTimeout](#steptimeout)
* [totalTimeout](#totaltimeout)

### version

`version` is the semantic version of the task.

### stepTimeout

`stepTimeout` can be used to set the maximum time a step has to execute. This property can be overridden by a particular Step's individual `timeout` property in seconds.

### totalTimeout

`totalTimeout` can be used to set the maximum time all steps must execute within.

## Step properties

```yaml
# An example Step
## A Step describes how to interact with a container.
## It consists of the following properties:

id: string (optional)
cmd: string (optional)
build: string (optional) # Build takes precedence over cmd. Build is required if cmd is not present.
push: [string, string, ...]
workingDirectory: string (optional)
entryPoint: string (optional)
env: [string, string, ...] (optional)
ports: [string, string, ...] (optional)
when: [string, string, ...] (optional)
exitedWith: [int, int, ...] (optional)
exitedWithout: [int, int, ...] (optional)
timeout: int (in seconds) (optional)
keep: bool (optional)
detach: bool (optional)
startDelay: int (in seconds) (optional)
ignoreErrors: bool (optional)
```

For details on each specific property in a Step, follow these links:

* [id](#id)
* [cmd](#cmd)
* [build](#build)
* [push](#push)
* [workingDirectory](#workingdirectory)
* [entryPoint](#entrypoint)
* [env](#env)
* [ports](#ports)
* [when](#when)
* [exitedWith](#exitedwith)
* [exitedWithout](#exitedwithout)
* [timeout](#timeout)
* [keep](#keep)
* [detach](#deatch)
* [startDelay](#startdelay)
* [ignoreErrors](#ignoreerrors)

### id

The `id` property is a unique identifier to reference the step throughout the task.

### cmd

The `cmd` property of a step specifies which image to use when running the operation as well as any additional command-line parameters. This property is required if `build` is not present.

### build

The `build` property of a step specifies how to build a set of images. If build is specified, it takes precedence over `cmd`. It is required if `cmd` is not present.

### push

`push` is an optional list of images and tags to push after the build has completed. This is a shortcut to creating multiple `push` commands after `build`s.

### workingDirectory

`workingDirectory` can be used to set a working directory when executing a step. By default, Azure Container Builder will produce a default root directory as the working directory. However, if your build has more than one step, you can share the artifacts created from previous steps.

### entryPoint

`entryPoint` overrides the entry point of a step's container.

### env

`env` is a list of strings in `key=val` format which define environment variables for a step.

### ports

`ports` is a list of ports to publish to the host.

### timeout

`timeout` is the maximum duration for a step to execute in seconds.

### keep

`keep` determines whether or not the step's container should be kept after execution.

### detach

`detach` determines whether or not the container should be detached when running.

### when

`when` is used to block a Step's execution until one or more other Steps have been completed.

### exitedWith

`exitedWith` can be used to trigger a task when previous steps exited with one or more of the specified exit codes.

### exitedWithout

`exitedWithout` can be used to trigger a task when previous steps exited without one or more of the specified exit codes.

### startDelay

`startDelay` can be used to delay a step's execution. This is an integer value measured in seconds.

### ignoreErrors

If `ignoreErrors` is set to `true`, the step will be marked as complete regardless of whether or not an error occurred during its execution. Defaults to false.