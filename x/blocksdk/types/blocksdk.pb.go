// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: sdk/blocksdk/v1/blocksdk.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Lane defines a block-sdk lane and its associated parameters.  Only the
// parameters that are critical to consensus are stored on-chain in this object.
// The other associated configuration for a lane can be set and stored locally,
// per-validator.
type Lane struct {
	// id is the unique identifier of a Lane.  Maps to a block-sdk laneName.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// max_block_space defines the relative percentage of block space that can be
	// used by this lane. NOTE: If this is set to zero, then there is no limit
	// on the number of transactions that can be included in the block for this
	// lane (up to maxTxBytes as provided by the request). This is useful for the
	// default lane.
	MaxBlockSpace cosmossdk_io_math.LegacyDec `protobuf:"bytes,2,opt,name=max_block_space,json=maxBlockSpace,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"max_block_space"`
	// order is the priority ordering of the Lane when processed in
	// PrepareProposal and ProcessProposal. Lane orders should be set in order of
	// priority starting from 0, monotonically increasing and non-overlapping. A
	// lane with a lower order value will have a higher priority over a lane with
	// a higher order value.  For example, if LaneA has priority of 0 and LaneB
	// has a priority of 1, LaneA has priority over LaneB.
	Order uint64 `protobuf:"varint,3,opt,name=order,proto3" json:"order,omitempty"`
}

func (m *Lane) Reset()         { *m = Lane{} }
func (m *Lane) String() string { return proto.CompactTextString(m) }
func (*Lane) ProtoMessage()    {}
func (*Lane) Descriptor() ([]byte, []int) {
	return fileDescriptor_f70e4127b99be786, []int{0}
}
func (m *Lane) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Lane) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Lane.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Lane) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Lane.Merge(m, src)
}
func (m *Lane) XXX_Size() int {
	return m.Size()
}
func (m *Lane) XXX_DiscardUnknown() {
	xxx_messageInfo_Lane.DiscardUnknown(m)
}

var xxx_messageInfo_Lane proto.InternalMessageInfo

func (m *Lane) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Lane) GetOrder() uint64 {
	if m != nil {
		return m.Order
	}
	return 0
}

func init() {
	proto.RegisterType((*Lane)(nil), "sdk.blocksdk.v1.Lane")
}

func init() { proto.RegisterFile("sdk/blocksdk/v1/blocksdk.proto", fileDescriptor_f70e4127b99be786) }

var fileDescriptor_f70e4127b99be786 = []byte{
	// 280 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2b, 0x4e, 0xc9, 0xd6,
	0x4f, 0xca, 0xc9, 0x4f, 0xce, 0x06, 0x31, 0xca, 0x0c, 0xe1, 0x6c, 0xbd, 0x82, 0xa2, 0xfc, 0x92,
	0x7c, 0x21, 0x7e, 0x10, 0x13, 0x2e, 0x56, 0x66, 0x28, 0x25, 0x92, 0x9e, 0x9f, 0x9e, 0x0f, 0x96,
	0xd3, 0x07, 0xb1, 0x20, 0xca, 0xa4, 0x04, 0x13, 0x73, 0x33, 0xf3, 0xf2, 0xf5, 0xc1, 0x24, 0x54,
	0x48, 0x32, 0x39, 0xbf, 0x38, 0x37, 0xbf, 0x38, 0x1e, 0xa2, 0x16, 0xc2, 0x81, 0x48, 0x29, 0xf5,
	0x30, 0x72, 0xb1, 0xf8, 0x24, 0xe6, 0xa5, 0x0a, 0xf1, 0x71, 0x31, 0x65, 0xa6, 0x48, 0x30, 0x2a,
	0x30, 0x6a, 0x70, 0x06, 0x31, 0x65, 0xa6, 0x08, 0xc5, 0x71, 0xf1, 0xe7, 0x26, 0x56, 0xc4, 0x83,
	0xed, 0x8b, 0x2f, 0x2e, 0x48, 0x4c, 0x4e, 0x95, 0x60, 0x02, 0x49, 0x3a, 0x99, 0x9d, 0xb8, 0x27,
	0xcf, 0x70, 0xeb, 0x9e, 0xbc, 0x34, 0xc4, 0x1c, 0x90, 0x5b, 0x32, 0xf3, 0xf5, 0x73, 0x13, 0x4b,
	0x32, 0xf4, 0x7c, 0x52, 0xd3, 0x13, 0x93, 0x2b, 0x5d, 0x52, 0x93, 0x2f, 0x6d, 0xd1, 0xe5, 0x82,
	0x5a, 0xe3, 0x92, 0x9a, 0xbc, 0xe2, 0xf9, 0x06, 0x2d, 0xc6, 0x20, 0xde, 0xdc, 0xc4, 0x0a, 0x27,
	0x90, 0x69, 0xc1, 0x20, 0xc3, 0x84, 0x44, 0xb8, 0x58, 0xf3, 0x8b, 0x52, 0x52, 0x8b, 0x24, 0x98,
	0x15, 0x18, 0x35, 0x58, 0x82, 0x20, 0x1c, 0x27, 0x8f, 0x13, 0x8f, 0xe4, 0x18, 0x2f, 0x3c, 0x92,
	0x63, 0x7c, 0xf0, 0x48, 0x8e, 0x71, 0xc2, 0x63, 0x39, 0x86, 0x0b, 0x8f, 0xe5, 0x18, 0x6e, 0x3c,
	0x96, 0x63, 0x88, 0xd2, 0x4b, 0xcf, 0x2c, 0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x2f,
	0xce, 0xce, 0x2c, 0xd0, 0xcd, 0x4d, 0x2d, 0x83, 0x84, 0x90, 0x2e, 0x28, 0xb8, 0x2a, 0x10, 0x21,
	0x57, 0x52, 0x59, 0x90, 0x5a, 0x9c, 0xc4, 0x06, 0xf6, 0x9f, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff,
	0xba, 0xb0, 0x4c, 0x7e, 0x56, 0x01, 0x00, 0x00,
}

func (m *Lane) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Lane) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Lane) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Order != 0 {
		i = encodeVarintBlocksdk(dAtA, i, uint64(m.Order))
		i--
		dAtA[i] = 0x18
	}
	{
		size := m.MaxBlockSpace.Size()
		i -= size
		if _, err := m.MaxBlockSpace.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintBlocksdk(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintBlocksdk(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintBlocksdk(dAtA []byte, offset int, v uint64) int {
	offset -= sovBlocksdk(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Lane) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovBlocksdk(uint64(l))
	}
	l = m.MaxBlockSpace.Size()
	n += 1 + l + sovBlocksdk(uint64(l))
	if m.Order != 0 {
		n += 1 + sovBlocksdk(uint64(m.Order))
	}
	return n
}

func sovBlocksdk(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBlocksdk(x uint64) (n int) {
	return sovBlocksdk(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Lane) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlocksdk
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Lane: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Lane: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlocksdk
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthBlocksdk
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBlocksdk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxBlockSpace", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlocksdk
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthBlocksdk
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBlocksdk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MaxBlockSpace.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Order", wireType)
			}
			m.Order = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlocksdk
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Order |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipBlocksdk(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlocksdk
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipBlocksdk(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBlocksdk
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowBlocksdk
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowBlocksdk
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthBlocksdk
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBlocksdk
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBlocksdk
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBlocksdk        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBlocksdk          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBlocksdk = fmt.Errorf("proto: unexpected end of group")
)
