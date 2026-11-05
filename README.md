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

#### 4. Test the CSI driver

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


## Multi-region block volumes

Keep the existing primary project and region in the CSI Secret's
`config.yaml`. Add the other regions under `regions`, and use the same
configuration and image for the controller and node plugin. For example:

```yaml
apiUrl: https://api.edgecenter.ru/cloud
apiToken: <existing-token>
projectID: 341372
regionID: 112
regions:
  dt:
    regionID: <actual-DT-region-ID>
    # projectID: <DT-project-ID-if-different>
```

The token must be able to look up instances and manage storage in every
configured scope. Additional regions inherit the primary project ID unless
overridden. Keep the primary project and region fixed while legacy PVs exist.

The node plugin resolves the VM UUID through the configured cloud API scopes.
It publishes the CSI topology key `topology.csi.edgecenter.org/region` with
value `<projectID>-<regionID>`. A lookup error blocks registration for that
node. The driver's topology comes from the cloud API. Workload selectors
such as `edgecenter.com/node-region=dt` still control scheduling independently.

Use a new StorageClass with `volumeBindingMode: WaitForFirstConsumer`. Copy
the disk type and other relevant settings from your current class. For example:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: edgecenter-regional
provisioner: csi.edgecenter.org
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

With a supported external-provisioner, enable `--strict-topology` so a
delayed-binding request is restricted to the scheduler-selected node topology.
For a DT-only StorageClass, also add `allowedTopologies` with the CSI topology
key and the actual DT project-region value. For a DT workload, select DT nodes
in its pod spec and reference the new class in its PVC.

New PVs advertise only the scope where their disk exists. Legacy UUID handles
continue to resolve to the primary scope. An existing PV provisioned without
topology keeps its old affinity. Restrict workloads using those primary-region
PVCs to primary-region nodes during the rollout. Snapshots can restore into
their own scope. Block disks cannot move between scopes or provide RWX.

Before enabling mixed-region provisioning, update the CSI node plugin on all
eligible nodes and verify that their topology labels are present, then update
the controller and create the new StorageClass. During staging validation,
create a PVC and pod in each region, verify the PV node affinity and cloud
disk location, then exercise attach, resize and deletion. Keep every configured
scope available until all of its PVs and snapshots are removed or migrated.
An older driver cannot operate on regional handles.
