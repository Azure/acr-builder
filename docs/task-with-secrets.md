# Execute ACR Task with secrets from Azure Keyvault using MSI

acr-builder has added support to execute tasks with secrets. Currently it supports secrets from azure keyvault using managed service dientity (https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview).  The below article describes how to use acb to execute a task with secrets from Azure keyvault using Managed Identities. 

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
   --credential value          registry credentials in the format of 'server;username;password'
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
az keyvault secret set --name SampleSecret --value "ACB Secret" --vault-name myacbvault
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

### Create task file with secrets 

```
vim task-secrets.yaml

version: 1.0-preview-1
secrets:
  - id: mysecret
    akv: https://myacbvault.vault.azure.net/secrets/SampleSecret/2c68e8cd93b941389ac2ad735ffc0353
steps:
  - cmd: bash echo mysecret value is {{.Secrets.mysecret}}

```

### Execute the task using acb image

```
root@testacbvm:~# docker run -v $(pwd):/workspace --workdir /workspace -v /var/run/docker.sock:/var/run/docker.sock acb exec  --homevol $(pwd) -f task-secrets.yaml 

The output will print the value of the mysecret

2019/02/11 21:29:02 Using /home/azureuser as the home volume
2019/02/11 21:29:03 Creating Docker network: acb_default_network, driver: 'bridge'
2019/02/11 21:29:03 Successfully set up Docker network: acb_default_network
2019/02/11 21:29:03 Setting up Docker configuration...
2019/02/11 21:29:04 Successfully set up Docker configuration
2019/02/11 21:29:04 Executing step ID: acb_step_0. Working directory: '', Network: 'acb_default_network'
2019/02/11 21:29:04 Launching container with name: acb_step_0
mysecret value is ACB Secret
2019/02/11 21:29:05 Successfully executed container: acb_step_0
2019/02/11 21:29:05 Step ID: acb_step_0 marked as successful (elapsed time in seconds: 1.194597)
```

### Create task file with secrets properties set through values file.

```
vim task-secrets.yaml

version: 1.0-preview-1
secrets:
  - id: mysecret
    akv: https://myacbvault.vault.azure.net/secrets/SampleSecret/2c68e8cd93b941389ac2ad735ffc0353
  - id: mysecret1
    akv: {{.Values.akv1}}
    clientID: {{ .Values.id }}
steps:
  - cmd: bash echo mysecret value is {{.Secrets.mysecret}}
  - cmd: bash echo mysecret1 value is {{.Secrets.mysecret1}}

vim values.yaml

akv1: https://myacbvault.vault.azure.net/secrets/SampleSecret/2c68e8cd93b941389ac2ad735ffc0353
id: c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86

```

### Execute the task using acb image

```
root@testacbvm:~# docker run -v $(pwd):/workspace --workdir /workspace -v /var/run/docker.sock:/var/run/docker.sock acb exec  --homevol $(pwd) -f task-secrets.yaml --values values.yaml

The output will print the value of both secrets

2019/02/11 21:32:39 Using /home/azureuser as the home volume
2019/02/11 21:32:39 Creating Docker network: acb_default_network, driver: 'bridge'
2019/02/11 21:32:40 Successfully set up Docker network: acb_default_network
2019/02/11 21:32:40 Setting up Docker configuration...
2019/02/11 21:32:41 Successfully set up Docker configuration
2019/02/11 21:32:41 Executing step ID: acb_step_0. Working directory: '', Network: 'acb_default_network'
2019/02/11 21:32:41 Launching container with name: acb_step_0
mysecret value is ACB Secret
2019/02/11 21:32:42 Successfully executed container: acb_step_0
2019/02/11 21:32:42 Executing step ID: acb_step_1. Working directory: '', Network: 'acb_default_network'
2019/02/11 21:32:42 Launching container with name: acb_step_1
mysecret1 value is ACB Secret
2019/02/11 21:32:43 Successfully executed container: acb_step_1
2019/02/11 21:32:43 Step ID: acb_step_0 marked as successful (elapsed time in seconds: 1.206231)
2019/02/11 21:32:43 Step ID: acb_step_1 marked as successful (elapsed time in seconds: 1.252829)
```

If you're done with the resource group and all the resources it contains, delete it:

```
az group delete --name myResourceGroup
```
