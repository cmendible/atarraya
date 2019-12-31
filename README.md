# atarraya

**atarraya** is a solution designed to inject secrets from an [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/) into Pods running in [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) (AKS), using environment variables without having to modify your code or add dependencies to the Azure SDK.

This project is an evolution of my work on [az-keyvault-reader](https://github.com/cmendible/az-keyvault-reader) and inspired by the following post: [Inject secrets directly into Pods from Vault revisited](https://banzaicloud.com/blog/inject-secrets-into-pods-vault-revisited/) by Nandor Kracser for [BanzaiCloud](https://banzaicloud.com/)

## Components

* **atarraya-webhook**: A Mutating Admission Webhook designed to inject an executable (atarraya) into the containers inside Pods in such a way that the containers runs it instead of running the original application.
* **atarraya**: A wrapper executable designed to read secrets from an [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/) and inject them as environment variables into a process that runs the original application of the containers.

## Dependencies

**atarraya** works better if used with [AAD Pod Identities](https://github.com/Azure/aad-pod-identity).

## How it works

1. When a deployment is pushed to Kubernetes, **atarraya-webhook** check for the ```atarraya/keyvault``` annotation to see if it needs to do its magic.
1. If the ```atarraya/keyvault``` annotation is present, **atarraya-webhook** proceeds as follows:
    1. Mutates each container so it executes **atarraya** instead of the original application
    1. Mounts the volume named ```atarraya-volume``` where the **atarraya** will live
    1. Injects an init container named ```az-atarraya-init``` which copies the **atarraya** executable into the ```atarraya-volume``` volume.
    1. Injects a memory based volume named ```atarraya-volume```
    1. And injects an annotation to mark the container to avoid duplicate processing.
1. With **atarraya-webhook**'s work finished the init container runs and copies the **atarraya** executable into the ```atarraya-volume``` volume
1. Then the container runs **atarraya** which does the following:
    1. Reads all environment variables
    1. If environment variables starting with ```ATARRAYA_SECRET_``` exists, the executable strips the ```ATARRAYA_SECRET_``` prefix from the name of the variables and use the remaining value to querie for secrets in the [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/) specified by the ```atarraya/keyvault``` annotation.
    1. A new process is started where secrets are injected as environment variables without the ```ATARRAYA_SECRET_``` prefix and the original application of the container is executed.

## Sample

Deploying the following yaml to Kubernetes:

``` yaml
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: az-atarraya-test
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: az-atarraya-test
        aadpodidbinding: requires-vault
      annotations:
        atarraya/keyvault: <KEYVAULT NAME>
    spec:
      containers:
        - name: testbox
          image: alpine:3.10
          command: ["sh", "-c", "echo $<SECRET NAME>"]
          imagePullPolicy: IfNotPresent
          env:
            - name: ATARRAYA_SECRET_<SECRET NAME>
          resources:
            requests:
              memory: "16Mi"
              cpu: "100m"
            limits:
              memory: "32Mi"
              cpu: "200m"
```

will result in the following running inside the cluster:

``` yaml
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: az-atarraya-test
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: az-atarraya-test
        aadpodidbinding: requires-vault
    spec:
      initContainers:
        - name: az-atarraya-init
          image: cmendibl3/az-atarraya:0.1
          imagePullPolicy: Always
          command: ["sh", "-c", "cp /usr/local/bin/atarraya /atarraya/"]
          volumeMounts:
            - mountPath: "/atarraya/"
              name: atarraya-volume
      containers:
        - name: testbox
          image: alpine:3.10
          command:
            - /atarraya/atarraya
          args:
            - sh
            - -c
            - echo $<SECRET NAME>
          imagePullPolicy: IfNotPresent
          env:
            - name: ATARRAYA_SECRET_<SECRET NAME>
            - name: ATARRAYA_AZURE_KEYVAULT_NAME
              value: "<KEYVAULT NAME>"
          resources:
            requests:
              memory: "16Mi"
              cpu: "100m"
            limits:
              memory: "32Mi"
              cpu: "200m"
          volumeMounts:
            - mountPath: "/atarraya/"
              name: atarraya-volume
      volumes:
        - name: atarraya-volume
          emptyDir:
            medium: Memory
```

## What's the meaning of atarraya?

**atarraya** is the Venezuelan name for a [kind of fishing net](https://en.wikipedia.org/wiki/Cast_net) which is thrown by hand in such a manner that it spreads out while it's in the air before it sinks into the water.
