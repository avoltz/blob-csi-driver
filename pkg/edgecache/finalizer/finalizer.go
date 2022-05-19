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

package finalizer

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/util"
)

func GetPVByVolumeID(client clientset.Interface, volumeID string) (*v1.PersistentVolume, error) {
	klog.Infof("No pvName provided, looking up via volumeID: %s", volumeID)
	pvList, err := client.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Infof("unable to list volumes via volumeID: %s", volumeID)
		return nil, err
	}
	for _, pv := range pvList.Items {
		if pv.Spec.CSI.VolumeHandle == volumeID {
			return &pv, nil
		}
	}
	return nil, errors.New("no pv found")
}

func GetPVByName(client clientset.Interface, pvName string) (*v1.PersistentVolume, error) {
	pv, err := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err != nil {
		klog.V(3).Infof("Unable to get PV %s", pvName)
		return nil, err
	}
	return pv, nil
}

func AddFinalizer(client clientset.Interface, pv *v1.PersistentVolume, storageAccount string, containerName string) error {
	/* add to claim first, then pv
	We need on claim for dynamic pvc. deleting a pvc will trigger deleteVolume which then will delete az container
	so intercept pvc deletes
	*/
	klog.V(3).Infof("AddFinalizer called with pv: %s, account: %s, container: %s", pv, storageAccount, containerName)
	if util.ContainsString(pv.GetFinalizers(), finalizerName, nil) {
		// finalizer already present means we succeeded here already
		return nil
	}
	namespace := pv.Spec.ClaimRef.Namespace
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pv.Spec.ClaimRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.V(3).Infof("Unable to get PVC for pv %s", pv.Name)
		return err
	}
	if !util.ContainsString(pvc.GetFinalizers(), finalizerName, nil) {
		pvcClone := pvc.DeepCopy()
		pvcClone.ObjectMeta.Finalizers = append(pvcClone.ObjectMeta.Finalizers, finalizerName)
		_, err := client.CoreV1().PersistentVolumeClaims(namespace).Update(context.TODO(), pvcClone, metav1.UpdateOptions{})
		if err != nil {
			klog.V(3).Infof("Error adding protection finalizer to PVC %s: %v", pv.Name, err)
			return err
		}
		klog.V(3).Infof("Added protection finalizer to PVC %s", pvc.Name)
	} else {
		klog.Infof("pvc already has finalizer: %v", pvc.GetFinalizers())
	}
	/* finally, update pv */
	pvClone := pv.DeepCopy()
	pvClone.ObjectMeta.Finalizers = append(pvClone.ObjectMeta.Finalizers, finalizerName)
	pvClone.ObjectMeta.Annotations[accountAnnotation] = storageAccount
	pvClone.ObjectMeta.Annotations[containerAnnotation] = containerName
	_, err = client.CoreV1().PersistentVolumes().Update(context.TODO(), pvClone, metav1.UpdateOptions{})
	if err != nil {
		klog.V(3).Infof("Error adding protection finalizer and annotations to PV %s: %v", pv.Name, err)
		return err
	}
	klog.V(3).Infof("Updated PV %s", pv.Name)
	return nil
}

func RemoveFinalizer(client clientset.Interface, pv *v1.PersistentVolume, pvc *v1.PersistentVolumeClaim) error {
	if util.ContainsString(pvc.GetFinalizers(), finalizerName, nil) {
		klog.Infof("Removing finalizer from PVC %s:%s", pvc.Namespace, pvc.Name)
		pvcClone := pvc.DeepCopy()
		pvcClone.ObjectMeta.Finalizers = util.RemoveString(pvcClone.ObjectMeta.Finalizers, finalizerName, nil)
		if _, err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.TODO(), pvcClone, metav1.UpdateOptions{}); err != nil {
			klog.V(3).Infof("Error adding protection finalizer to PVC %s: %v", pv.Name, err)
			return err
		}
		klog.Infof("Removed finalizer from PVC %s:%s", pvc.Namespace, pvc.Name)
	} else {
		klog.Infof("RemoveFinalizer called on PVC without finalizer %s:%s", pvc.Namespace, pvc.Name)
	}

	klog.V(3).Infof("Removing protection finalizer from PV %s", pv.Name)
	if !util.ContainsString(pv.GetFinalizers(), finalizerName, nil) {
		return nil
	}
	pvClone := pv.DeepCopy()
	pvClone.ObjectMeta.Finalizers = util.RemoveString(pvClone.ObjectMeta.Finalizers, finalizerName, nil)
	if _, err := client.CoreV1().PersistentVolumes().Update(context.TODO(), pvClone, metav1.UpdateOptions{}); err != nil {
		klog.V(3).Infof("Error removing protection finalizer from PV %s: %v", pv.Name, err)
		return err
	}
	klog.V(3).Infof("Removed protection finalizer from PV %s", pv.Name)
	return nil
}
