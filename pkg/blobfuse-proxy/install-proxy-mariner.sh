#!/bin/sh

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -xe

# install blobfuse/blobfuse2
echo "skip install blobfuse/blobfuse2 for mariner...."

# install blobfuse-proxy
updateBlobfuseProxy="true"
if [ -f "/host/usr/bin/blobfuse-proxy" ];then
  old=$(sha256sum /host/usr/bin/blobfuse-proxy | awk '{print $1}')
  new=$(sha256sum /blobfuse-proxy/blobfuse-proxy | awk '{print $1}')
  if [ "$old" = "$new" ];then
    updateBlobfuseProxy="false"
    echo "no need to update blobfuse-proxy"
  fi
fi
if [ "$updateBlobfuseProxy" = "true" ];then
  echo "copy blobfuse-proxy...."
  rm -rf /host/var/lib/kubelet/plugins/blob.csi.azure.com/blobfuse-proxy.sock
  rm -rf /host/usr/bin/blobfuse-proxy
  cp /blobfuse-proxy/blobfuse-proxy /host/usr/bin/blobfuse-proxy
  chmod 755 /host/usr/bin/blobfuse-proxy
fi

updateService="true"
if [ -f "/host/usr/lib/systemd/system/blobfuse-proxy.service" ];then
  old=$(sha256sum /host/usr/lib/systemd/system/blobfuse-proxy.service | awk '{print $1}')
  new=$(sha256sum /blobfuse-proxy/blobfuse-proxy.service | awk '{print $1}')
  if [ "$old" = "$new" ];then
      updateService="false"
      echo "no need to update blobfuse-proxy.service"
  fi
fi
if [ "$updateService" = "true" ];then
  echo "copy blobfuse-proxy.service...."
  mkdir -p /host/usr/lib/systemd/system
  cp /blobfuse-proxy/blobfuse-proxy.service /host/usr/lib/systemd/system/blobfuse-proxy.service
fi

if [ "${INSTALL_BLOBFUSE_PROXY}" = "true" ];then
  if [ "$updateBlobfuseProxy" = "true" ] || [ "$updateService" = "true" ];then
    echo "start blobfuse-proxy...."
    $HOST_CMD systemctl daemon-reload
    $HOST_CMD systemctl enable blobfuse-proxy.service
    $HOST_CMD systemctl restart blobfuse-proxy.service
  fi
fi
