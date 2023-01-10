# Azure Blob Storage CSI driver for Kubernetes
[![Travis](https://travis-ci.org/kubernetes-sigs/blob-csi-driver.svg)](https://travis-ci.org/kubernetes-sigs/blob-csi-driver)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/blob-csi-driver/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/blob-csi-driver?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fblob-csi-driver.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fblob-csi-driver?ref=badge_shield)

### About
This driver allows Kubernetes to access Azure Storage through one of following methods:
 - [azure-storage-fuse](https://github.com/Azure/azure-storage-fuse)
 - [NFSv3](https://docs.microsoft.com/en-us/azure/storage/blobs/network-file-system-protocol-support)
 
Disclaimer: Deploying this driver manually is not an officially supported Microsoft product. For a fully managed and supported experience on Kubernetes, use [AKS with the managed blob csi driver](https://learn.microsoft.com/en-us/azure/aks/azure-blob-csi).

#### csi plugin name: `blob.csi.azure.com`

### Project status: GA

### Container Images & Kubernetes Compatibility:
|driver version  |Image                                                 | supported k8s version | built-in blobfuse v1 version | built-in blobfuse v2 version|
|----------------|------------------------------------------------------|-----------------------|------------------------------| ----------------------------|
|master branch   |mcr.microsoft.com/k8s/csi/blob-csi:latest             | 1.21+                 | 1.4.5                        | 2.0.1           | 
|v1.19.0         |mcr.microsoft.com/oss/kubernetes-csi/blob-csi:v1.19.0 | 1.21+                 | 1.4.5                        | 2.0.1           |
|v1.18.0         |mcr.microsoft.com/oss/kubernetes-csi/blob-csi:v1.18.0 | 1.21+                 | 1.4.5                        | 2.0.0-preview.3 |
|v1.17.0         |mcr.microsoft.com/oss/kubernetes-csi/blob-csi:v1.17.0 | 1.21+                 | 1.4.5                        | n/a             |

### Driver parameters
Please refer to `blob.csi.azure.com` [driver parameters](./docs/driver-parameters.md)

### Set up CSI driver on AKS cluster (only for AKS users)
follow guide [here](./docs/install-driver-on-aks.md)

### Prerequisites
#### Option#1: Provide cloud provider config with Azure credentials
 - This option depends on [cloud provider config file](https://github.com/kubernetes/cloud-provider-azure/blob/master/docs/cloud-provider-config.md), usually it's `/etc/kubernetes/azure.json` on agent nodes deployed by [AKS](https://docs.microsoft.com/en-us/azure/aks/) or [aks-engine](https://github.com/Azure/aks-engine), here is [azure.json example](./deploy/example/azure.json). <details> <summary>specify a different cloud provider config file</summary></br>create `azure-cred-file` configmap before driver installation, e.g. for OpenShift, it's `/etc/kubernetes/cloud.conf` (make sure config file path is in the `volumeMounts.mountPath`)
</br><pre>```kubectl create configmap azure-cred-file --from-literal=path="/etc/kubernetes/cloud.conf" --from-literal=path-windows="C:\\k\\cloud.conf" -n kube-system```</pre></details>

 - This driver also supports [read cloud config from kubernetes secret](./docs/read-from-secret.md) as first priority
 - Make sure identity used by driver has `Contributor` role on node resource group
 - [How to set up CSI driver on Azure RedHat OpenShift(ARO)](https://github.com/ezYakaEagle442/aro-pub-storage/blob/master/setup-store-CSI-driver-azure-blob.md)

#### Option#2: Bring your own storage account
This option does not depend on cloud provider config file, supports cross subscription and on-premise cluster scenario. Refer to [detailed steps](./deploy/example/e2e_usage.md#option2-bring-your-own-storage-account).

### Install driver on a Kubernetes cluster
 - install via [kubectl](./docs/install-blob-csi-driver.md) on public Azure (please use helm for Azure Stack, RedHat/CentOS)
 - install via [helm charts](./charts) on public Azure, Azure Stack, RedHat/CentOS
   - configure with [blobfuse-proxy](./deploy/blobfuse-proxy) to make blobfuse mount still available after driver restart

### Usage
 - [Basic usage](./deploy/example/e2e_usage.md)
 - [NFSv3](./deploy/example/nfs)
 - [fsGroupPolicy](./deploy/example/fsgroup)

### Troubleshooting
 - [CSI driver troubleshooting guide](./docs/csi-debug.md)

### Support
 - Please see our [support policy][support-policy]

### Limitations
 - Please refer to [Azure Blob Storage CSI Driver Limitations](./docs/limitations.md)

## Kubernetes Development
 - Please refer to [development guide](./docs/csi-dev.md)

### View CI Results
 - Check testgrid [provider-azure-blobfuse-csi-driver](https://testgrid.k8s.io/provider-azure-blobfuse-csi-driver) dashboard.

### Links
 - [azure-storage-fuse](https://github.com/Azure/azure-storage-fuse)
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
 - [Blobfuse FlexVolume driver](https://github.com/Azure/kubernetes-volume-drivers/tree/master/flexvolume/blobfuse)

[support-policy]: support.md
