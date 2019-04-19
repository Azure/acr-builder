# Custom registry login using Identity

acr-builder has added support to login to multiple custom registries using managed service identity (https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview). The below article describes how to use acb to retrieve a login credentials from Azure keyvault using Managed Identities or directly logging in to the registry using Managed Identity. 

We will use **exec** command of the acr-builder to execute acr task with specified values

```
NAME:
   acb.exe exec - execute a task file

USAGE:
   acb.exe exec [command options] [arguments...]

OPTIONS:
   --file value, -f value      the path to the task file
   --encoded-file value        a base64 encoded task file
   --working-directory value   the default working directory to use if the underlying Task doesn't have one specified
   --network value             the default network to use
   --env value                 the default environment variables which are applied to each step (use --env multiple times or use commas: env1=val1,env2=val2)
   --credential value          login credentials for custom registry
   --dry-run                   evaluates the command, but doesn't execute it
   --debug                     enables diagnostic logging
   --values value              the path to the values file to use for rendering
   --encoded-values value      a base64 encoded values file to use for rendering
   --homevol value             the home volume to use
   --id value                  the unique run identifier
   --commit value, -c value    the commit SHA that triggered the run
   --repository value          the run's repository
   --branch value              the git branch
   --triggered-by value        describes what the run was triggered by
   --git-tag value             the git tag that triggered the run
   --registry value, -r value  the fully qualified name of the registry
   --set value                 set values on the command line (use --set multiple times or use commas: key1=val1,key2=val2)
   --az-cloud-name value       the name of azure environment
```

Build and Push the acr-builder docker image to a docker hub or azure container registry. Here for simplicity , let's us push to docker hub:

```
docker build -t <your-dockerhub-username>/acb-exec .
docker push <your-dockerhub-username>/acb-exec
```

### Run **acb** on Azure VM with Managed Service Identity enabled.

We will create a Azure Linux VM with Identity enabled. To create azure VM, open either a terminal with the Azure CLI installed or check out CloudShell on Azure Portal.

First, create a resource group or skip this step if you already have one:

```
az group create --name myResourceGroup --location eastus
```

### Create the User Assigned Identity

Next we create a user assigned identity. Once we create and add the permissions, we will be able to use this for container instances:

```
az identity create -g myResourceGroup --name myUserIdentity
```

The output is as following
```
{
  "clientId": "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86",
  "clientSecretUrl": "<secret URL>",
  "id": "<resourceID of identity>",
  "location": "eastus",
  "name": "myuseridentity",
  "principalId": "05109190-7167-4e14-8363-95bcf8b17992",
  "resourceGroup": "myresourcegroup",
  "tags": {},
  "tenantId": "72f988bf-86f1-41af-91ab-2d7cd011db47",
  "type": "Microsoft.ManagedIdentity/userAssignedIdentities"
}
```

lets add the important information from this output to some environment variables.

```
CLIENT_ID="<clientId>"
PRINCIPAL_ID="<principalId>"
ID="<id>"
```

### Create a Key Vault and Set the Permissions
Use the following command to create key vault:

```
az keyvault create -g myResourceGroup --name myacbvault
```

We can quickly add a secret to the new vault:

```
 az keyvault secret set --name username --value "registry1_username" --vault-name myacbvault
 az keyvault secret set --name password --value "registry1_password" --vault-name myacbvault
```

Now, we can give our identity access to the Key Vault:

```
az keyvault set-policy -n myacbvault --object-id $PRINCIPAL_ID -g myResourceGroup --secret-permissions get
```
The above command uses the environment variable we set to give our identity “get” permission for secrets in the Key Vault

### Create Azure VM with identity

```
az vm create -n testacbvm -g myResourceGroup --image UbuntuLTS --assign-identity $ID
```

### SSH to VM and set up with docker and acb image

```
sudo apt-get install docker-ce -y
docker pull <your-dockerhub-username>/acb-exec
docker tag <your-dockerhub-username>/acb-exec acb 
```

### Create a simple dockerfile
```
vim hello-world.dockerfile

FROM hello-world
```

### Create task file which requires pushing to multiple registries

```
vim mytask.yaml

version: v1.0.0
steps:
  - build: -t registry1.azurecr.io/hello-world . -f hello-world.dockerfile
  - push: ["registry1.azurecr.io/hello-world"]
  - build: -t registry2.azurecr.io/hello-world . -f hello-world.dockerfile
  - push: ["registry2.azurecr.io/hello-world"]

```

The idea is:
- We will login to `registry1.azurecr.io` using Vault Secrets (username and password) from myacbvault.
- We will login to `registry2.azurecr.io` using Identity. NOTE: you need to provide `$CLIENT_ID` atleast Contributor access so it can push images to registry.

### Execute the task using acb image

```
root@testacbvm:~# docker run -v $(pwd):/workspace --workdir /workspace -v /var/run/docker.sock:/var/run/docker.sock acb exec  \
--homevol $(pwd) -f mytask.yaml \
--credential '{"registry":"myregistry1.azurecr.io","userNameProviderType":"vaultsecret","username":"https://myacbvault.vault.azure.net/secrets/username","passwordProviderType":"vaultsecret","password":"https://myacbvault.vault.azure.net/secrets/password","identity":"c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"}' \
--credential '{"registry":"myregistry2.azurecr.io","identity":"c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86","aadResourceId":"https://management.azure.com/"}'


2019/03/23 00:58:07 Using /home/azureuser as the home volume
2019/03/23 00:58:08 Creating Docker network: acb_default_network, driver: 'bridge'
2019/03/23 00:58:09 Successfully set up Docker network: acb_default_network
2019/03/23 00:58:09 Setting up Docker configuration...
2019/03/23 00:58:11 Successfully set up Docker configuration

2019/03/23 00:58:11 Logging in to registry: myregistry1.azurecr.io
2019/03/23 00:58:14 Successfully logged into myregistry1.azurecr.io
2019/03/23 00:58:14 Logging in to registry: myregistry2.azurecr.io
2019/03/23 00:58:16 Successfully logged into myregistry2.azurecr.io

2019/03/23 00:58:16 Executing step ID: acb_step_0. Working directory: '', Network: 'acb_default_network'
2019/03/23 00:58:16 Obtaining source code and scanning for dependencies...
2019/03/23 00:58:19 Successfully obtained source code and scanned for dependencies
2019/03/23 00:58:19 Launching container with name: acb_step_0
Sending build context to Docker daemon   16.9kB
Step 1/1 : FROM hello-world
 ---> fce289e99eb9
Successfully built fce289e99eb9
Successfully tagged myregistry1.azurecr.io/hello-world:latest
2019/03/23 00:58:22 Successfully executed container: acb_step_0
2019/03/23 00:58:22 Executing step ID: acb_step_1. Working directory: '', Network: 'acb_default_network'
2019/03/23 00:58:22 Pushing image: myregistry1.azurecr.io/hello-world:latest, attempt 1
The push refers to repository [myregistry1.azurecr.io/hello-world]
af0b15c8625b: Preparing
af0b15c8625b: Layer already exists
latest: digest: sha256:92c7f9c92844bbbb5d0a101b22f7c2a7949e40f8ea90c8b3bc396879d95e899a size: 524
2019/03/23 00:58:25 Successfully pushed image: myregistry1.azurecr.io/hello-world:latest
2019/03/23 00:58:25 Executing step ID: acb_step_2. Working directory: '', Network: 'acb_default_network'
2019/03/23 00:58:25 Obtaining source code and scanning for dependencies...
2019/03/23 00:58:28 Successfully obtained source code and scanned for dependencies
2019/03/23 00:58:28 Launching container with name: acb_step_2
Sending build context to Docker daemon   16.9kB
Step 1/1 : FROM hello-world
 ---> fce289e99eb9
Successfully built fce289e99eb9
Successfully tagged myregistry2.azurecr.io/hello-world:latest
2019/03/23 00:58:30 Successfully executed container: acb_step_2
2019/03/23 00:58:30 Executing step ID: acb_step_3. Working directory: '', Network: 'acb_default_network'
2019/03/23 00:58:30 Pushing image: myregistry2.azurecr.io/hello-world:latest, attempt 1
The push refers to repository [myregistry2.azurecr.io/hello-world]
af0b15c8625b: Preparing
af0b15c8625b: Layer already exists
latest: digest: sha256:92c7f9c92844bbbb5d0a101b22f7c2a7949e40f8ea90c8b3bc396879d95e899a size: 524
2019/03/23 00:58:34 Successfully pushed image: myregistry2.azurecr.io/hello-world:latest
2019/03/23 00:58:34 Step ID: acb_step_0 marked as successful (elapsed time in seconds: 5.100598)
2019/03/23 00:58:34 Populating digests for step ID: acb_step_0...
2019/03/23 00:58:39 Successfully populated digests for step ID: acb_step_0
2019/03/23 00:58:39 Step ID: acb_step_1 marked as successful (elapsed time in seconds: 3.610487)
2019/03/23 00:58:39 Step ID: acb_step_2 marked as successful (elapsed time in seconds: 5.091546)
2019/03/23 00:58:39 Populating digests for step ID: acb_step_2...
2019/03/23 00:58:44 Successfully populated digests for step ID: acb_step_2
2019/03/23 00:58:44 Step ID: acb_step_3 marked as successful (elapsed time in seconds: 3.498264)
2019/03/23 00:58:44 The following dependencies were found:
2019/03/23 00:58:44
[{"image":{"registry":"myregistry1.azurecr.io","repository":"hello-world","tag":"latest","digest":"sha256:92c7f9c92844bbbb5d0a101b22f7c2a7949e40f8ea90c8b3bc396879d95e899a","reference":"myregistry1.azurecr.io/hello-world:latest"},"runtime-dependency":{"registry":"registry.hub.docker.com","repository":"library/hello-world","tag":"latest","digest":"sha256:2557e3c07ed1e38f26e389462d03ed943586f744621577a99efb77324b0fe535","reference":"hello-world:latest"},"buildtime-dependency":null,"git":{"git-head-revision":""}},{"image":{"registry":"myregistry2.azurecr.io","repository":"hello-world","tag":"latest","digest":"sha256:92c7f9c92844bbbb5d0a101b22f7c2a7949e40f8ea90c8b3bc396879d95e899a","reference":"myregistry2.azurecr.io/hello-world:latest"},"runtime-dependency":{"registry":"registry.hub.docker.com","repository":"library/hello-world","tag":"latest","digest":"sha256:2557e3c07ed1e38f26e389462d03ed943586f744621577a99efb77324b0fe535","reference":"hello-world:latest"},"buildtime-dependency":null,"git":{"git-head-revision":""}}]
```

If you're done with the resource group and all the resources it contains, delete it:

```
az group delete --name myResourceGroup
```
