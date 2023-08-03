/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cachevolume

import (
	"errors"
	"fmt"

	"golang.org/x/exp/maps"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"
	blobcsiutil "sigs.k8s.io/blob-csi-driver/pkg/util"
	"sigs.k8s.io/cloud-provider-azure/pkg/provider/config"

	v1 "k8s.io/api/core/v1"
)

const (
	accountAnnotation               string = "external/edgecache-account"
	containerAnnotation             string = "external/edgecache-container"
	createVolumeAnnotation          string = "external/edgecache-create-volume"
	secretNameAnnotation            string = "external/edgecache-secret-name"
	secretNamespaceAnnotation       string = "external/edgecache-secret-namespace"
	storageAuthenticationAnnotation string = "external/edgecache-authentication"
	provisionerSecretNameField      string = "volume.kubernetes.io/provisioner-deletion-secret-name"
	provisionerSecretNamespaceField string = "volume.kubernetes.io/provisioner-deletion-secret-namespace"
)

var (
	validStorageAuthentications      = []string{"WorkloadIdentity", "AccountKey"}
	ErrVolumeAlreadyBeingProvisioned = errors.New("pv is already being provisioned")
)

type BlobAuth struct {
	account         string
	container       string
	secretName      string
	secretNamespace string
	authType        string
}

func NewBlobAuth(account, container, secretName, secretNamespace, authType string) BlobAuth {
	return BlobAuth{
		account:         account,
		container:       container,
		secretName:      secretName,
		secretNamespace: secretNamespace,
		authType:        authType,
	}
}

type PVCAnnotator struct {
	client clientset.Interface
}

func NewPVCAnnotator(client clientset.Interface) *PVCAnnotator {
	return &PVCAnnotator{
		client: client,
	}
}

type PVCAnnotatorInterface interface {
	SendProvisionVolume(pv *v1.PersistentVolume, cloudConfig config.AzureAuthConfig, strgAuthentication, acct, container string) error
}

func (c *PVCAnnotator) requestAuthIsValid(auth string) bool {
	return slices.Contains(validStorageAuthentications, auth)
}

func (c *PVCAnnotator) buildAnnotations(pv *v1.PersistentVolume, cfg config.AzureAuthConfig, providedAuth BlobAuth) (map[string]string, error) {
	annotations := map[string]string{
		createVolumeAnnotation:          "yes",
		accountAnnotation:               providedAuth.account,
		containerAnnotation:             providedAuth.container,
		storageAuthenticationAnnotation: providedAuth.authType,
	}

	// check if authentication is possible
	if providedAuth.authType == "WorkloadIdentity" && !cfg.UseFederatedWorkloadIdentityExtension {
		err := fmt.Errorf("workload identity was requested by the csi driver didn't initialize with the workload identity env vars")
		klog.Error(err)
		return nil, err

	} else if providedAuth.authType == "AccountKey" {
		secretName := providedAuth.secretName
		secretNamespace := providedAuth.secretNamespace
		if len(secretName) == 0 {
			// attempt to figure out the name of the kube secret for the storage account key
			var secretNameOk bool
			var secretNamespaceOk bool

			secretName, secretNameOk = pv.ObjectMeta.Annotations[provisionerSecretNameField]
			secretNamespace, secretNamespaceOk = pv.ObjectMeta.Annotations[provisionerSecretNamespaceField]
			if !secretNameOk || !secretNamespaceOk { // if keyName doesn't exist in the PV annotations
				err := fmt.Errorf("failed to discover storage account key secret; name: '%s' ns: '%s'", secretName, secretNamespace)
				klog.Error(err)
				return nil, err
			}
		}

		secretHintAnno := map[string]string{
			secretNameAnnotation:      secretName,
			secretNamespaceAnnotation: secretNamespace,
		}
		maps.Copy(annotations, secretHintAnno)

	}

	return annotations, nil
}

func (c *PVCAnnotator) needsToBeProvisioned(pvc *v1.PersistentVolumeClaim) bool {
	// check if pv connected to the pvc has already been passed to be created
	pvState, pvStateOk := pvc.ObjectMeta.Annotations[createVolumeAnnotation]
	if pvStateOk && pvState == "no" {
		return false
	}

	return true
}

func (c *PVCAnnotator) SendProvisionVolume(pv *v1.PersistentVolume, cloudConfig config.AzureAuthConfig, providedAuth BlobAuth) error {
	pvc, err := blobcsiutil.GetPVCByName(c.client, pv.Spec.ClaimRef.Name, pv.Spec.ClaimRef.Namespace)
	if err != nil {
		return err
	}

	if prepare := c.needsToBeProvisioned(pvc); !prepare {
		klog.Info("pv is already being provisioned")
		return ErrVolumeAlreadyBeingProvisioned
	}

	if valid := c.requestAuthIsValid(providedAuth.authType); !valid {
		err := fmt.Errorf("requested storage auth %s is not a member of valid auths %+v", providedAuth.authType, validStorageAuthentications)
		klog.Error(err)
		return err
	}

	annotations, err := c.buildAnnotations(pv, cloudConfig, providedAuth)
	if err != nil {
		return err
	}

	// Pass this to RetryUpdatePVC to confidently add these annotations
	var addAnnotations = func(inpvc *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
		pvcClone := inpvc.DeepCopy()
		maps.Copy(pvcClone.ObjectMeta.Annotations, annotations)
		return pvcClone
	}
	err = blobcsiutil.RetryUpdatePVC(c.client, pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name, addAnnotations)
	if err != nil {
		return err
	}

	return nil
}
