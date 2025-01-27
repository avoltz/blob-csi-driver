// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.18.1
// source: csi_mounts.proto

package csi_mounts

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	blob_cache_volume "sigs.k8s.io/blob-csi-driver/pkg/edgecache/blob_cache_volume"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MountReqResult int32

const (
	MountReqResult_SUCCESS   MountReqResult = 0
	MountReqResult_TRY_AGAIN MountReqResult = 1
)

// Enum value maps for MountReqResult.
var (
	MountReqResult_name = map[int32]string{
		0: "SUCCESS",
		1: "TRY_AGAIN",
	}
	MountReqResult_value = map[string]int32{
		"SUCCESS":   0,
		"TRY_AGAIN": 1,
	}
)

func (x MountReqResult) Enum() *MountReqResult {
	p := new(MountReqResult)
	*p = x
	return p
}

func (x MountReqResult) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (MountReqResult) Descriptor() protoreflect.EnumDescriptor {
	return file_csi_mounts_proto_enumTypes[0].Descriptor()
}

func (MountReqResult) Type() protoreflect.EnumType {
	return &file_csi_mounts_proto_enumTypes[0]
}

func (x MountReqResult) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use MountReqResult.Descriptor instead.
func (MountReqResult) EnumDescriptor() ([]byte, []int) {
	return file_csi_mounts_proto_rawDescGZIP(), []int{0}
}

type VolumeInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to VolumeInfo:
	//	*VolumeInfo_BlobVolume
	VolumeInfo isVolumeInfo_VolumeInfo `protobuf_oneof:"volume_info"`
	VolumeId   *string                 `protobuf:"bytes,2,opt,name=volume_id,json=volumeId,proto3,oneof" json:"volume_id,omitempty"`
}

func (x *VolumeInfo) Reset() {
	*x = VolumeInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_csi_mounts_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VolumeInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VolumeInfo) ProtoMessage() {}

func (x *VolumeInfo) ProtoReflect() protoreflect.Message {
	mi := &file_csi_mounts_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VolumeInfo.ProtoReflect.Descriptor instead.
func (*VolumeInfo) Descriptor() ([]byte, []int) {
	return file_csi_mounts_proto_rawDescGZIP(), []int{0}
}

func (m *VolumeInfo) GetVolumeInfo() isVolumeInfo_VolumeInfo {
	if m != nil {
		return m.VolumeInfo
	}
	return nil
}

func (x *VolumeInfo) GetBlobVolume() *blob_cache_volume.Name {
	if x, ok := x.GetVolumeInfo().(*VolumeInfo_BlobVolume); ok {
		return x.BlobVolume
	}
	return nil
}

func (x *VolumeInfo) GetVolumeId() string {
	if x != nil && x.VolumeId != nil {
		return *x.VolumeId
	}
	return ""
}

type isVolumeInfo_VolumeInfo interface {
	isVolumeInfo_VolumeInfo()
}

type VolumeInfo_BlobVolume struct {
	BlobVolume *blob_cache_volume.Name `protobuf:"bytes,1,opt,name=blob_volume,json=blobVolume,proto3,oneof"`
}

func (*VolumeInfo_BlobVolume) isVolumeInfo_VolumeInfo() {}

type AddMountReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TargetPath *string     `protobuf:"bytes,1,opt,name=target_path,json=targetPath,proto3,oneof" json:"target_path,omitempty"`
	VolumeInfo *VolumeInfo `protobuf:"bytes,2,opt,name=volume_info,json=volumeInfo,proto3,oneof" json:"volume_info,omitempty"`
}

func (x *AddMountReq) Reset() {
	*x = AddMountReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_csi_mounts_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddMountReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddMountReq) ProtoMessage() {}

func (x *AddMountReq) ProtoReflect() protoreflect.Message {
	mi := &file_csi_mounts_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddMountReq.ProtoReflect.Descriptor instead.
func (*AddMountReq) Descriptor() ([]byte, []int) {
	return file_csi_mounts_proto_rawDescGZIP(), []int{1}
}

func (x *AddMountReq) GetTargetPath() string {
	if x != nil && x.TargetPath != nil {
		return *x.TargetPath
	}
	return ""
}

func (x *AddMountReq) GetVolumeInfo() *VolumeInfo {
	if x != nil {
		return x.VolumeInfo
	}
	return nil
}

type AddMountRsp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *AddMountRsp) Reset() {
	*x = AddMountRsp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_csi_mounts_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddMountRsp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddMountRsp) ProtoMessage() {}

func (x *AddMountRsp) ProtoReflect() protoreflect.Message {
	mi := &file_csi_mounts_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddMountRsp.ProtoReflect.Descriptor instead.
func (*AddMountRsp) Descriptor() ([]byte, []int) {
	return file_csi_mounts_proto_rawDescGZIP(), []int{2}
}

type RemoveMountReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TargetPath *string `protobuf:"bytes,1,opt,name=target_path,json=targetPath,proto3,oneof" json:"target_path,omitempty"`
}

func (x *RemoveMountReq) Reset() {
	*x = RemoveMountReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_csi_mounts_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoveMountReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveMountReq) ProtoMessage() {}

func (x *RemoveMountReq) ProtoReflect() protoreflect.Message {
	mi := &file_csi_mounts_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveMountReq.ProtoReflect.Descriptor instead.
func (*RemoveMountReq) Descriptor() ([]byte, []int) {
	return file_csi_mounts_proto_rawDescGZIP(), []int{3}
}

func (x *RemoveMountReq) GetTargetPath() string {
	if x != nil && x.TargetPath != nil {
		return *x.TargetPath
	}
	return ""
}

type RemoveMountRsp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *RemoveMountRsp) Reset() {
	*x = RemoveMountRsp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_csi_mounts_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoveMountRsp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveMountRsp) ProtoMessage() {}

func (x *RemoveMountRsp) ProtoReflect() protoreflect.Message {
	mi := &file_csi_mounts_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveMountRsp.ProtoReflect.Descriptor instead.
func (*RemoveMountRsp) Descriptor() ([]byte, []int) {
	return file_csi_mounts_proto_rawDescGZIP(), []int{4}
}

var File_csi_mounts_proto protoreflect.FileDescriptor

var file_csi_mounts_proto_rawDesc = []byte{
	0x0a, 0x10, 0x63, 0x73, 0x69, 0x5f, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0a, 0x63, 0x73, 0x69, 0x5f, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x1a, 0x17,
	0x62, 0x6c, 0x6f, 0x62, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x76, 0x6f, 0x6c, 0x75, 0x6d,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x87, 0x01, 0x0a, 0x0a, 0x56, 0x6f, 0x6c, 0x75,
	0x6d, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x3a, 0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x62, 0x5f, 0x76,
	0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x62, 0x6c,
	0x6f, 0x62, 0x5f, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x2e,
	0x4e, 0x61, 0x6d, 0x65, 0x48, 0x00, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x62, 0x56, 0x6f, 0x6c, 0x75,
	0x6d, 0x65, 0x12, 0x20, 0x0a, 0x09, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x5f, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x08, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x49,
	0x64, 0x88, 0x01, 0x01, 0x42, 0x0d, 0x0a, 0x0b, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x5f, 0x69,
	0x6e, 0x66, 0x6f, 0x42, 0x0c, 0x0a, 0x0a, 0x5f, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x5f, 0x69,
	0x64, 0x22, 0x91, 0x01, 0x0a, 0x0b, 0x41, 0x64, 0x64, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65,
	0x71, 0x12, 0x24, 0x0a, 0x0b, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x5f, 0x70, 0x61, 0x74, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0a, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x50, 0x61, 0x74, 0x68, 0x88, 0x01, 0x01, 0x12, 0x3c, 0x0a, 0x0b, 0x76, 0x6f, 0x6c, 0x75, 0x6d,
	0x65, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x63,
	0x73, 0x69, 0x5f, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x2e, 0x56, 0x6f, 0x6c, 0x75, 0x6d, 0x65,
	0x49, 0x6e, 0x66, 0x6f, 0x48, 0x01, 0x52, 0x0a, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x49, 0x6e,
	0x66, 0x6f, 0x88, 0x01, 0x01, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x5f, 0x70, 0x61, 0x74, 0x68, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65,
	0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x22, 0x0d, 0x0a, 0x0b, 0x41, 0x64, 0x64, 0x4d, 0x6f, 0x75, 0x6e,
	0x74, 0x52, 0x73, 0x70, 0x22, 0x46, 0x0a, 0x0e, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4d, 0x6f,
	0x75, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x12, 0x24, 0x0a, 0x0b, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0a, 0x74,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x50, 0x61, 0x74, 0x68, 0x88, 0x01, 0x01, 0x42, 0x0e, 0x0a, 0x0c,
	0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x22, 0x10, 0x0a, 0x0e,
	0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x73, 0x70, 0x2a, 0x2c,
	0x0a, 0x0e, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43, 0x45, 0x53, 0x53, 0x10, 0x00, 0x12, 0x0d, 0x0a,
	0x09, 0x54, 0x52, 0x59, 0x5f, 0x41, 0x47, 0x41, 0x49, 0x4e, 0x10, 0x01, 0x32, 0x90, 0x01, 0x0a,
	0x09, 0x43, 0x53, 0x49, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x12, 0x3c, 0x0a, 0x08, 0x41, 0x64,
	0x64, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x17, 0x2e, 0x63, 0x73, 0x69, 0x5f, 0x6d, 0x6f, 0x75,
	0x6e, 0x74, 0x73, 0x2e, 0x41, 0x64, 0x64, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x1a,
	0x17, 0x2e, 0x63, 0x73, 0x69, 0x5f, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x2e, 0x41, 0x64, 0x64,
	0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x73, 0x70, 0x12, 0x45, 0x0a, 0x0b, 0x52, 0x65, 0x6d, 0x6f,
	0x76, 0x65, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1a, 0x2e, 0x63, 0x73, 0x69, 0x5f, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x73, 0x2e, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4d, 0x6f, 0x75, 0x6e, 0x74,
	0x52, 0x65, 0x71, 0x1a, 0x1a, 0x2e, 0x63, 0x73, 0x69, 0x5f, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x73,
	0x2e, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4d, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x73, 0x70, 0x42,
	0x36, 0x5a, 0x34, 0x73, 0x69, 0x67, 0x73, 0x2e, 0x6b, 0x38, 0x73, 0x2e, 0x69, 0x6f, 0x2f, 0x62,
	0x6c, 0x6f, 0x62, 0x2d, 0x63, 0x73, 0x69, 0x2d, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x2f, 0x70,
	0x6b, 0x67, 0x2f, 0x65, 0x64, 0x67, 0x65, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2f, 0x63, 0x73, 0x69,
	0x5f, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_csi_mounts_proto_rawDescOnce sync.Once
	file_csi_mounts_proto_rawDescData = file_csi_mounts_proto_rawDesc
)

func file_csi_mounts_proto_rawDescGZIP() []byte {
	file_csi_mounts_proto_rawDescOnce.Do(func() {
		file_csi_mounts_proto_rawDescData = protoimpl.X.CompressGZIP(file_csi_mounts_proto_rawDescData)
	})
	return file_csi_mounts_proto_rawDescData
}

var file_csi_mounts_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_csi_mounts_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_csi_mounts_proto_goTypes = []interface{}{
	(MountReqResult)(0),            // 0: csi_mounts.MountReqResult
	(*VolumeInfo)(nil),             // 1: csi_mounts.VolumeInfo
	(*AddMountReq)(nil),            // 2: csi_mounts.AddMountReq
	(*AddMountRsp)(nil),            // 3: csi_mounts.AddMountRsp
	(*RemoveMountReq)(nil),         // 4: csi_mounts.RemoveMountReq
	(*RemoveMountRsp)(nil),         // 5: csi_mounts.RemoveMountRsp
	(*blob_cache_volume.Name)(nil), // 6: blob_cache_volume.Name
}
var file_csi_mounts_proto_depIdxs = []int32{
	6, // 0: csi_mounts.VolumeInfo.blob_volume:type_name -> blob_cache_volume.Name
	1, // 1: csi_mounts.AddMountReq.volume_info:type_name -> csi_mounts.VolumeInfo
	2, // 2: csi_mounts.CSIMounts.AddMount:input_type -> csi_mounts.AddMountReq
	4, // 3: csi_mounts.CSIMounts.RemoveMount:input_type -> csi_mounts.RemoveMountReq
	3, // 4: csi_mounts.CSIMounts.AddMount:output_type -> csi_mounts.AddMountRsp
	5, // 5: csi_mounts.CSIMounts.RemoveMount:output_type -> csi_mounts.RemoveMountRsp
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_csi_mounts_proto_init() }
func file_csi_mounts_proto_init() {
	if File_csi_mounts_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_csi_mounts_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VolumeInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_csi_mounts_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddMountReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_csi_mounts_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddMountRsp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_csi_mounts_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoveMountReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_csi_mounts_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoveMountRsp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_csi_mounts_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*VolumeInfo_BlobVolume)(nil),
	}
	file_csi_mounts_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_csi_mounts_proto_msgTypes[3].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_csi_mounts_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_csi_mounts_proto_goTypes,
		DependencyIndexes: file_csi_mounts_proto_depIdxs,
		EnumInfos:         file_csi_mounts_proto_enumTypes,
		MessageInfos:      file_csi_mounts_proto_msgTypes,
	}.Build()
	File_csi_mounts_proto = out.File
	file_csi_mounts_proto_rawDesc = nil
	file_csi_mounts_proto_goTypes = nil
	file_csi_mounts_proto_depIdxs = nil
}
