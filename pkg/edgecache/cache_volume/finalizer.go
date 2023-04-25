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
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/util"
)

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
	var addFinalizers = func(inpvc *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
		if !util.ContainsString(inpvc.GetFinalizers(), finalizerName, nil) {
			pvcClone := inpvc.DeepCopy()
			pvcClone.ObjectMeta.Finalizers = append(pvcClone.ObjectMeta.Finalizers, finalizerName)
			return pvcClone
		}
		return inpvc
	}
	namespace := pv.Spec.ClaimRef.Namespace
	err := RetryUpdatePVC(client, namespace, pv.Spec.ClaimRef.Name, addFinalizers)
	if err != nil {
		klog.Errorf("Unable to addFinalizers to pvc %s", pv.Spec.ClaimRef.Name)
		return err
	}

	/* finally, update pv */
	var addFinalizersAndAnnotations = func(inpv *v1.PersistentVolume) *v1.PersistentVolume {
		pvClone := inpv.DeepCopy()
		pvClone.ObjectMeta.Finalizers = append(pvClone.ObjectMeta.Finalizers, finalizerName)
		pvClone.ObjectMeta.Annotations[accountAnnotation] = storageAccount
		pvClone.ObjectMeta.Annotations[containerAnnotation] = containerName
		return pvClone
	}
	err = RetryUpdatePV(client, pv.Name, addFinalizersAndAnnotations)
	if err != nil {
		klog.Errorf("unable to add protection finalizer and annotations to PV %s: %v", pv.Name, err)
		return err
	}
	klog.V(3).Infof("AddFinalizer: updated PV %s", pv.Name)
	return nil
}

func RemoveFinalizer(client clientset.Interface, pv *v1.PersistentVolume, pvc *v1.PersistentVolumeClaim) error {
	var removePVCFinalizers = func(inpvc *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
		pvcClone := inpvc.DeepCopy()
		pvcClone.ObjectMeta.Finalizers = util.RemoveString(pvcClone.ObjectMeta.Finalizers, finalizerName, nil)
		return pvcClone
	}
	if util.ContainsString(pvc.GetFinalizers(), finalizerName, nil) {
		klog.V(3).Infof("Removing protection finalizer from PVC %s", pvc.Name)
		err := RetryUpdatePVC(client, pvc.Namespace, pvc.Name, removePVCFinalizers)
		if err != nil {
			klog.Errorf("Unable to remove protection finalizer to PVC %s: %v", pvc.Name, err)
			return err
		}
	}

	klog.V(3).Infof("Removing protection finalizer from PV %s", pv.Name)
	if !util.ContainsString(pv.GetFinalizers(), finalizerName, nil) {
		return nil
	}

	var removePVFinalizers = func(inpv *v1.PersistentVolume) *v1.PersistentVolume {
		pvClone := pv.DeepCopy()
		pvClone.ObjectMeta.Finalizers = util.RemoveString(pvClone.ObjectMeta.Finalizers, finalizerName, nil)
		return pvClone
	}
	err := RetryUpdatePV(client, pv.Name, removePVFinalizers)
	if err != nil {
		klog.Errorf("unable to remove protection finalizer from PV %s: %v", pv.Name, err)
		return err
	}

	klog.V(3).Infof("RemoveFinalizer updated PV %s", pv.Name)
	return nil
}
