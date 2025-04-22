# Edgecenter CSI driver for Kubernetes

### About

This driver allows Kubernetes to access [Edgecenter Disks](https://edgecenter.ru/knowledge-base/cloud/volumes) volume, csi plugin name: `csi.edgecenter.org`, supported accessModes: `ReadWriteOnce`

## Kubernetes installation

### Requirements

* Kubernetes 1.31+
* Kubernetes has to allow privileged containers

### Manual installation

#### 1. Create a configmap and secret with your credentials [config.yaml](./deploy/manifests/config.yaml)

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: csi-ec-plugin-config
  namespace: kube-system
type: Opaque
data:
  config.yaml: |-
    apiUrl: __CLOUD_API_URL__
    projectID: __PROJECT_ID__
    regionID: __REGION_ID__
    apiToken: __API_TOKEN__
---
kind: ConfigMap
apiVersion: v1
metadata:
   name: csi-ec-plugin-config
   namespace: kube-system
data:
   CLUSTER_ID: "__K8S_CLUSTER_ID__"
```

#### 2. Deploy the driver

```bash
cd deploy/manifests
for i in $(ls *.yaml);do kubectl apply -f $i;done
```

#### 4. Test the S3 driver

1. Create a pvc using the new storage class:

    ```bash
    kubectl create -f examples/pvc.yaml
    ```

2. Check if the PVC has been bound:

    ```bash
    $ kubectl get pvc test-pvc
    NAME         STATUS    VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
    test-pvc     Bound     pvc-c5d4634f-8507-11e8-9f33-0e243832354b   5Gi        RWO            csi-ec-hiiops  9s
    ```

3. Create a test pod which mounts your volume:

    ```bash
    kubectl create -f examples/pod.yaml
    ```

   If the pod can start, everything should be working.

## Development

This project can be built like any other go application.

```bash
go get -u github.com/Edge-Center/edgecenter-csi
```

