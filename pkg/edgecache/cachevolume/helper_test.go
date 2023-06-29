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
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/cloud-provider-azure/pkg/provider/config"
)

const (
	defaultPVCName      string = "pvc"
	defaultPVCNamespace string = "pvcspace"
	defaultPVName       string = "pv"
	defaultVolumeID     string = "volumeid"
)

func pvc() *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaultPVCName,
			Namespace:   defaultPVCNamespace,
			Finalizers:  []string{},
			Annotations: make(map[string]string),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: defaultPVName,
		},
	}
}

func pv() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaultPVName,
			Finalizers:  []string{},
			Annotations: make(map[string]string),
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{
				VolumeHandle: defaultVolumeID,
			}},
			ClaimRef: &v1.ObjectReference{
				Namespace: defaultPVCNamespace,
				Name:      defaultPVCName,
			},
		},
	}
}

func pvcWithVolume(p *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
	p.Spec = v1.PersistentVolumeClaimSpec{
		VolumeName: defaultPVName,
	}
	return p
}

func pvcWithAnnotations(p *v1.PersistentVolumeClaim, annotations map[string]string) *v1.PersistentVolumeClaim {
	p.SetAnnotations(annotations)
	return p
}

func pvWithAnnotations(p *v1.PersistentVolume, annotations map[string]string) *v1.PersistentVolume {
	p.SetAnnotations(annotations)
	return p
}

type FailureTestCase struct {
	name        string
	annotations map[string]string
	config      config.AzureAuthConfig
	blobAuth    BlobAuth
}

func TestSendProvisionVolumeFailures(t *testing.T) {
	testcases := []FailureTestCase{
		{
			name:        "InvalidAuthenticationType",
			annotations: map[string]string{},
			config:      config.AzureAuthConfig{},
			blobAuth:    NewBlobAuth("", "", "", "", "SomethingInvalid"),
		},
		{
			name:        "VolumeIsAlreadyBeingProvisioned",
			annotations: map[string]string{createVolumeAnnotation: "no"},
			config:      config.AzureAuthConfig{},
			blobAuth:    NewBlobAuth("", "", "", "", "WorkloadIdentity"),
		},
		{
			name:        "ConfigDoesntHaveWIEnabled",
			annotations: map[string]string{},
			config:      config.AzureAuthConfig{},
			blobAuth:    NewBlobAuth("", "", "", "", "WorkloadIdentity"),
		},
		{
			name:        "ProvisionerFieldsAreEmpty",
			annotations: map[string]string{},
			config:      config.AzureAuthConfig{},
			blobAuth:    NewBlobAuth("", "", "", "", "AccountKey"),
		},
	}

	for _, testcase := range testcases {
		pvc := pvcWithAnnotations(pvcWithVolume(pvc()), testcase.annotations)
		pv := pv()

		client := fake.NewSimpleClientset(pvc, pv)
		fkHelper := NewCVHelper(client)
		res := fkHelper.SendProvisionVolume(pv, testcase.config, testcase.blobAuth)
		assert.NotNil(t, res)

		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetAnnotations(), testcase.annotations)
	}
}

type SuccessTestCase struct {
	name                string
	config              config.AzureAuthConfig
	blobAuth            BlobAuth
	expectedAnnotations map[string]string
	pvAnnotations       map[string]string
}

func TestSendProvisionVolumeSuccess(t *testing.T) {
	acct := "iamaccount"
	container := "iamcontainer"
	secret := "shhhhhhhhh"
	secretNamespace := "shhhhhhh "

	testcases := []SuccessTestCase{
		{
			name:     "ValidWIAuth",
			config:   config.AzureAuthConfig{UseFederatedWorkloadIdentityExtension: true},
			blobAuth: NewBlobAuth(acct, container, "", "", "WorkloadIdentity"),
			expectedAnnotations: map[string]string{
				createVolumeAnnotation:          "yes",
				accountAnnotation:               acct,
				containerAnnotation:             container,
				storageAuthenticationAnnotation: "WorkloadIdentity",
			},
			pvAnnotations: map[string]string{},
		},
		{
			name:     "ProvidedAuthWithAccountKey",
			config:   config.AzureAuthConfig{},
			blobAuth: NewBlobAuth(acct, container, secret, secretNamespace, "AccountKey"),
			expectedAnnotations: map[string]string{
				createVolumeAnnotation:          "yes",
				accountAnnotation:               acct,
				containerAnnotation:             container,
				secretNamespaceAnnotation:       secretNamespace,
				secretNameAnnotation:            secret,
				storageAuthenticationAnnotation: "AccountKey",
			},
			pvAnnotations: map[string]string{},
		},
		{
			name:     "ProvisionerFieldsExist",
			config:   config.AzureAuthConfig{},
			blobAuth: NewBlobAuth(acct, container, "", "", "AccountKey"),
			expectedAnnotations: map[string]string{
				createVolumeAnnotation:          "yes",
				accountAnnotation:               acct,
				containerAnnotation:             container,
				secretNamespaceAnnotation:       secretNamespace,
				secretNameAnnotation:            secret,
				storageAuthenticationAnnotation: "AccountKey",
			},
			pvAnnotations: map[string]string{
				provisionerSecretNameField:      secret,
				provisionerSecretNamespaceField: secretNamespace,
			},
		},
	}

	for _, testcase := range testcases {
		pvc := pvcWithVolume(pvc())
		pv := pvWithAnnotations(pv(), testcase.pvAnnotations)

		client := fake.NewSimpleClientset(pvc, pv)
		fkHelper := NewCVHelper(client)
		res := fkHelper.SendProvisionVolume(pv, testcase.config, testcase.blobAuth)
		assert.Nil(t, res)

		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		assert.Equal(t, testcase.expectedAnnotations, pvcAfter.GetAnnotations())
	}
}
