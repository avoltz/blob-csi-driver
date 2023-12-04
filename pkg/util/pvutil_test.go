/*
Copyright 2022 The Kubernetes Authors.

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

package util

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

const (
	defaultPVCName      string = "pvc"
	defaultPVCNamespace string = "namespace"
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

//revive:disable:unused-parameter
func conflictError(action kubetesting.Action) (bool, runtime.Object, error) {
	conflictError := kuberrors.NewApplyConflict([]metav1.StatusCause{}, "OperationNotPermitted")
	return true, nil, conflictError
}

//revive:enable:unused-parameter

func TestGetPVByVolumeID(t *testing.T) {
	t.Run("ListFail", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.PrependReactor("list", "persistentvolumes", func(action kubetesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("error")
		})
		pv, err := GetPVByVolumeID(client, defaultVolumeID)
		assert.Nil(t, pv)
		assert.NotNil(t, err)
	})
	t.Run("NoneFound", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		pv, err := GetPVByVolumeID(client, defaultVolumeID)
		assert.Nil(t, pv)
		assert.NotNil(t, err)
	})
	t.Run("MatchFound", func(t *testing.T) {
		client := fake.NewSimpleClientset(pv())
		pv, err := GetPVByVolumeID(client, defaultVolumeID)
		assert.NotNil(t, pv)
		assert.Nil(t, err)
	})
	t.Run("NilCSISpec", func(t *testing.T) {
		client := fake.NewSimpleClientset(&v1.PersistentVolume{Spec: v1.PersistentVolumeSpec{PersistentVolumeSource: v1.PersistentVolumeSource{CSI: nil}}})
		pv, err := GetPVByVolumeID(client, defaultVolumeID)
		assert.Nil(t, pv)
		assert.NotNil(t, err)
	})
}

func TestGetPVByName(t *testing.T) {
	t.Run("NoneFound", func(t *testing.T) {
		client := fake.NewSimpleClientset(pv())
		pv, err := GetPVByName(client, "other")
		assert.Nil(t, pv)
		assert.NotNil(t, err)
	})
	t.Run("Found", func(t *testing.T) {
		client := fake.NewSimpleClientset(pv())
		pv, err := GetPVByName(client, defaultPVName)
		assert.NotNil(t, pv)
		assert.Nil(t, err)
	})
}

func TestRetryUpdatePVC(t *testing.T) {
	t.Run("NonIsConflictError", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		var lamfn = func(*v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim { return pvc() }
		err := RetryUpdatePVC(client, defaultPVCNamespace, defaultPVCName, lamfn)
		assert.NotNil(t, err)
	})

	t.Run("IsConflictErrorTimeout", func(t *testing.T) {
		client := fake.NewSimpleClientset(pvc())
		client.PrependReactor("update", "persistentvolumeclaims", conflictError)
		var lamfn = func(*v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
			pvc := pvc()
			pvc.ObjectMeta.Finalizers = append(pvc.GetFinalizers(), "hello")
			return pvc
		}
		err := RetryUpdatePVC(client, defaultPVCNamespace, defaultPVCName, lamfn)
		assert.NotNil(t, err)
	})

	t.Run("NoError", func(t *testing.T) {
		client := fake.NewSimpleClientset(pvc())
		var lamfn = func(*v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
			pvc := pvc()
			pvc.ObjectMeta.Finalizers = append(pvc.GetFinalizers(), "hello")
			return pvc
		}
		err := RetryUpdatePVC(client, defaultPVCNamespace, defaultPVCName, lamfn)
		assert.Nil(t, err)
		afterPVC, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		assert.Equal(t, []string{"hello"}, afterPVC.GetFinalizers())
	})
}
