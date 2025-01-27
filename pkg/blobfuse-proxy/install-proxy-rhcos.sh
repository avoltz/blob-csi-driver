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

# in coreos, we could just copy the blobfuse2 binary to /usr/local/bin/blobfuse2
if [ "${INSTALL_BLOBFUSE}" = "true" ] || [ "${INSTALL_BLOBFUSE2}" = "true" ]
then
  echo "copy blobfuse2...."
  cp /usr/bin/blobfuse2 /host/usr/local/bin/blobfuse2
fi

# install blobfuse-proxy
updateBlobfuseProxy="true"
if [ -f "/host/usr/local/bin/blobfuse-proxy" ];then
  old=$(sha256sum /host/usr/local/bin/blobfuse-proxy | awk '{print $1}')
  new=$(sha256sum /blobfuse-proxy/blobfuse-proxy | awk '{print $1}')
  if [ "$old" = "$new" ];then
    updateBlobfuseProxy="false"
    echo "no need to update blobfuse-proxy"
  fi
fi
if [ "$updateBlobfuseProxy" = "true" ];then
  echo "copy blobfuse-proxy...."
  rm -rf /host/"$KUBELET_PATH"/plugins/blob.csi.azure.com/blobfuse-proxy.sock
  rm -rf /host/usr/local/bin/blobfuse-proxy
  cp /blobfuse-proxy/blobfuse-proxy /host/usr/local/bin/blobfuse-proxy
  chmod 755 /host/usr/local/bin/blobfuse-proxy
fi

updateService="true"
echo "change from /usr/bin/blobfuse-proxy to /usr/local/bin/blobfuse-proxy in blobfuse-proxy.service"
sed -i 's/\/usr\/bin\/blobfuse-proxy/\/usr\/local\/bin\/blobfuse-proxy/g' /blobfuse-proxy/blobfuse-proxy.service
if [ -f "/host/etc/systemd/system/blobfuse-proxy.service" ];then
  old=$(sha256sum /host/etc/systemd/system/blobfuse-proxy.service | awk '{print $1}')
  new=$(sha256sum /blobfuse-proxy/blobfuse-proxy.service | awk '{print $1}')
  if [ "$old" = "$new" ];then
      updateService="false"
      echo "no need to update blobfuse-proxy.service"
  fi
fi
if [ "$updateService" = "true" ];then
  echo "copy blobfuse-proxy.service...."
  mkdir -p /host/etc/systemd/system/
  cp /blobfuse-proxy/blobfuse-proxy.service /host/etc/systemd/system/blobfuse-proxy.service
fi

if [ "${INSTALL_BLOBFUSE_PROXY}" = "true" ];then
  if [ "$updateBlobfuseProxy" = "true" ] || [ "$updateService" = "true" ];then
    echo "start blobfuse-proxy...."
    $HOST_CMD systemctl daemon-reload
    $HOST_CMD systemctl enable blobfuse-proxy.service
    $HOST_CMD systemctl restart blobfuse-proxy.service
  fi
fi
