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

FROM mcr.microsoft.com/cbl-mariner/base/core:2.0

ARG ARCH=amd64
ARG binary=./_output/${ARCH}/blobplugin
COPY ${binary} /blobplugin

RUN mkdir /blobfuse-proxy/

COPY ./pkg/blobfuse-proxy/init.sh /blobfuse-proxy/
COPY ./pkg/blobfuse-proxy/blobfuse-proxy.service /blobfuse-proxy/
COPY ./_output/${ARCH}/blobfuse-proxy /blobfuse-proxy/

RUN chmod +x /blobfuse-proxy/init.sh && \
 chmod +x /blobfuse-proxy/blobfuse-proxy.service && \
 chmod +x /blobfuse-proxy/blobfuse-proxy

# Currently no CBL-Mariner image with fix for "curl"/"zlib" CVE-2023-38545/CVE-2023-38546/CVE-2023-45853.
# So, temporarily do update here. Remove "curl"/"zlib" when image is updated.
RUN tdnf updateinfo && \
    tdnf install -y util-linux e2fsprogs nfs-utils quota-rpc rpcbind blobfuse2 fuse3 libcap-ng libcap ca-certificates curl zlib && \
    tdnf clean all

LABEL maintainers="andyzhangx"
LABEL description="Azure Blob Storage CSI driver"

ENTRYPOINT ["/blobplugin"]
