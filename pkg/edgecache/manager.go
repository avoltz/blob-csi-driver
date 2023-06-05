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
	"context"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/blob_cache_volume"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/csi_mounts"
)

type Manager struct {
	connectTimeout int
	mountEndpoint  string
}

type ManagerInterface interface {
	MountVolume(account string, container string, targetPath string) error
	UnmountVolume(volumeID string, targetPath string) error
}

func NewManager(connectTimeout int, mountEndpoint string) *Manager {
	return &Manager{
		connectTimeout: connectTimeout,
		mountEndpoint:  mountEndpoint,
	}
}

func GetStagingPath(path string) string {
	/*
		Use a special suffix for staging mounts during NodeStageVolume/NodeUnstageVolume

		During NodeUnstageVolume, typically umount is used to teardown the node mount.
		Cache mounts require special mount/unmount GRPC calls via csi_mounts.

		Cache volumes are indicated during staging by a property, but these properties are not
		included in the NodeUnstageVolume request, so this suffix is used by cache volumes
		during staging so that during unstaging we can check the same location and handle
		unmount using GRPC instead of the default path.

		If edgecache were a standalone CSI driver we could drop this suffix.
	*/
	return filepath.Join(path, "edgecache")
}

func sendMount(client csi_mounts.CSIMountsClient, account string, container string, targetPath string, interval time.Duration, timeout time.Duration) error {
	blobVolume := blob_cache_volume.Name{
		Account:   &account,
		Container: &container,
	}
	addReq := csi_mounts.AddMountReq{
		TargetPath: &targetPath,
		VolumeInfo: &csi_mounts.VolumeInfo{
			VolumeInfo: &csi_mounts.VolumeInfo_BlobVolume{
				BlobVolume: &blobVolume,
			},
		},
	}

	mntctx, mntcancel := context.WithCancel(context.Background())
	defer mntcancel()

	// There can be a delay between CreateBlob and the mount being available to the Mount Service
	// Use a goroutine to try a few times
	result := make(chan bool)
	go func() {
		for {
			klog.V(3).Infof("AddMount: %s, %s, %s", account, container, targetPath)
			_, err := client.AddMount(context.TODO(), &addReq)
			if err != nil {
				klog.Warningf("AddMount GRPC failed (will retry) returned with an error: %v", err)
				time.Sleep(interval)
			} else {
				result <- true
				return
			}
			if mntctx.Err() != nil {
				klog.Errorf("AddMount cancelled %q", mntctx.Err().Error())
				return
			}
		}
	}()
	select {
	case <-result:
		klog.V(3).Infof("AddMount: succeeded for %s/%s", account, container)
	case <-time.After(timeout):
		return status.Errorf(codes.DeadlineExceeded, "Deadline exceeded for mount %q", targetPath)
	}
	return nil
}

func sendUnmount(client csi_mounts.CSIMountsClient, targetPath string) error {
	rmReq := csi_mounts.RemoveMountReq{
		TargetPath: &targetPath,
	}

	klog.V(3).Infof("RemoveMount: %s", targetPath)
	if _, err := client.RemoveMount(context.TODO(), &rmReq); err != nil {
		klog.Errorf("GRPC call returned with an error: %v", err)
		return err
	}
	return nil
}

type ConnectionUsingFunc func(conn grpc.ClientConnInterface) error

func (m *Manager) callWithConnection(fun ConnectionUsingFunc, endpoint string) error {
	connectionTimeout := time.Duration(m.connectTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	// do not use blocking here so that we can unit test without a server / no bufconn required
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()
	return fun(conn)
}

func (m *Manager) MountVolume(account string, container string, targetPath string) error {
	return m.callWithConnection(func(conn grpc.ClientConnInterface) error {
		return sendMount(csi_mounts.NewCSIMountsClient(conn), account, container, targetPath, 500*time.Millisecond, 5*time.Second)
	}, m.mountEndpoint)
}

func (m *Manager) UnmountVolume(volumeID string, targetPath string) error {
	return m.callWithConnection(func(conn grpc.ClientConnInterface) error {
		return sendUnmount(csi_mounts.NewCSIMountsClient(conn), targetPath)
	}, m.mountEndpoint)
}
