param(
  [string]
  [Parameter(Mandatory = $true)]
  $client_id,
  [string]
  [Parameter(Mandatory = $true)]
  $client_secret,
  [string]
  [Parameter(Mandatory = $false)]
  $resourceGroupName = "atarraya-sample",
  [string]
  [Parameter(Mandatory = $false)]
  $location = "West Europe",
  [string]
  [Parameter(Mandatory = $false)]
  $identityName = "atarraya-sample",
  [string]
  [Parameter(Mandatory = $false)]
  $identitySelector = "requires-vault",
  [string]
  [Parameter(Mandatory = $false)]
  $aksName = "atarraya-sample",
  [string]
  [Parameter(Mandatory = $false)]
  $keyVaultName = "atarraya-kv"
)

# Get the current subscription
$subscriptionId = (az account show | ConvertFrom-Json).id

az group create -n $resourceGroupName -l $location

# Create Managed Identity
$identity = az identity create `
  -g $resourceGroupName `
  -n $identityName `
  -o json | ConvertFrom-Json

# Assign the Reader role to the Managed Identity
az role assignment create `
  --role "Reader" `
  --assignee $identity.principalId `
  --scope /subscriptions/$subscriptionId/resourcegroups/$resourceGroupName

# Assign the Managed Identity Operator role to the AKS Service Principal
az role assignment create `
  --role "Managed Identity Operator" `
  --assignee $client_id `
  --scope $identity.id

$currentUserObjectId = (az ad signed-in-user show | ConvertFrom-Json).objectId

$env:TF_VAR_client_id = $client_id
$env:TF_VAR_client_secret = $client_secret
$env:TF_VAR_resource_group_name = $resourceGroupName
$env:TF_VAR_cluster_name = $aksName
$env:TF_VAR_dns_prefix = $aksName
$env:TF_VAR_key_vault_name = $keyVaultName
$env:TF_VAR_managed_identity_client_id = $identity.principalId
$env:TF_VAR_current_user_object_id = $currentUserObjectId

terraform init
terraform apply -auto-approve

az aks get-credentials -g $resourceGroupName -n $aksName --admin

# Enable AAD Pod Identity on AKS
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml

# Create the Azure Identity and AzureIdentityBinding yaml on the fly
$k8sAzureIdentityandBinding = @"
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: $($identityName)
spec:
  type: 0
  ResourceID: $($identity.id)
  ClientID: $($identity.clientId)
---
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: $($identityName)-identity-binding
spec:
  AzureIdentity: $($identityName)
  Selector: $($identitySelector)
"@

# Deploy the yamls 
$k8sAzureIdentityandBinding | kubectl apply -f -

# Install Atarraya webhook
helm install atarraya-webhook ../charts/atarraya-webhook --namespace kube-system --set caBundle=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

# Deploy sample
((Get-Content -path ../cmd/atarraya/atarraya-test.yaml -Raw) -replace '<KEYVAULT NAME>', $keyVaultName) | kubectl apply -f -

kubectl logs -f $(kubectl get po --selector=app=az-atarraya-test -n default -o jsonpath='{.items[*].metadata.name}') -c testbox -n default