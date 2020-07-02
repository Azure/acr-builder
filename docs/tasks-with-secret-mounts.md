# Mountaineer: Mounting Secrets as Files into Containers

## Problem

Developers want to be able to access specified secrets as a file inside of containers generated at each step. Currently, developers are forced to encode any secrets to mount as environment variables passed in at each step. This requires developers to complete a post processing of these environment variables in order to extract and place into the destination format/file.

Furthermore, currently, multiline secrets and secrets that container special/reserved words and characters must manually be decoded when populated as environment variables.

## Proposed Solution

Add a new section to Tasks file which will allow Volumes and their contents to be listed. Inside each step, an optional volumeMounts property can be specified which lists all the Volumes and corresponding container paths that are to be mounted into the container at that specific step.

In order to account for possible multiline and special character secrets, we will require all secrets be base64 encoded prior. We will provide an overridden base64enc method which will by default handle nil strings.

We are loosely mimicking [Kubernetes Secret Mounting](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets):

## High Level Architecture

### Two Key Components

- Creating and Populating the listed Volumes
  - When we initially run the Task, we will take each listed Volume and create the appropriate files with the listed secret values on the host machine.
  - For all values listed, we will write these values
    - We require that all secret values provided be base64 encoded. We will first base64 decode the provided string.
    - Take the value from previous step and write to a file with name = filename
  - Then we will create a temporary data container that will add ALL the files with values into the created volume.
- Matching and Mounting specified Volumes at each step
  - For each Mount specified, we will take the named volume and the container file path and add it to the argument string builder as a new ‘-v’ flag.

## User Experience

### Execute a task and mount a secret from KeyVault to a step

``` bash
az acr run -f mounts-secrets.yaml https://github.com/Azure-Samples/acr-tasks.git
```

mount-secrets.yaml

``` YAML
secrets:
  - id: sampleSecret
    keyvault: https://myacbvault2.vault.azure.net/secrets/SampleSecret

volumes:
  - name: mysecrets
    secret:
      mysecret1: {{.Secrets.sampleSecret | b64enc}}
      mysecret2: {{"this is a non encoded string" | b64enc}}

steps:
  - cmd: bash cat /run/test/mysecret1 /run/test/mysecret2
    volumeMounts:
      - name: mysecrets
        mountPath: /run/test
```

### Execute a multistep task with multiple secrets

``` bash
az acr run -f mounts-multistep.yaml https://github.com/Azure-Samples/acr-tasks.git
```

mounts-multistep.yaml

``` YAML
volumes:
  - name: secret1
    secret:
      mysecret1: {{"-----SECRET VALUE 1------" | b64enc}}
  - name: secret2
    secret:
      mysecret2: {{"-----SECRET VALUE 2------" | b64enc}}

steps:
  - cmd: bash cat /run/test/mysecret1
    volumeMounts:
      - name: secret1
        mountPath: /run/test
  - cmd: bash cat /run/test/mysecret1 /run/test2/mysecret2
    volumeMounts:
      - name: secret1
        mountPath: /run/test
      - name: secret2
        mountPath: /run/test2
```

## [Documentation](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-reference-yaml)

### [Task Properties](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-reference-yaml#task-properties)

| Property       | Type         | Optional | Description   | Override Supported | Default Value |
| :------------- | :----------: | :------: | :-----------: | :----------------: | ------------: |
|  volumes | [volume, volume, …]  | Yes    | Array of volume objects. Specifies volumes with source content to mount to a step | None | None |

#### volume

| Property       | Type         | Optional | Description        | Default Value |
| :------------- | :----------: | :------: | :----------------: | ------------: |
|  name | string | no    | The name of the volume to be mounted. It must be alphanumeric with – and _ allowed | None |
| secret* | map[string]string | no | Each key of the map is the name of file created and populated in the volume. The value is the string version of the secret. Note: All secret values must be Base64 encoded | None |

### [CMD](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-reference-yaml#cmd)

#### [Properties: cmd](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-tasks-reference-yaml#properties-cmd)

| Property Name | Type | Optional |
| :----------- | :-----------------------------: | -------: |
| volumeMounts | [volumeMount, volumeMount, ...] | Optional |

#### volumeMount

The volume mount object specifies a volume to mount at a CMD step. The volume mount object has the following properties.

| Property | Type   | Optional | Description | Default Value |
| :------- | :--:   | :------: | :---------: | ------------: |
| name     | string | no       | The volume name to mount. Must exactly match name from volumes property | None |
