/*
Copyright 2017 The Kubernetes Authors.

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

package csicommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	typedv1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

const (
	NameSpaceEnvVar   = "KUBERNETES_NAMESPACE"
	PodNameEnvVar     = "POD_NAME"
	CSIEventSourceStr = "blob-csi-driver"
)

const (
	// Driver "Normal" event Reason list
	NodeStagingVolume      = "NodeStagingVolume"
	NodeStagedVolume       = "NodeStagedVolume"
	NodeUnStagingVolume    = "NodeUnStagingVolume"
	NodeUnStagedVolume     = "NodeUnStagedVolume"
	NodePublishingVolume   = "NodePublishingVolume"
	NodePublishedVolume    = "NodePublishedVolume"
	NodeUnPublishingVolume = "NodeUnPublishingVolume"
	NodeUnPublishedVolume  = "NodeUnPublishedVolume"
	CreatingBlobContainer  = "CreatingBlobContainer"
	CreatedBlobContainer   = "CreatedBlobContainer"
	DeletingBlobContainer  = "DeletingBlobContainer"
	DeletedBlobContainer   = "DeletedBlobContainer"
)

const (
	// Driver "Warning" event Reason list
	FailedToInitializeDriver = "Failed"
	FailedToProvisionVolume  = "Failed"
	FailedAuthentication     = "FailedAuthentication"
	InvalidAuthentication    = "InvalidAuthentication"
)

// Event correlation is done on the client side: need to use a global variable for the
// event broadcaster(eventBroadcaster)
// https://pkg.go.dev/k8s.io/client-go/tools/record#NewEventCorrelator
// https://pkg.go.dev/k8s.io/client-go/tools/record#EventCorrelator
var (
	eventBroadcaster         record.EventBroadcaster = nil // revive:disable:var-declaration
	eventBroadcasterInitLock sync.Mutex
)

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("Invalid endpoint: %v", ep)
}

func NewVolumeCapabilityAccessMode(mode csi.VolumeCapability_AccessMode_Mode) *csi.VolumeCapability_AccessMode {
	return &csi.VolumeCapability_AccessMode{Mode: mode}
}

func NewControllerServiceCapability(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: cap,
			},
		},
	}
}

func NewNodeServiceCapability(cap csi.NodeServiceCapability_RPC_Type) *csi.NodeServiceCapability {
	return &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: cap,
			},
		},
	}
}

func getLogLevel(method string) int32 {
	if method == "/csi.v1.Identity/Probe" ||
		method == "/csi.v1.Node/NodeGetCapabilities" ||
		method == "/csi.v1.Node/NodeGetVolumeStats" {
		return 6
	}
	return 2
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	level := klog.Level(getLogLevel(info.FullMethod))
	klog.V(level).Infof("GRPC call: %s", info.FullMethod)
	klog.V(level).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))

	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("GRPC error: %v", err)
	} else {
		klog.V(level).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
	}
	return resp, err
}

// Gets a Kubernetes client set.
func GetKubeClient(inCluster bool) (*kubernetes.Clientset, error) {
	config, err := GetKubeConfig(inCluster)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// Gets a Kubernetes config.
func GetKubeConfig(inCluster bool) (*rest.Config, error) {
	if inCluster {
		return getInClusterKubeConfig()
	}

	return getLocalKubeConfig()
}

// Gets the Kubernetes config for local use.
func getLocalKubeConfig() (*rest.Config, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// Gets the Kubernetes client for in-cluster use.
func getInClusterKubeConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error
	if config, err = rest.InClusterConfig(); err != nil {
		return nil, err
	}
	return config, nil
}

func SendKubeEvent(eventType string, reasonCode string, eventSource string, messageStr string) {
	var err error
	client, err := GetKubeClient(true)
	if err != nil {
		klog.Errorf(err.Error())
		return
	}

	nameSpace := os.Getenv(NameSpaceEnvVar)
	if nameSpace == "" {
		klog.Errorf("%s environment variable not set", NameSpaceEnvVar)
		return
	}

	podName := os.Getenv(PodNameEnvVar)
	if podName == "" {
		klog.Errorf("%s environment variable not set", PodNameEnvVar)
		return
	}

	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		klog.Errorf(err.Error())
		return
	}
	if eventBroadcaster == nil {
		eventBroadcasterInitLock.Lock()
		if eventBroadcaster == nil { // In case eventBroadcaster was just set
			eventBroadcaster = record.NewBroadcaster() // https://pkg.go.dev/k8s.io/client-go/tools/record#EventBroadcaster
			eventBroadcaster.StartLogging(klog.Infof)
			eventBroadcaster.StartRecordingToSink(&typedv1core.EventSinkImpl{Interface: client.CoreV1().Events(nameSpace)})
		}
		eventBroadcasterInitLock.Unlock()
	}

	pod, err := client.CoreV1().Pods(nameSpace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		return
	}

	eventRecorder := eventBroadcaster.NewRecorder(scheme, v1.EventSource{Component: eventSource})
	eventRecorder.Event(pod, eventType, reasonCode, messageStr)
}
