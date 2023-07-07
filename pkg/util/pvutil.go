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
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// PVCModification is a type of function that modifies a PVC
type PVCModification func(*v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim

// PVModification is a type of function that modifies a PV
type PVModification func(*v1.PersistentVolume) *v1.PersistentVolume

func GetPVByVolumeID(client clientset.Interface, volumeID string) (*v1.PersistentVolume, error) {
	klog.V(3).Infof("No pvName provided, looking up via volumeID: %s", volumeID)
	pvList, err := client.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("unable to list volumes via volumeID: %s", volumeID)
		return nil, err
	}
	for _, pv := range pvList.Items {
		if pv.Spec.CSI != nil && pv.Spec.CSI.VolumeHandle == volumeID {
			return &pv, nil
		}
	}
	return nil, errors.New("no pv found")
}

func GetPVByName(client clientset.Interface, pvName string) (*v1.PersistentVolume, error) {
	pv, err := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("unable to get PV %s", pvName)
		return nil, err
	}
	return pv, nil
}

func GetPVCByName(client clientset.Interface, pvcName string, namespace string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("unable to get PVC %s", pvcName)
		return nil, err
	}
	return pvc, nil
}

var CustomRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   2.0,
	Jitter:   0.1,
}

// Attempts to update a pvc, retries if pvc isn't fresh
func RetryUpdatePVC(client clientset.Interface, namespace string, pvcName string, fn PVCModification) error {
	err := retry.RetryOnConflict(CustomRetry, func() error {
		// Fetch the resource here; you need to refetch it on every try, since
		// if you got a conflict on the last update attempt then you need to get
		// the current version before making your own changes.
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("unable to get PVC %s", pvcName)
			return err
		}

		// Make whatever updates to the resource are needed
		pvc = fn(pvc)

		// Try to update
		_, err = client.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), pvc, metav1.UpdateOptions{})
		// You have to return err itself here (not wrapped inside another error)
		// so that RetryOnConflict can identify it correctly.
		return err
	})
	if err != nil {
		// May be conflict if max retries were hit, or may be something unrelated
		// like permissions or a network error
		klog.Errorf("RetryOnConflict failed with: %s", err)
		return err
	}

	return nil
}

// Attempts to update a pv, retries if pv isn't fresh
func RetryUpdatePV(client clientset.Interface, pvName string, fn PVModification) error {
	err := retry.RetryOnConflict(CustomRetry, func() error {
		// Fetch the resource here; you need to refetch it on every try, since
		// if you got a conflict on the last update attempt then you need to get
		// the current version before making your own changes.
		pv, err := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("unable to get PV %s", pvName)
			return err
		}

		// Make whatever updates to the resource are needed
		pv = fn(pv)

		// Try to update
		_, err = client.CoreV1().PersistentVolumes().Update(context.TODO(), pv, metav1.UpdateOptions{})
		// You have to return err itself here (not wrapped inside another error)
		// so that RetryOnConflict can identify it correctly.
		return err
	})
	if err != nil {
		// May be conflict if max retries were hit, or may be something unrelated
		// like permissions or a network error
		klog.Errorf("RetryOnConflict failed with: %s", err)
		return err
	}

	return nil
}
