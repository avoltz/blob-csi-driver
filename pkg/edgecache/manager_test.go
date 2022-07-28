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
package edgecache

import (
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/blob_cache_volume"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/cache_volume_service"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/csi_mounts"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/mock_cache_volume_service"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/mock_csi_mounts"
)

func NewFakeManager() *Manager {
	return NewManager(5, "", "")
}

func TestNewFakeManager(t *testing.T) {
	m := NewFakeManager()
	assert.NotNil(t, m)
}

func TestCreateVolume(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_cache_volume_service.NewMockCacheVolumeClient(ctrl)
	account := "account"
	container := "container"
	key := "key"
	name := blob_cache_volume.Name{
		Account:   &account,
		Container: &container,
	}
	getReq := cache_volume_service.GetBlobRequest{
		Names: []*blob_cache_volume.Name{&name},
	}
	rspVolume := blob_cache_volume.BlobCacheVolume{Name: &name}
	authenticator := blob_cache_volume.Authenticator{
		Authenticator: &blob_cache_volume.Authenticator_AccountKey{AccountKey: key},
	}
	createReq := cache_volume_service.CreateBlobRequest{
		Volume: &blob_cache_volume.BlobCacheVolume{
			Name: &name,
			Auth: &authenticator,
		},
	}
	createRsp := cache_volume_service.CreateBlobResponse{}
	t.Run("NoBlobFoundWillCreate", func(t *testing.T) {
		getRsp := cache_volume_service.GetBlobResponse{}
		client.EXPECT().GetBlob(gomock.Any(), &getReq).Times(1).Return(&getRsp, nil)
		client.EXPECT().CreateBlob(gomock.Any(), &createReq).Times(1).Return(&createRsp, nil)
		err := createVolume(client, account, key, container)
		assert.Nil(t, err)

	})
	t.Run("BlobFoundDoesNotCreate", func(t *testing.T) {
		getRsp := cache_volume_service.GetBlobResponse{
			Volumes: []*blob_cache_volume.BlobCacheVolume{&rspVolume},
		}
		client.EXPECT().GetBlob(gomock.Any(), &getReq).Times(1).Return(&getRsp, nil)
		client.EXPECT().CreateBlob(gomock.Any(), gomock.Any()).Times(0)
		err := createVolume(client, account, key, container)
		assert.Nil(t, err)
	})
}

func TestSendMount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
	successRsp := csi_mounts.AddMountRsp{}
	targetPath := "target/path"
	account := "account"
	container := "container"
	vol := blob_cache_volume.Name{
		Account:   &account,
		Container: &container,
	}
	addReq := csi_mounts.AddMountReq{
		TargetPath: &targetPath,
		VolumeInfo: &csi_mounts.VolumeInfo{
			VolumeInfo: &csi_mounts.VolumeInfo_BlobVolume{
				BlobVolume: &vol,
			},
		},
	}
	interval := time.Duration(1 * time.Microsecond)
	timeout := time.Duration(10 * time.Millisecond)
	t.Run("WorksFirstTry", func(t *testing.T) {
		client.EXPECT().AddMount(gomock.Any(), &addReq).Times(1).Return(&successRsp, nil)
		ret := sendMount(client, account, container, targetPath, interval, timeout)
		assert.Nil(t, ret)
	})
	t.Run("WorksWithRetries", func(t *testing.T) {
		gomock.InOrder(
			client.EXPECT().AddMount(gomock.Any(), &addReq).Return(nil, status.Errorf(codes.Internal, "")),
			client.EXPECT().AddMount(gomock.Any(), &addReq).Return(&successRsp, nil),
		)
		ret := sendMount(client, account, container, targetPath, interval, timeout)
		assert.Nil(t, ret)
	})
	t.Run("CancelsEventually", func(t *testing.T) {
		client.EXPECT().AddMount(gomock.Any(), &addReq).MinTimes(1).Return(nil, status.Errorf(codes.DeadlineExceeded, ""))
		ret := sendMount(client, account, container, targetPath, interval, timeout)
		assert.NotNil(t, ret)
	})
}

func TestSendUnmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
	targetPath := "target/path"
	rmReq := csi_mounts.RemoveMountReq{
		TargetPath: &targetPath,
	}
	rmRsp := csi_mounts.RemoveMountRsp{}
	t.Run("Success", func(t *testing.T) {
		client.EXPECT().RemoveMount(gomock.Any(), &rmReq).Times(1).Return(&rmRsp, nil)
		ret := sendUnmount(client, targetPath)
		assert.Nil(t, ret)
	})
	t.Run("Fail", func(t *testing.T) {
		err := status.Errorf(codes.Internal, "")
		client.EXPECT().RemoveMount(gomock.Any(), &rmReq).Times(1).Return(nil, err)
		ret := sendUnmount(client, targetPath)
		assert.Equal(t, ret, err)
	})
}

func TestMain(m *testing.M) {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "false")
	_ = flag.Set("alsologtostderr", "false")
	_ = flag.Set("stderrthreshold", "10")
	klog.SetOutput(io.Discard)
	os.Exit(m.Run())
}
