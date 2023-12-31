// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: sdk/mempool/v1/query.proto

package types

import (
	context "context"
	fmt "fmt"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

// GetTxDistributionRequest is the request type for the Service.GetTxDistribution
// RPC method.
type GetTxDistributionRequest struct {
}

func (m *GetTxDistributionRequest) Reset()         { *m = GetTxDistributionRequest{} }
func (m *GetTxDistributionRequest) String() string { return proto.CompactTextString(m) }
func (*GetTxDistributionRequest) ProtoMessage()    {}
func (*GetTxDistributionRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2b59d6882c9c3543, []int{0}
}
func (m *GetTxDistributionRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GetTxDistributionRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GetTxDistributionRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GetTxDistributionRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetTxDistributionRequest.Merge(m, src)
}
func (m *GetTxDistributionRequest) XXX_Size() int {
	return m.Size()
}
func (m *GetTxDistributionRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetTxDistributionRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetTxDistributionRequest proto.InternalMessageInfo

// GetTxDistributionResponse is the response type for the Service.GetTxDistribution
// RPC method.
type GetTxDistributionResponse struct {
	// Distribution is a map of lane to the number of transactions in the mempool for that lane.
	Distribution map[string]uint64 `protobuf:"bytes,1,rep,name=distribution,proto3" json:"distribution,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (m *GetTxDistributionResponse) Reset()         { *m = GetTxDistributionResponse{} }
func (m *GetTxDistributionResponse) String() string { return proto.CompactTextString(m) }
func (*GetTxDistributionResponse) ProtoMessage()    {}
func (*GetTxDistributionResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_2b59d6882c9c3543, []int{1}
}
func (m *GetTxDistributionResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GetTxDistributionResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GetTxDistributionResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GetTxDistributionResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetTxDistributionResponse.Merge(m, src)
}
func (m *GetTxDistributionResponse) XXX_Size() int {
	return m.Size()
}
func (m *GetTxDistributionResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetTxDistributionResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetTxDistributionResponse proto.InternalMessageInfo

func (m *GetTxDistributionResponse) GetDistribution() map[string]uint64 {
	if m != nil {
		return m.Distribution
	}
	return nil
}

func init() {
	proto.RegisterType((*GetTxDistributionRequest)(nil), "sdk.mempool.v1.GetTxDistributionRequest")
	proto.RegisterType((*GetTxDistributionResponse)(nil), "sdk.mempool.v1.GetTxDistributionResponse")
	proto.RegisterMapType((map[string]uint64)(nil), "sdk.mempool.v1.GetTxDistributionResponse.DistributionEntry")
}

func init() { proto.RegisterFile("sdk/mempool/v1/query.proto", fileDescriptor_2b59d6882c9c3543) }

var fileDescriptor_2b59d6882c9c3543 = []byte{
	// 326 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2a, 0x4e, 0xc9, 0xd6,
	0xcf, 0x4d, 0xcd, 0x2d, 0xc8, 0xcf, 0xcf, 0xd1, 0x2f, 0x33, 0xd4, 0x2f, 0x2c, 0x4d, 0x2d, 0xaa,
	0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x2b, 0x4e, 0xc9, 0xd6, 0x83, 0xca, 0xe9, 0x95,
	0x19, 0x4a, 0xc9, 0xa4, 0xe7, 0xe7, 0xa7, 0xe7, 0xa4, 0xea, 0x27, 0x16, 0x64, 0xea, 0x27, 0xe6,
	0xe5, 0xe5, 0x97, 0x24, 0x96, 0x64, 0xe6, 0xe7, 0x15, 0x43, 0x54, 0x2b, 0x49, 0x71, 0x49, 0xb8,
	0xa7, 0x96, 0x84, 0x54, 0xb8, 0x64, 0x16, 0x97, 0x14, 0x65, 0x26, 0x95, 0x82, 0xe4, 0x82, 0x52,
	0x0b, 0x4b, 0x53, 0x8b, 0x4b, 0x94, 0xf6, 0x32, 0x72, 0x49, 0x62, 0x91, 0x2c, 0x2e, 0xc8, 0xcf,
	0x2b, 0x4e, 0x15, 0x8a, 0xe7, 0xe2, 0x49, 0x41, 0x12, 0x97, 0x60, 0x54, 0x60, 0xd6, 0xe0, 0x36,
	0xb2, 0xd6, 0x43, 0xb5, 0x5e, 0x0f, 0xa7, 0x01, 0x7a, 0xc8, 0x82, 0xae, 0x79, 0x25, 0x45, 0x95,
	0x41, 0x28, 0x06, 0x4a, 0xd9, 0x73, 0x09, 0x62, 0x28, 0x11, 0x12, 0xe0, 0x62, 0xce, 0x4e, 0xad,
	0x94, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0x02, 0x31, 0x85, 0x44, 0xb8, 0x58, 0xcb, 0x12, 0x73,
	0x4a, 0x53, 0x25, 0x98, 0x14, 0x18, 0x35, 0x58, 0x82, 0x20, 0x1c, 0x2b, 0x26, 0x0b, 0x46, 0xa3,
	0x05, 0x8c, 0x5c, 0xec, 0xc1, 0xa9, 0x45, 0x65, 0x99, 0xc9, 0xa9, 0x42, 0x53, 0x18, 0xb9, 0x04,
	0x31, 0x9c, 0x22, 0xa4, 0x41, 0x84, 0x6b, 0xc1, 0x61, 0x21, 0xa5, 0x49, 0xb4, 0xbf, 0x94, 0xb4,
	0x9a, 0x2e, 0x3f, 0x99, 0xcc, 0xa4, 0x22, 0xa4, 0xa4, 0x9f, 0x94, 0x93, 0x9f, 0x9c, 0xad, 0x8b,
	0x16, 0x57, 0xc8, 0x7e, 0x74, 0xf2, 0x3e, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07,
	0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96, 0x63, 0xb8, 0xf0, 0x58, 0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x86,
	0x28, 0xc3, 0xf4, 0xcc, 0x92, 0x8c, 0xd2, 0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0xfd, 0xe2, 0xec, 0xcc,
	0x02, 0xdd, 0xdc, 0xd4, 0x32, 0x24, 0x03, 0xc1, 0x2c, 0xfd, 0x62, 0x88, 0xef, 0xf4, 0x4b, 0x2a,
	0x0b, 0x52, 0x8b, 0x93, 0xd8, 0xc0, 0x51, 0x6a, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff, 0xfb, 0xaf,
	0x0d, 0xf5, 0x1e, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ServiceClient is the client API for Service service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ServiceClient interface {
	// GetTxDistribution returns the distribution of transactions in the mempool.
	GetTxDistribution(ctx context.Context, in *GetTxDistributionRequest, opts ...grpc.CallOption) (*GetTxDistributionResponse, error)
}

type serviceClient struct {
	cc grpc1.ClientConn
}

func NewServiceClient(cc grpc1.ClientConn) ServiceClient {
	return &serviceClient{cc}
}

func (c *serviceClient) GetTxDistribution(ctx context.Context, in *GetTxDistributionRequest, opts ...grpc.CallOption) (*GetTxDistributionResponse, error) {
	out := new(GetTxDistributionResponse)
	err := c.cc.Invoke(ctx, "/sdk.mempool.v1.Service/GetTxDistribution", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServiceServer is the server API for Service service.
type ServiceServer interface {
	// GetTxDistribution returns the distribution of transactions in the mempool.
	GetTxDistribution(context.Context, *GetTxDistributionRequest) (*GetTxDistributionResponse, error)
}

// UnimplementedServiceServer can be embedded to have forward compatible implementations.
type UnimplementedServiceServer struct {
}

func (*UnimplementedServiceServer) GetTxDistribution(ctx context.Context, req *GetTxDistributionRequest) (*GetTxDistributionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTxDistribution not implemented")
}

func RegisterServiceServer(s grpc1.Server, srv ServiceServer) {
	s.RegisterService(&_Service_serviceDesc, srv)
}

func _Service_GetTxDistribution_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTxDistributionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).GetTxDistribution(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sdk.mempool.v1.Service/GetTxDistribution",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).GetTxDistribution(ctx, req.(*GetTxDistributionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Service_serviceDesc = grpc.ServiceDesc{
	ServiceName: "sdk.mempool.v1.Service",
	HandlerType: (*ServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetTxDistribution",
			Handler:    _Service_GetTxDistribution_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "sdk/mempool/v1/query.proto",
}

func (m *GetTxDistributionRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetTxDistributionRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GetTxDistributionRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *GetTxDistributionResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetTxDistributionResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GetTxDistributionResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Distribution) > 0 {
		for k := range m.Distribution {
			v := m.Distribution[k]
			baseI := i
			i = encodeVarintQuery(dAtA, i, uint64(v))
			i--
			dAtA[i] = 0x10
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintQuery(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintQuery(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintQuery(dAtA []byte, offset int, v uint64) int {
	offset -= sovQuery(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GetTxDistributionRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *GetTxDistributionResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Distribution) > 0 {
		for k, v := range m.Distribution {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovQuery(uint64(len(k))) + 1 + sovQuery(uint64(v))
			n += mapEntrySize + 1 + sovQuery(uint64(mapEntrySize))
		}
	}
	return n
}

func sovQuery(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozQuery(x uint64) (n int) {
	return sovQuery(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GetTxDistributionRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: GetTxDistributionRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GetTxDistributionRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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
func (m *GetTxDistributionResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: GetTxDistributionResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GetTxDistributionResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Distribution", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Distribution == nil {
				m.Distribution = make(map[string]uint64)
			}
			var mapkey string
			var mapvalue uint64
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowQuery
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
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowQuery
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthQuery
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthQuery
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowQuery
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						mapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipQuery(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthQuery
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Distribution[mapkey] = mapvalue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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
func skipQuery(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQuery
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
					return 0, ErrIntOverflowQuery
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
					return 0, ErrIntOverflowQuery
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
				return 0, ErrInvalidLengthQuery
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQuery
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQuery
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQuery        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQuery          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQuery = fmt.Errorf("proto: unexpected end of group")
)
