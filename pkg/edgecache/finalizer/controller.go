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
)


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

// NewEdgeCacheFinalizerController returns a new *Controller.
func NewEdgeCacheFinalizerController(manager edgecache.ManagerInterface, pvcInformer coreinformers.PersistentVolumeClaimInformer, client clientset.Interface) *Controller {
	c := &Controller{
		client: client,
		queue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "edgecachefinalizer"),
	}
	if client == nil {
		klog.Error("edgecache finalizer requires client")
		return nil
	}
	if manager == nil {
		klog.Error("edgecache finalizer requires edgecache manager")
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
	c.eventRecorder = broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "edgecachefinalizer"})

	klog.V(3).Infof("NewEdgecacheFinalizerController: complete")
	return c
}

// Run runs the controller goroutines.
func (c *Controller) Run(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.V(3).Infof("Starting edgecache finalizer controller")
	defer klog.V(3).Infof("Shutting down edgecache finalizer controller")

	if !cache.WaitForNamedCacheSync("edgecachefinalizer", ctx.Done(), c.pvcListerSynced) {
		return
	}

	klog.V(3).Infof("edgecache finalizer controller waiting on workers")
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
		klog.V(3).Info("edgecache finalizer controller processed next item")
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
	// ignore pvs not using edgecache or which do not have the finalizer
	if !util.ContainsString(pv.GetFinalizers(), finalizerName, nil) {
		return nil
	}
	// wait for the PVC to be deleted before cleaning up the volume
	if pvc.GetDeletionTimestamp() == nil {
		klog.V(3).Infof("PVC is not being deleted, ignoring")
		return nil
	}
	// We always add the finalizer from node plugin so might as well use an annotation here!
	err = c.ecManager.DeleteVolume(pv.ObjectMeta.Annotations[accountAnnotation], pv.ObjectMeta.Annotations[containerAnnotation])
	if err != nil {
		return err
	}
	klog.V(2).Infof("edgecache volume deleted, removing finalizer")
	// edgecache delete success, remove the finalizer
	return RemoveFinalizer(c.client, pv, pvc)
}

func (c *Controller) pvcUpdated(a interface{}, b interface{}) {
	newpvc, ok := b.(*v1.PersistentVolumeClaim)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("PVC informer returned non-PVC object: %#v", b))
		return
	}
	// ignore pvcs not using edgecache or which already have no finalizer
	if !util.ContainsString(newpvc.GetFinalizers(), finalizerName, nil) {
		return
	}
	// ignore any pvcs which still have attached pods
	if util.ContainsString(newpvc.GetFinalizers(), pvcProtectionFinalizer, nil) {
		return
	}

	c.queue.Add(newpvc)
}
