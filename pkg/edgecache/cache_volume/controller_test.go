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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/mock_manager"
)

func pvcWithVolume(p *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
	p.Spec = v1.PersistentVolumeClaimSpec{
		VolumeName: defaultPVName,
	}
	return p
}

func pvcWithDeletion(p *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
	p.DeletionTimestamp = &metav1.Time{}
	return p
}

func pvcWithFinalizers(p *v1.PersistentVolumeClaim, finalizers []string) *v1.PersistentVolumeClaim {
	p.SetFinalizers(finalizers)
	return p
}

func pvcWithAnnotations(p *v1.PersistentVolumeClaim, annotations map[string]string) *v1.PersistentVolumeClaim {
	p.SetAnnotations(annotations)
	return p
}

func pvWithFinalizers(p *v1.PersistentVolume, finalizers []string) *v1.PersistentVolume {
	p.SetFinalizers(finalizers)
	return p
}

func secret(name string, namespace string, secret map[string][]byte) *v1.Secret {
	return &v1.Secret{
		Data: secret,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewFakeCVController(client *fake.Clientset, mgr *mock_manager.MockManagerInterface) (*Controller, informers.SharedInformerFactory) {
	klog.Info("Creating fake finalizer")
	if client == nil {
		client = fake.NewSimpleClientset()
	}
	informers := informers.NewSharedInformerFactory(client, 0)
	pvcInformer := informers.Core().V1().PersistentVolumeClaims()
	return NewEdgeCacheCVController(mgr, pvcInformer, client), informers
}

func TestFinalizerRequires(t *testing.T) {
	mgr := edgecache.NewManager(1, "", "")
	client := fake.NewSimpleClientset()
	informers := informers.NewSharedInformerFactory(client, 0)
	pvcInformer := informers.Core().V1().PersistentVolumeClaims()
	t.Run("KubeClient", func(t *testing.T) {
		assert.Nil(t, NewEdgeCacheCVController(mgr, pvcInformer, nil))
	})
	t.Run("EdgeCacheManager", func(t *testing.T) {
		assert.Nil(t, NewEdgeCacheCVController(nil, pvcInformer, client))
	})
}

func TestNewFakeCVControllerNotSynced(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl, informers := NewFakeCVController(nil, nil)
	informers.Start(ctx.Done())
	go ctrl.Run(ctx, 2)
}

func TestRunWorker(t *testing.T) {
	client := fake.NewSimpleClientset()
	c := gomock.NewController(t)
	mgr := mock_manager.NewMockManagerInterface(c)
	ctrl, _ := NewFakeCVController(client, mgr)
	assert.NotNil(t, ctrl)
	ctrl.queue.Add(pvc())
	go ctrl.runWorker(context.Background())
	ctrl.queue.ShutDownWithDrain()
}

type TestCase struct {
	name string
	pv   *v1.PersistentVolume
	pvc  *v1.PersistentVolumeClaim
}

func TestProcessPVCSkip(t *testing.T) {
	testcases := []TestCase{
		{
			name: "PVCNotFoundPV",
			pv:   nil,
			pvc:  pvc(),
		},
		{
			name: "NoFinalizer",
			pv:   pv(),
			pvc:  pvcWithVolume(pvc()),
		},
		{
			name: "FinalizerNotBeingDeleted",
			pv:   pvWithFinalizers(pv(), []string{finalizerName}),
			pvc:  pvcWithFinalizers(pvcWithVolume(pvc()), []string{finalizerName}),
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			objs := []runtime.Object{testcase.pvc}
			if testcase.pv != nil {
				objs = append(objs, testcase.pv)
			}
			client := fake.NewSimpleClientset(objs...)
			c := gomock.NewController(t)
			mgr := mock_manager.NewMockManagerInterface(c)
			ctrl, _ := NewFakeCVController(client, mgr)
			assert.NotNil(t, ctrl)
			assert.Nil(t, ctrl.processPVC(testcase.pvc))
			mgr.EXPECT().DeleteVolume(gomock.Any, gomock.Any()).Times(0)
			mgr.EXPECT().EnsureVolume(gomock.Any, gomock.Any, gomock.Any).Times(0)
		})
	}
}

func TestProcessPVCDeleteVolume(t *testing.T) {
	pvc1 := pvcWithDeletion(pvcWithFinalizers(pvcWithVolume(pvc()), []string{finalizerName}))
	pv1 := pvWithFinalizers(pv(), []string{finalizerName})
	pv1.Annotations[accountAnnotation] = "account"
	pv1.Annotations[containerAnnotation] = "container"
	client := fake.NewSimpleClientset(pv1, pvc1)
	c := gomock.NewController(t)
	mgr := mock_manager.NewMockManagerInterface(c)
	ctrl, _ := NewFakeCVController(client, mgr)
	assert.NotNil(t, ctrl)
	mgr.EXPECT().DeleteVolume("account", "container").Times(1).Return(nil)
	assert.Nil(t, ctrl.processPVC(pvc1))
	pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), pv1.Name, metav1.GetOptions{})
	assert.Empty(t, pvAfter.GetFinalizers())
	pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), pvc1.Name, metav1.GetOptions{})
	assert.Empty(t, pvcAfter.GetFinalizers())
}

func TestPVCUpdated(t *testing.T) {
	client := fake.NewSimpleClientset()
	c := gomock.NewController(t)
	mgr := mock_manager.NewMockManagerInterface(c)
	t.Run("IgnoreNoECFinalizer", func(t *testing.T) {
		ctrl, _ := NewFakeCVController(client, mgr)
		ctrl.pvcUpdated(pvc(), pvcWithFinalizers(pvc(), []string{"hello", "world", pvcProtectionFinalizer}))
		assert.Empty(t, ctrl.queue.Len())
	})
	t.Run("IgnoreProtected", func(t *testing.T) {
		ctrl, _ := NewFakeCVController(client, mgr)
		ctrl.pvcUpdated(pvc(), pvcWithFinalizers(pvc(), []string{finalizerName, pvcProtectionFinalizer}))
		assert.Empty(t, ctrl.queue.Len())
	})
	t.Run("ProcessValidWithFinalizers", func(t *testing.T) {
		ctrl, _ := NewFakeCVController(client, mgr)
		ctrl.pvcUpdated(pvc(), pvcWithFinalizers(pvc(), []string{finalizerName}))
		assert.Equal(t, ctrl.queue.Len(), 1)
	})

	t.Run("ProcessValidWithAnnotations", func(t *testing.T) {
		ctrl, _ := NewFakeCVController(client, mgr)
		createVolumeAnno := map[string]string{createVolumeAnnotation: "yes"}
		ctrl.pvcUpdated(pvc(), pvcWithAnnotations(pvc(), createVolumeAnno))
		assert.Equal(t, ctrl.queue.Len(), 1)
	})
}

func TestProcessPVCCreateVolume(t *testing.T) {
	pvBefore := pv()

	accountName := "acct"
	accountKey := "shhh"
	secretName := "i am secret"
	secretNamespace := "i am secret space"
	containerName := "contain me"
	markUpAnnotations := map[string]string{
		createVolumeAnnotation:    "yes",
		secretNameAnnotation:      secretName,
		secretNamespaceAnnotation: secretNamespace,
		accountAnnotation:         accountName,
		containerAnnotation:       containerName,
	}
	pvcBefore := pvcWithAnnotations(pvc(), markUpAnnotations)

	volumeSecret := secret(secretName, secretNamespace, map[string][]byte{
		secretKeyField: []byte(accountKey),
	})

	c := gomock.NewController(t)
	mgr := mock_manager.NewMockManagerInterface(c)
	client := fake.NewSimpleClientset(pvBefore, pvcBefore, volumeSecret)
	ctrl, _ := NewFakeCVController(client, mgr)
	assert.NotNil(t, ctrl)

	mgr.EXPECT().EnsureVolume(accountName, accountKey, containerName).Times(1).Return(nil)

	assert.Nil(t, ctrl.processPVC(pvcBefore))

	expectedFinalizers := []string{finalizerName}
	pvAfter, _ := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvBefore.Name, metav1.GetOptions{})
	assert.Equal(t, expectedFinalizers, pvAfter.GetFinalizers())
	pvcAfter, _ := client.CoreV1().PersistentVolumeClaims(defaultPVCNamespace).Get(context.TODO(), pvcBefore.Name, metav1.GetOptions{})
	assert.Equal(t, expectedFinalizers, pvcAfter.GetFinalizers())

}
