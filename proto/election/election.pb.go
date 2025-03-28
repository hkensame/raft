// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v3.21.12
// source: election.proto

package election

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ReceiveVoteReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 候选人的term
	CandidateTerm int32 `protobuf:"varint,1,opt,name=candidateTerm,proto3" json:"candidateTerm,omitempty"`
	// 候选人的id号
	CandidateId     string `protobuf:"bytes,2,opt,name=candidateId,proto3" json:"candidateId,omitempty"`
	LastLogginTerm  int32  `protobuf:"varint,3,opt,name=lastLogginTerm,proto3" json:"lastLogginTerm,omitempty"`
	LastLogginIndex int32  `protobuf:"varint,4,opt,name=lastLogginIndex,proto3" json:"lastLogginIndex,omitempty"` // int32 commitIndex = 5;
}

func (x *ReceiveVoteReq) Reset() {
	*x = ReceiveVoteReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_election_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReceiveVoteReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReceiveVoteReq) ProtoMessage() {}

func (x *ReceiveVoteReq) ProtoReflect() protoreflect.Message {
	mi := &file_election_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReceiveVoteReq.ProtoReflect.Descriptor instead.
func (*ReceiveVoteReq) Descriptor() ([]byte, []int) {
	return file_election_proto_rawDescGZIP(), []int{0}
}

func (x *ReceiveVoteReq) GetCandidateTerm() int32 {
	if x != nil {
		return x.CandidateTerm
	}
	return 0
}

func (x *ReceiveVoteReq) GetCandidateId() string {
	if x != nil {
		return x.CandidateId
	}
	return ""
}

func (x *ReceiveVoteReq) GetLastLogginTerm() int32 {
	if x != nil {
		return x.LastLogginTerm
	}
	return 0
}

func (x *ReceiveVoteReq) GetLastLogginIndex() int32 {
	if x != nil {
		return x.LastLogginIndex
	}
	return 0
}

type ReceiveVoteRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 可能用于丢弃老term时期发送的投票请求
	CandidatorTerm int32 `protobuf:"varint,3,opt,name=candidatorTerm,proto3" json:"candidatorTerm,omitempty"`
	VoterTerm      int32 `protobuf:"varint,1,opt,name=voterTerm,proto3" json:"voterTerm,omitempty"`
	VoteFor        bool  `protobuf:"varint,2,opt,name=voteFor,proto3" json:"voteFor,omitempty"`
}

func (x *ReceiveVoteRes) Reset() {
	*x = ReceiveVoteRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_election_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReceiveVoteRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReceiveVoteRes) ProtoMessage() {}

func (x *ReceiveVoteRes) ProtoReflect() protoreflect.Message {
	mi := &file_election_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReceiveVoteRes.ProtoReflect.Descriptor instead.
func (*ReceiveVoteRes) Descriptor() ([]byte, []int) {
	return file_election_proto_rawDescGZIP(), []int{1}
}

func (x *ReceiveVoteRes) GetCandidatorTerm() int32 {
	if x != nil {
		return x.CandidatorTerm
	}
	return 0
}

func (x *ReceiveVoteRes) GetVoterTerm() int32 {
	if x != nil {
		return x.VoterTerm
	}
	return 0
}

func (x *ReceiveVoteRes) GetVoteFor() bool {
	if x != nil {
		return x.VoteFor
	}
	return false
}

var File_election_proto protoreflect.FileDescriptor

var file_election_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xaa, 0x01, 0x0a, 0x0e, 0x52, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x56, 0x6f, 0x74, 0x65,
	0x52, 0x65, 0x71, 0x12, 0x24, 0x0a, 0x0d, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x54, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0d, 0x63, 0x61, 0x6e, 0x64,
	0x69, 0x64, 0x61, 0x74, 0x65, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x20, 0x0a, 0x0b, 0x63, 0x61, 0x6e,
	0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b,
	0x63, 0x61, 0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x49, 0x64, 0x12, 0x26, 0x0a, 0x0e, 0x6c,
	0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x54, 0x65, 0x72, 0x6d, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x0e, 0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x54,
	0x65, 0x72, 0x6d, 0x12, 0x28, 0x0a, 0x0f, 0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x67, 0x69,
	0x6e, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0f, 0x6c, 0x61,
	0x73, 0x74, 0x4c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x22, 0x70, 0x0a,
	0x0e, 0x52, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x73, 0x12,
	0x26, 0x0a, 0x0e, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x54, 0x65, 0x72,
	0x6d, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0e, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64, 0x61,
	0x74, 0x6f, 0x72, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x1c, 0x0a, 0x09, 0x76, 0x6f, 0x74, 0x65, 0x72,
	0x54, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x76, 0x6f, 0x74, 0x65,
	0x72, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x6f, 0x74, 0x65, 0x46, 0x6f, 0x72,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x76, 0x6f, 0x74, 0x65, 0x46, 0x6f, 0x72, 0x32,
	0x3b, 0x0a, 0x08, 0x45, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2f, 0x0a, 0x0b, 0x52,
	0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x12, 0x0f, 0x2e, 0x52, 0x65, 0x63,
	0x65, 0x69, 0x76, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x71, 0x1a, 0x0f, 0x2e, 0x52, 0x65,
	0x63, 0x65, 0x69, 0x76, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x73, 0x42, 0x0d, 0x5a, 0x0b,
	0x2e, 0x2f, 0x3b, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_election_proto_rawDescOnce sync.Once
	file_election_proto_rawDescData = file_election_proto_rawDesc
)

func file_election_proto_rawDescGZIP() []byte {
	file_election_proto_rawDescOnce.Do(func() {
		file_election_proto_rawDescData = protoimpl.X.CompressGZIP(file_election_proto_rawDescData)
	})
	return file_election_proto_rawDescData
}

var file_election_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_election_proto_goTypes = []interface{}{
	(*ReceiveVoteReq)(nil), // 0: ReceiveVoteReq
	(*ReceiveVoteRes)(nil), // 1: ReceiveVoteRes
}
var file_election_proto_depIdxs = []int32{
	0, // 0: Election.ReceiveVote:input_type -> ReceiveVoteReq
	1, // 1: Election.ReceiveVote:output_type -> ReceiveVoteRes
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_election_proto_init() }
func file_election_proto_init() {
	if File_election_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_election_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReceiveVoteReq); i {
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
		file_election_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReceiveVoteRes); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_election_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_election_proto_goTypes,
		DependencyIndexes: file_election_proto_depIdxs,
		MessageInfos:      file_election_proto_msgTypes,
	}.Build()
	File_election_proto = out.File
	file_election_proto_rawDesc = nil
	file_election_proto_goTypes = nil
	file_election_proto_depIdxs = nil
}
