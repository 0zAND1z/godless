// Code generated by protoc-gen-go. DO NOT EDIT.
// source: namespace.proto

/*
Package godless is a generated protocol buffer package.

It is generated from these files:
	namespace.proto

It has these top-level messages:
	NamespaceMessage
	IndexProto
	IndexEntryMessage
	NamespaceEntryMessage
*/
package godless

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type NamespaceMessage struct {
	Entries []*NamespaceEntryMessage `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty"`
}

func (m *NamespaceMessage) Reset()                    { *m = NamespaceMessage{} }
func (m *NamespaceMessage) String() string            { return proto.CompactTextString(m) }
func (*NamespaceMessage) ProtoMessage()               {}
func (*NamespaceMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *NamespaceMessage) GetEntries() []*NamespaceEntryMessage {
	if m != nil {
		return m.Entries
	}
	return nil
}

type IndexProto struct {
	Entries []*IndexEntryMessage `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty"`
}

func (m *IndexProto) Reset()                    { *m = IndexProto{} }
func (m *IndexProto) String() string            { return proto.CompactTextString(m) }
func (*IndexProto) ProtoMessage()               {}
func (*IndexProto) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *IndexProto) GetEntries() []*IndexEntryMessage {
	if m != nil {
		return m.Entries
	}
	return nil
}

type IndexEntryMessage struct {
	Table string   `protobuf:"bytes,1,opt,name=table" json:"table,omitempty"`
	Links []string `protobuf:"bytes,2,rep,name=links" json:"links,omitempty"`
}

func (m *IndexEntryMessage) Reset()                    { *m = IndexEntryMessage{} }
func (m *IndexEntryMessage) String() string            { return proto.CompactTextString(m) }
func (*IndexEntryMessage) ProtoMessage()               {}
func (*IndexEntryMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *IndexEntryMessage) GetTable() string {
	if m != nil {
		return m.Table
	}
	return ""
}

func (m *IndexEntryMessage) GetLinks() []string {
	if m != nil {
		return m.Links
	}
	return nil
}

type NamespaceEntryMessage struct {
	Table  string   `protobuf:"bytes,1,opt,name=table" json:"table,omitempty"`
	Row    string   `protobuf:"bytes,2,opt,name=row" json:"row,omitempty"`
	Entry  string   `protobuf:"bytes,3,opt,name=entry" json:"entry,omitempty"`
	Points []string `protobuf:"bytes,4,rep,name=points" json:"points,omitempty"`
}

func (m *NamespaceEntryMessage) Reset()                    { *m = NamespaceEntryMessage{} }
func (m *NamespaceEntryMessage) String() string            { return proto.CompactTextString(m) }
func (*NamespaceEntryMessage) ProtoMessage()               {}
func (*NamespaceEntryMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *NamespaceEntryMessage) GetTable() string {
	if m != nil {
		return m.Table
	}
	return ""
}

func (m *NamespaceEntryMessage) GetRow() string {
	if m != nil {
		return m.Row
	}
	return ""
}

func (m *NamespaceEntryMessage) GetEntry() string {
	if m != nil {
		return m.Entry
	}
	return ""
}

func (m *NamespaceEntryMessage) GetPoints() []string {
	if m != nil {
		return m.Points
	}
	return nil
}

func init() {
	proto.RegisterType((*NamespaceMessage)(nil), "godless.NamespaceMessage")
	proto.RegisterType((*IndexProto)(nil), "godless.IndexProto")
	proto.RegisterType((*IndexEntryMessage)(nil), "godless.IndexEntryMessage")
	proto.RegisterType((*NamespaceEntryMessage)(nil), "godless.NamespaceEntryMessage")
}

func init() { proto.RegisterFile("namespace.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 212 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xcf, 0x4b, 0xcc, 0x4d,
	0x2d, 0x2e, 0x48, 0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4f, 0xcf, 0x4f,
	0xc9, 0x49, 0x2d, 0x2e, 0x56, 0xf2, 0xe1, 0x12, 0xf0, 0x83, 0xc9, 0xf9, 0xa6, 0x16, 0x17, 0x27,
	0xa6, 0xa7, 0x0a, 0x59, 0x70, 0xb1, 0xa7, 0xe6, 0x95, 0x14, 0x65, 0xa6, 0x16, 0x4b, 0x30, 0x2a,
	0x30, 0x6b, 0x70, 0x1b, 0xc9, 0xe9, 0x41, 0x95, 0xeb, 0xc1, 0xd5, 0xba, 0xe6, 0x95, 0x14, 0x55,
	0x42, 0x35, 0x04, 0xc1, 0x94, 0x2b, 0x39, 0x71, 0x71, 0x79, 0xe6, 0xa5, 0xa4, 0x56, 0x04, 0x80,
	0x2d, 0x31, 0x41, 0x37, 0x47, 0x0a, 0x6e, 0x0e, 0x58, 0x15, 0x76, 0x33, 0xec, 0xb9, 0x04, 0x31,
	0x64, 0x85, 0x44, 0xb8, 0x58, 0x4b, 0x12, 0x93, 0x72, 0x52, 0x25, 0x18, 0x15, 0x18, 0x35, 0x38,
	0x83, 0x20, 0x1c, 0x90, 0x68, 0x4e, 0x66, 0x5e, 0x76, 0xb1, 0x04, 0x93, 0x02, 0x33, 0x48, 0x14,
	0xcc, 0x51, 0xca, 0xe5, 0x12, 0xc5, 0xea, 0x4c, 0x1c, 0x86, 0x08, 0x70, 0x31, 0x17, 0xe5, 0x97,
	0x4b, 0x30, 0x81, 0xc5, 0x40, 0x4c, 0x90, 0x3a, 0x90, 0x63, 0x2a, 0x25, 0x98, 0x21, 0xea, 0xc0,
	0x1c, 0x21, 0x31, 0x2e, 0xb6, 0x82, 0xfc, 0xcc, 0xbc, 0x92, 0x62, 0x09, 0x16, 0xb0, 0x6d, 0x50,
	0x5e, 0x12, 0x1b, 0x38, 0x44, 0x8d, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x7c, 0x3a, 0x56, 0x50,
	0x64, 0x01, 0x00, 0x00,
}
