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

package cachevolume

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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

func requestFail(action kubetesting.Action) (bool, runtime.Object, error) {
	return true, nil, errors.New(action.GetVerb() + " failed")
}

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

func TestAddFinalizer(t *testing.T) {
	t.Run("AlreadyFinalized", func(t *testing.T) {
		pv1 := pv()
		finalizers := []string{finalizerName}
		pv1.SetFinalizers(finalizers)
		client := fake.NewSimpleClientset(pv1)
		err := AddFinalizer(client, pv1, "account", "container")
		assert.Nil(t, err)
	})
	t.Run("MissingPVC", func(t *testing.T) {
		pv1 := pv()
		client := fake.NewSimpleClientset(pv1)
		err := AddFinalizer(client, pv1, "account", "container")
		assert.NotNil(t, err)
	})

	t.Run("UpdatePVCFails", func(t *testing.T) {
		pv1 := pv()
		pvc1 := pvc()
		client := fake.NewSimpleClientset(pv1, pvc1)
		client.PrependReactor("update", "persistentvolumeclaims", requestFail)
		err := AddFinalizer(client, pv1, "account", "cointainer")
		assert.NotNil(t, err)
		// check the finalizers are applied
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), []string{})
		assert.Equal(t, pvAfter.GetFinalizers(), []string{})
	})
	t.Run("PVCFinalizedUpdatePVFails", func(t *testing.T) {
		pv1 := pv()
		pvc1 := pvc()
		pvcFinalizers := []string{finalizerName, "hello"}
		pvc1.SetFinalizers(pvcFinalizers)
		client := fake.NewSimpleClientset(pv1, pvc1)
		client.PrependReactor("update", "persistentvolumes", requestFail)
		err := AddFinalizer(client, pv1, "account", "cointainer")
		assert.NotNil(t, err)
		// check the finalizers are applied
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), pvcFinalizers)
		assert.Equal(t, pvAfter.GetFinalizers(), []string{})
	})
	t.Run("PVCFinalized", func(t *testing.T) {
		pv1 := pv()
		pvc1 := pvc()
		pvcFinalizers := []string{finalizerName, "hello"}
		pvc1.SetFinalizers(pvcFinalizers)
		client := fake.NewSimpleClientset(pv1, pvc1)
		err := AddFinalizer(client, pv1, "account", "cointainer")
		assert.Nil(t, err)
		// check the finalizers are applied
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), pvcFinalizers)
		assert.Equal(t, pvAfter.GetFinalizers(), []string{finalizerName})
	})
	t.Run("BothFinalized", func(t *testing.T) {
		pv1 := pv()
		pvc1 := pvc()
		client := fake.NewSimpleClientset(pv1, pvc1)
		err := AddFinalizer(client, pv1, "account", "cointainer")
		assert.Nil(t, err)
		// check the finalizers are applied
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		expectedFinalizers := []string{finalizerName}
		assert.Equal(t, pvcAfter.GetFinalizers(), expectedFinalizers)
		assert.Equal(t, pvAfter.GetFinalizers(), expectedFinalizers)
		assert.Contains(t, pvAfter.GetAnnotations(), accountAnnotation)
		assert.Contains(t, pvAfter.GetAnnotations(), containerAnnotation)
	})
}

func TestRemoveFinalizer(t *testing.T) {
	finalizersBefore := []string{"hello", finalizerName, "goodbye"}
	expectedFinalizers := []string{"hello", "goodbye"}
	t.Run("NoneOk", func(t *testing.T) {
		pvc1 := pvc()
		pv1 := pv()
		client := fake.NewSimpleClientset(pv1, pvc1)
		err := RemoveFinalizer(client, pv1, pvc1)
		assert.Nil(t, err)
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), []string{})
		assert.Equal(t, pvAfter.GetFinalizers(), []string{})
	})
	t.Run("PVCUpdateFail", func(t *testing.T) {
		pvc1 := pvc()
		pv1 := pv()
		pvc1.SetFinalizers(finalizersBefore)
		pv1.SetFinalizers(finalizersBefore)
		client := fake.NewSimpleClientset(pv1, pvc1)
		client.PrependReactor("update", "persistentvolumeclaims", requestFail)
		err := RemoveFinalizer(client, pv1, pvc1)
		assert.NotNil(t, err)
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), finalizersBefore)
		assert.Equal(t, pvAfter.GetFinalizers(), finalizersBefore)
	})
	t.Run("PVUpdateFail", func(t *testing.T) {
		pvc1 := pvc()
		pv1 := pv()
		pvc1.SetFinalizers(finalizersBefore)
		pv1.SetFinalizers(finalizersBefore)
		client := fake.NewSimpleClientset(pv1, pvc1)
		client.PrependReactor("update", "persistentvolumes", requestFail)
		err := RemoveFinalizer(client, pv1, pvc1)
		assert.NotNil(t, err)
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), expectedFinalizers)
		assert.Equal(t, pvAfter.GetFinalizers(), finalizersBefore)
	})
	t.Run("BothRemoved", func(t *testing.T) {
		pvc1 := pvc()
		pvc1.SetFinalizers(finalizersBefore)
		pv1 := pv()
		pv1.SetFinalizers(finalizersBefore)
		client := fake.NewSimpleClientset(pv1, pvc1)
		err := RemoveFinalizer(client, pv1, pvc1)
		assert.Nil(t, err)
		pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), defaultPVCName, metav1.GetOptions{})
		pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), defaultPVName, metav1.GetOptions{})
		assert.Equal(t, pvcAfter.GetFinalizers(), expectedFinalizers)
		assert.Equal(t, pvAfter.GetFinalizers(), expectedFinalizers)
	})
}
