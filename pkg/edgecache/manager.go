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
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/cache_volume_service"
	"sigs.k8s.io/blob-csi-driver/pkg/edgecache/csi_mounts"
)

type Manager struct {
	connectTimeout int
	configEndpoint string
	mountEndpoint  string
}

func NewManager(connectTimeout int, configEndpoint string, mountEndpoint string) *Manager {
	return &Manager{
		connectTimeout: connectTimeout,
		configEndpoint: configEndpoint,
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

		If edge cache were a standalone CSI driver we could drop this suffix.
	*/
	return filepath.Join(path, "edgecache")
}

func sendGetVolume(client cache_volume_service.CacheVolumeClient, account string, container string) (bool, error) {
	blobVolumeName := blob_cache_volume.Name{
		Account:   &account,
		Container: &container,
	}

	getReq := cache_volume_service.GetBlobRequest{
		Names: []*blob_cache_volume.Name{&blobVolumeName},
	}

	klog.V(2).Infof("calling edge cache GetBlob")
	getRsp, err := client.GetBlob(context.TODO(), &getReq)
	if err != nil {
		klog.Errorf("GRPC call returned with an error: %v", err)
		return false, err
	}
	klog.V(2).Infof("GetBlob found %d volumes", len(getRsp.Volumes))
	found := len(getRsp.Volumes) > 0
	return found, err
}

func sendCreateVolume(client cache_volume_service.CacheVolumeClient, account string, container string, key string) error {
	authenticator := blob_cache_volume.Authenticator{
		Authenticator: &blob_cache_volume.Authenticator_AccountKey{AccountKey: key},
	}

	blobVolume := blob_cache_volume.BlobCacheVolume{
		Name: &blob_cache_volume.Name{
			Account:   &account,
			Container: &container,
		},
		Auth: &authenticator,
	}

	addReq := cache_volume_service.CreateBlobRequest{
		Volume: &blobVolume,
	}

	klog.V(2).Infof("calling edge cache CreateBlob")
	_, err := client.CreateBlob(context.TODO(), &addReq)
	if err != nil {
		klog.Errorf("GRPC call returned with an error: %v", err)
		return err
	}
	klog.V(2).Infof("CreateBlob succeeded")
	return nil
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
	klog.V(2).Infof("Edgecache AddMount: %s", &addReq)
	go func() {
		for {
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
		klog.V(2).Infof("edge cache AddMount success")
	case <-time.After(timeout):
		return status.Errorf(codes.DeadlineExceeded, "Deadline exceeded for mount %q", targetPath)
	}
	return nil
}

func sendUnmount(client csi_mounts.CSIMountsClient, targetPath string) error {
	rmReq := csi_mounts.RemoveMountReq{
		TargetPath: &targetPath,
	}

	klog.V(2).Infof("Calling RemoveMount: %s", &rmReq)
	if _, err := client.RemoveMount(context.TODO(), &rmReq); err != nil {
		klog.Errorf("GRPC call returned with an error: %v", err)
		return err
	}
	return nil
}

func createVolume(client cache_volume_service.CacheVolumeClient, accountName string, accountKey string, containerName string) error {
	if found, err := sendGetVolume(client, accountName, containerName); err != nil {
		return err
	} else if !found {
		klog.V(2).Infof("volume does not exist, creating...")
		return sendCreateVolume(client, accountName, containerName, accountKey)
	}
	return nil
}

func (m *Manager) EnsureVolume(accountName string, accountKey string, containerName string, targetPath string) error {
	connectionTimeout := time.Duration(m.connectTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, m.configEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := cache_volume_service.NewCacheVolumeClient(conn)
	return createVolume(client, accountName, accountKey, containerName)
}

func (m *Manager) MountVolume(account string, container string, targetPath string) error {
	connectionTimeout := time.Duration(m.connectTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, m.mountEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := csi_mounts.NewCSIMountsClient(conn)
	return sendMount(client, account, container, targetPath, 500*time.Millisecond, 5*time.Second)
}

func (m *Manager) UnmountVolume(volumeID string, targetPath string) error {
	connectionTimeout := time.Duration(m.connectTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, m.mountEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := csi_mounts.NewCSIMountsClient(conn)
	return sendUnmount(client, targetPath)
}
