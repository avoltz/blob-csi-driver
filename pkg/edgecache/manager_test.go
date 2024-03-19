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
	"errors"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/blob_cache_volume"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/csi_mounts"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/mock_csi_mounts"
)

func TestGetStagingPath(t *testing.T) {
	assert.Regexp(t, `^hello[\\/]edgecache$`, GetStagingPath("hello"))
}

func TestSendMount(t *testing.T) {
	ctrl := gomock.NewController(t)
	successRsp := csi_mounts.AddMountRsp{}
	targetPath := "target/path"
	account := "account"
	container := "container"
	suffix := "my.url.org"
	volumeID := "test-id"
	vol := blob_cache_volume.Name{
		Account:       &account,
		Container:     &container,
		StorageSuffix: &suffix,
	}
	addReq := csi_mounts.AddMountReq{
		TargetPath: &targetPath,
		VolumeInfo: &csi_mounts.VolumeInfo{
			VolumeInfo: &csi_mounts.VolumeInfo_BlobVolume{
				BlobVolume: &vol,
			},
			VolumeId: &volumeID,
		},
	}
	interval := time.Duration(1 * time.Millisecond)
	timeout := time.Duration(200 * time.Millisecond)
	t.Run("WorksFirstTry", func(t *testing.T) {
		client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
		client.EXPECT().AddMount(gomock.Any(), &addReq).Times(1).Return(&successRsp, nil)
		ret := sendMount(client, account, container, suffix, volumeID, targetPath, interval, timeout)
		assert.Nil(t, ret)
	})
	t.Run("WorksWithRetries", func(t *testing.T) {
		client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
		gomock.InOrder(
			client.EXPECT().AddMount(gomock.Any(), &addReq).Return(nil, status.Errorf(codes.Internal, "")),
			client.EXPECT().AddMount(gomock.Any(), &addReq).Return(&successRsp, nil),
		)
		ret := sendMount(client, account, container, suffix, volumeID, targetPath, interval, timeout)
		assert.Nil(t, ret)
	})
	t.Run("CancelsEventually", func(t *testing.T) {
		client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
		client.EXPECT().AddMount(gomock.Any(), &addReq).MinTimes(1).Return(nil, status.Errorf(codes.DeadlineExceeded, ""))
		ret := sendMount(client, account, container, suffix, volumeID, targetPath, interval, timeout)
		assert.NotNil(t, ret)
	})
	// use large interval to force outer timeout cancellation
	t.Run("TimeoutCancels", func(t *testing.T) {
		client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
		client.EXPECT().AddMount(gomock.Any(), &addReq).Times(1).Return(nil, status.Errorf(codes.Internal, ""))
		err := sendMount(client, account, container, suffix, volumeID, targetPath, 1*time.Second, 10*time.Millisecond)
		if statusErr, ok := status.FromError(err); ok {
			assert.Equal(t, statusErr.Code(), codes.DeadlineExceeded)
		} else {
			assert.True(t, ok, "Unrecognized error returned from sendMount")
		}
	})
}

func TestSendUnmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	targetPath := "target/path"
	rmReq := csi_mounts.RemoveMountReq{
		TargetPath: &targetPath,
	}
	rmRsp := csi_mounts.RemoveMountRsp{}
	t.Run("Success", func(t *testing.T) {
		client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
		client.EXPECT().RemoveMount(gomock.Any(), &rmReq).Times(1).Return(&rmRsp, nil)
		ret := sendUnmount(client, targetPath)
		assert.Nil(t, ret)
	})
	t.Run("Fail", func(t *testing.T) {
		client := mock_csi_mounts.NewMockCSIMountsClient(ctrl)
		err := status.Errorf(codes.Internal, "")
		client.EXPECT().RemoveMount(gomock.Any(), &rmReq).Times(1).Return(nil, err)
		ret := sendUnmount(client, targetPath)
		assert.Equal(t, ret, err)
	})
}

func TestCallWithConnection(t *testing.T) {
	mgr := NewManager(5, "")
	t.Run("CalledReturnIsRetured", func(t *testing.T) {
		err := errors.New("hello")
		ret := mgr.callWithConnection(func(conn grpc.ClientConnInterface) error {
			return err
		}, "endpoint")
		assert.Equal(t, ret, err)
	})
	t.Run("CalledReturnNilIsReturned", func(t *testing.T) {
		ret := mgr.callWithConnection(func(conn grpc.ClientConnInterface) error {
			return nil
		}, "endpoint")
		assert.Nil(t, ret)
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
