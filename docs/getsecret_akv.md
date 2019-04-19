# GetSecrets from Azure Keyvault using MSI

acr-builder has added support to get secrets from azure keyvault using managed service identity (https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview). It has a new command called **getsecret** that can fetch secret from different vault providers. Currently it supports secrets from Azure key vault. The below article describes how to use acb to retrieve a secret from Azure keyvault using Managed Identities. 

We will use the command group `keyvault` of getsecret to retrieve secrets from Azure keyvault.

```
USAGE:
   acb getsecret keyvault [command options] [arguments...]

OPTIONS:
   --url value                 the azure keyvault secret URL
   --client-id value           the MSI user assigned identity client ID
```

Build and Push the acr-builder docker image to a docker hub or azure container registry. Here for simplicity , let's us push to docker hub:

```
docker build -t <your-dockerhub-username>/acb-getsecret .
docker push <your-dockerhub-username>/acb-getsecret
```

### Deploying acb to Azure Container Instances
To deploy the container, open either a terminal with the Azure CLI installed or check out CloudShell on Azure Portal.

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

### Deploying the acb image

Next, simply deploy the acb image and run the getsecret command to fetch the value of SampleSecret :

```
az container create \
    --resource-group myResourceGroup \
    --name acb-getsecret \
    --assign-identity $ID \
    --image <YourDockerHubUsername>/acb-getsecret \
    --command-line "acb get-secret keyvault --url https://myacbvault.vault.azure.net/secrets/SampleSecret/2c68e8cd93b941389ac2ad735ffc0353
```
The above command will create the container instance and set up everything needed for Managed Identities.

Use the following command to show that it was able to access the key vault and get the secret "SampleSecret" :

```
C:\>az container logs -n acb-getsecret -g myresourcegroup
2019/02/08 21:11:50 The secret value:
2019/02/08 21:11:50 ACB Secret
```

If you're done with the resource group and all the resources it contains, delete it:

```
az group delete --name myResourceGroup
```
