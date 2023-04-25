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
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache"
	"sigs.k8s.io/blob-csi-driver/pkg/util"
)

const (
	finalizerName          string = "external/blob-csi-driver-edgecache"
	pvcProtectionFinalizer string = "kubernetes.io/pvc-protection"
	accountAnnotation      string = "external/edgecache-account"
	containerAnnotation    string = "external/edgecache-container"
    createVolumeAnnotation string = "external/edgecache-create-volume"
    secretNameAnnotation   string = "external/edgecache-secret-name"
    secretKeyField         string = "azurestorageaccountkey"
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
		if pv.Spec.CSI.VolumeHandle == volumeID {
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

func GetPVCByName(client clientset.Interface, namespace string, pvcName string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("unable to get PVC %s", pvcName)
		return nil, err
	}
	return pvc, nil
}

// Attempts to update a pvc, retries if pvc isn't fresh
func RetryUpdatePVC(client clientset.Interface, namespace string, pvcName string, fn PVCModification) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
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
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
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

// Controller is controller that removes PVProtectionFinalizer
// from PVs that are not bound to PVCs.
type Controller struct {
	client clientset.Interface

	pvcLister       corelisters.PersistentVolumeClaimLister
	pvcListerSynced cache.InformerSynced

	queue workqueue.RateLimitingInterface

	eventRecorder record.EventRecorder

	ecManager edgecache.ManagerInterface
}

// NewEdgeCacheCVController returns a new *Controller.
func NewEdgeCacheCVController(manager edgecache.ManagerInterface, pvcInformer coreinformers.PersistentVolumeClaimInformer, client clientset.Interface) *Controller {
	c := &Controller{
		client: client,
		queue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "edgecachecvcontroller"),
	}
	if client == nil {
		klog.Error("edgecache cv controller requires client")
		return nil
	}
	if manager == nil {
		klog.Error("edgecache cv controller requires edgecache manager")
		return nil
	}

	c.ecManager = manager
	c.pvcLister = pvcInformer.Lister()
	c.pvcListerSynced = pvcInformer.Informer().HasSynced

	_, err := pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.pvcUpdated,
	})
	if err != nil {
		klog.Error("edgecache unable to add eventhandler")
		return nil
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartLogging(klog.Infof)
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: client.CoreV1().Events(v1.NamespaceAll)})
	c.eventRecorder = broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "edgecachecvcontroller"})

	klog.V(3).Infof("NewEdgecacheCVController: complete")
	return c
}

// Run runs the controller goroutines.
func (c *Controller) Run(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.V(3).Infof("Starting edgecachecv controller")
	defer klog.V(3).Infof("Shutting down edgecachecv controller")

	if !cache.WaitForNamedCacheSync("edgecachecvcontroller", ctx.Done(), c.pvcListerSynced) {
		return
	}

	klog.V(3).Infof("edgecache cv controller waiting on workers")
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
		klog.V(3).Info("edgecache cv controller processed next item")
	}
}

// processNextWorkItem deals with one pvcKey off the queue.  It returns false when it's time to quit.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	obj, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(obj)
	var err error

	pvc := obj.(*v1.PersistentVolumeClaim)
	err = c.processPVC(pvc)
	if err == nil {
		c.queue.Forget(obj)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("VolumeEvent %v failed with : %w", obj, err))
	c.queue.AddRateLimited(obj)

	return true
}

// processPVC is called on newly added pvcs to the queue
func (c *Controller) processPVC(pvc *v1.PersistentVolumeClaim) error {
	pvName := pvc.Spec.VolumeName
	klog.V(3).Infof("Processing PVC %s:%s for PV %s", pvc.Namespace, pvc.Name, pvName)
	// track the duration
	startTime := time.Now()
	defer klog.V(3).Infof("Finished processing (%v)", pvName, time.Since(startTime))

	// get the latest pv object
	pv, err := c.client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		klog.Warningf("PV %s not found, ignoring", pvName)
		return nil
	}
	if err != nil {
		klog.Error(err)
		return err
	}

	// if PVC has create volume annotation
	_, createVolume := pvc.ObjectMeta.Annotations[createVolumeAnnotation]
	if createVolume {
		klog.V(3).Infof("Running create volume for PV %s", pvName)
		if err := c.createVolume(pv, pvc); err != nil {
			klog.Error(err)
			return err
		}
		return nil
	}

	// ignore pvs not using edgecache or which do not have the finalizer
	if !util.ContainsString(pv.GetFinalizers(), finalizerName, nil) {
		klog.V(3).Infof("does not have finalizers %v", pvc.Annotations)
		return nil
	}
	// wait for the PVC to be deleted before cleaning up the volume
	if pvc.GetDeletionTimestamp() == nil {
		klog.V(3).Infof("PVC is not being deleted, ignoring")
		return nil
	}

	// volume is ready to be deleted
	err = c.deleteVolume(pv, pvc)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) pvcUpdated(a interface{}, b interface{}) {
	newpvc, ok := b.(*v1.PersistentVolumeClaim)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("PVC informer returned non-PVC object: %#v", b))
		return
	}

	// if PVC has create volume annotation
	_, createVolume := newpvc.ObjectMeta.Annotations[createVolumeAnnotation]
	if createVolume {
		c.queue.Add(newpvc)
		return
	}

	// ignore pvcs not using edgecache or which already have no finalizer
	if !util.ContainsString(newpvc.GetFinalizers(), finalizerName, nil) {
		klog.V(3).Infof("does not have finalizers or create volume annotation %v", newpvc.Name)
		return
	}
	// ignore any pvcs which still have attached pods
	if util.ContainsString(newpvc.GetFinalizers(), pvcProtectionFinalizer, nil) {
		return
	}

	c.queue.Add(newpvc)
}

// Finalizes the PV/PVC and calls EnsureVolume
func (c *Controller) createVolume(pv *v1.PersistentVolume, pvc *v1.PersistentVolumeClaim) error {
	// Remove create volume annotation so lister stops queueing
	var removeAnnotations = func(inpvc *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
		pvcClone := inpvc.DeepCopy()
		delete(pvcClone.ObjectMeta.Annotations, createVolumeAnnotation)
		return pvcClone
	}
	err := RetryUpdatePVC(c.client, pvc.Namespace, pvc.Name, removeAnnotations)
	if err != nil {
		klog.Error(err)
		return err
	}

	// Get volume attributes
	accountName, accountNameOk := pvc.ObjectMeta.Annotations[accountAnnotation]
	containerName, containerNameOk := pvc.ObjectMeta.Annotations[containerAnnotation]
	if !accountNameOk || !containerNameOk {
		klog.Errorf("Account or Container annotation missing from PVC %s", pvc.Name)
		return nil
	}

	// Finalize PV and PVC
	if err := AddFinalizer(c.client, pv, accountName, containerName); err != nil {
		klog.Error(err)
		return err
	}

	// Get storage account secret
	accountSecretName := pvc.ObjectMeta.Annotations[secretNameAnnotation]
	accountSecret, err := c.client.CoreV1().Secrets("default").Get(context.Background(), accountSecretName, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}
	accountKey := string(accountSecret.Data[secretKeyField])

	klog.V(3).Infof("Calling EnsureVolume with acctName %s, acctKey %s, contName %s", accountName, accountKey, containerName)
	// Create the Volume
	if err := c.ecManager.EnsureVolume(accountName, accountKey, containerName); err != nil {
		klog.Error(err)
		return err
	}

	klog.V(3).Infof("createVolume successfully created volume and add finalizers!")
	return nil
}

// Calls DeleteVolume and removes finalizers from the PV/PVC
func (c *Controller) deleteVolume(pv *v1.PersistentVolume, pvc *v1.PersistentVolumeClaim) error {
	// We always add the finalizer from node plugin so might as well use an annotation here!
	if err := c.ecManager.DeleteVolume(pv.ObjectMeta.Annotations[accountAnnotation], pv.ObjectMeta.Annotations[containerAnnotation]); err != nil {
		return err
	}
	klog.V(2).Infof("edgecache volume deleted, removing finalizer")
	// edgecache delete success, remove the finalizer
	if err := RemoveFinalizer(c.client, pv, pvc); err != nil {
		klog.Error("RemoveFinalizer failed")
		return err
	}

	klog.V(2).Infof("deleteVolume successfully removed finalizer and cleaned up volume")
	return nil
}
