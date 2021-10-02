package genopenapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	anypb "github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/grpc-ecosystem/grpc-gateway/v2/internal/descriptor"
	"github.com/grpc-ecosystem/grpc-gateway/v2/internal/descriptor/openapiconfig"
	"github.com/grpc-ecosystem/grpc-gateway/v2/internal/httprule"
	openapi_options "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv3/options"
)

func TestMessageToQueryParametersWithEnumAsInt(t *testing.T) {
	it := require.New(t)
	type test struct {
		Name     string
		MsgDescs []*descriptorpb.DescriptorProto
		Message  string
		Params   []Parameter
	}

	tests := []test{
		{
			Name: "basic query params",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   proto.String("a"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(1),
						},
						{
							Name:   proto.String("b"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(),
							Number: proto.Int32(2),
						},
						{
							Name:   proto.String("c"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Label:  descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
							Number: proto.Int32(3),
						},
					},
				},
			},
			Message: "ExampleMessage",
			Params: []Parameter{
				{
					Name:     "a",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type: "string",
						},
					},
				},
				{
					Name:     "b",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type:   "number",
							Format: "double",
						},
					},
				},
				{
					Name:     "c",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type: "array",
							Items: &SchemaRef{
								Value: &Schema{
									Type: "string",
								},
							},
							Format: "multi",
						},
					},
				},
			},
		},
		{
			Name: "nested query params",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("nested"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.Nested"),
							Number:   proto.Int32(1),
						},
					},
				},
				{
					Name: proto.String("Nested"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   proto.String("a"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(1),
						},
						{
							Name:     proto.String("deep"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.Nested.DeepNested"),
							Number:   proto.Int32(2),
						},
					},
					NestedType: []*descriptorpb.DescriptorProto{{
						Name: proto.String("DeepNested"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("b"),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Number: proto.Int32(1),
							},
							{
								Name:     proto.String("c"),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
								TypeName: proto.String(".example.Nested.DeepNested.DeepEnum"),
								Number:   proto.Int32(2),
							},
						},
						EnumType: []*descriptorpb.EnumDescriptorProto{
							{
								Name: proto.String("DeepEnum"),
								Value: []*descriptorpb.EnumValueDescriptorProto{
									{Name: proto.String("FALSE"), Number: proto.Int32(0)},
									{Name: proto.String("TRUE"), Number: proto.Int32(1)},
								},
							},
						},
					}},
				},
			},
			Message: "ExampleMessage",
			Params: []Parameter{
				{
					Name:     "nested.a",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type: "string",
						},
					},
				},
				{
					Name:     "nested.deep.b",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type: "string",
						},
					},
				},
				{
					Name:     "nested.deep.c",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type:    "integer",
							Enum:    []interface{}{"0", "1"},
							Default: "0",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			reg := descriptor.NewRegistry()
			reg.SetEnumsAsInts(true)
			msgs := []*descriptor.Message{}
			for _, msgdesc := range test.MsgDescs {
				msgs = append(msgs, &descriptor.Message{DescriptorProto: msgdesc})
			}
			file := descriptor.File{
				FileDescriptorProto: &descriptorpb.FileDescriptorProto{
					SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
					Name:           proto.String("example.proto"),
					Package:        proto.String("example"),
					Dependency:     []string{},
					MessageType:    test.MsgDescs,
					Service:        []*descriptorpb.ServiceDescriptorProto{},
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
					},
				},
				GoPkg: descriptor.GoPackage{
					Path: "example.com/path/to/example/example.pb",
					Name: "example_pb",
				},
				Messages: msgs,
			}
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
			})
			it.NoError(err)

			message, err := reg.LookupMsg("", ".example."+test.Message)
			it.NoError(err)
			params, err := messageToQueryParameters(message, reg, []descriptor.Parameter{}, nil)
			it.NoError(err)
			// avoid checking Items for array types
			//for i := range params {
			//	params[i].Schema.Value.Items = nil
			//}
			it.Equal(test.Params, params)
		})
	}
}

func TestMessageToQueryParameters(t *testing.T) {
	it := require.New(t)
	type test struct {
		Name string
		MsgDescs       []*descriptorpb.DescriptorProto
		Message        string
		ExpectedParams []Parameter
	}

	tests := []test{
		{
			Name: "basic",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   proto.String("a"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(1),
						},
						{
							Name:   proto.String("b"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(),
							Number: proto.Int32(2),
						},
						{
							Name:   proto.String("c"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Label:  descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
							Number: proto.Int32(3),
						},
					},
				},
			},
			Message: "ExampleMessage",
			ExpectedParams: []Parameter{
				{
					Name:     "a",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string"}},
				},
				{
					Name:     "b",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "number", Format: "double"}},
				},
				{
					Name:     "c",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "array", Format: "multi"}},
				},
			},
		},
		{
			Name: "nested",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("nested"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.Nested"),
							Number:   proto.Int32(1),
						},
					},
				},
				{
					Name: proto.String("Nested"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   proto.String("a"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(1),
						},
						{
							Name:     proto.String("deep"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.Nested.DeepNested"),
							Number:   proto.Int32(2),
						},
					},
					NestedType: []*descriptorpb.DescriptorProto{{
						Name: proto.String("DeepNested"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("b"),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Number: proto.Int32(1),
							},
							{
								Name:     proto.String("c"),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
								TypeName: proto.String(".example.Nested.DeepNested.DeepEnum"),
								Number:   proto.Int32(2),
							},
						},
						EnumType: []*descriptorpb.EnumDescriptorProto{
							{
								Name: proto.String("DeepEnum"),
								Value: []*descriptorpb.EnumValueDescriptorProto{
									{Name: proto.String("FALSE"), Number: proto.Int32(0)},
									{Name: proto.String("TRUE"), Number: proto.Int32(1)},
								},
							},
						},
					}},
				},
			},
			Message: "ExampleMessage",
			ExpectedParams: []Parameter{
				{
					Name:     "nested.a",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string"}},
				},
				{
					Name:     "nested.deep.b",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string"}},
				},
				{
					Name:     "nested.deep.c",
					In:       "query",
					Required: false,
					Schema: &SchemaRef{
						Value: &Schema{
							Type:    "string",
							Enum:    []interface{}{"FALSE", "TRUE"},
							Default: "FALSE",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.Name, func(t *testing.T) {
			reg := descriptor.NewRegistry()
			msgs := []*descriptor.Message{}
			for _, msgdesc := range test.MsgDescs {
				msgs = append(msgs, &descriptor.Message{DescriptorProto: msgdesc})
			}
			file := descriptor.File{
				FileDescriptorProto: &descriptorpb.FileDescriptorProto{
					SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
					Name:           proto.String("example.proto"),
					Package:        proto.String("example"),
					Dependency:     []string{},
					MessageType:    test.MsgDescs,
					Service:        []*descriptorpb.ServiceDescriptorProto{},
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
					},
				},
				GoPkg: descriptor.GoPackage{
					Path: "example.com/path/to/example/example.pb",
					Name: "example_pb",
				},
				Messages: msgs,
			}
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
			})
			it.NoError(err)

			message, err := reg.LookupMsg("", ".example."+test.Message)
			it.NoError(err)
			params, err := messageToQueryParameters(message, reg, []descriptor.Parameter{}, nil)
			it.NoError(err)
			// avoid checking Items for array types
			for i := range params {
				params[i].Schema.Value.Items = nil
			}
			it.Equal(test.ExpectedParams, params)
		})
	}
}

// TestMessagetoQueryParametersNoRecursive, is a check that cyclical references between messages
//  are not falsely detected given previous known edge-cases.
func TestMessageToQueryParametersNoRecursive(t *testing.T) {
	it := require.New(t)
	type test struct {
		Name     string
		MsgDescs []*descriptorpb.DescriptorProto
		Message  string
	}

	tests := []test{
		// First test:
		// Here is a message that has two of another message adjacent to one another in a nested message.
		// There is no loop but this was previouly falsely flagged as a cycle.
		// Example proto:
		// message NonRecursiveMessage {
		//      string field = 1;
		// }
		// message BaseMessage {
		//      NonRecursiveMessage first = 1;
		//      NonRecursiveMessage second = 2;
		// }
		// message QueryMessage {
		//      BaseMessage first = 1;
		//      string second = 2;
		// }
		{
			Name: "basic",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("QueryMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("first"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.BaseMessage"),
							Number:   proto.Int32(1),
						},
						{
							Name:   proto.String("second"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(2),
						},
					},
				},
				{
					Name: proto.String("BaseMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("first"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.NonRecursiveMessage"),
							Number:   proto.Int32(1),
						},
						{
							Name:     proto.String("second"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.NonRecursiveMessage"),
							Number:   proto.Int32(2),
						},
					},
				},
				// Note there is no recursive nature to this message
				{
					Name: proto.String("NonRecursiveMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name: proto.String("field"),
							//Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(1),
						},
					},
				},
			},
			Message: "QueryMessage",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			reg := descriptor.NewRegistry()
			msgs := []*descriptor.Message{}
			for _, msgdesc := range test.MsgDescs {
				msgs = append(msgs, &descriptor.Message{DescriptorProto: msgdesc})
			}
			file := descriptor.File{
				FileDescriptorProto: &descriptorpb.FileDescriptorProto{
					SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
					Name:           proto.String("example.proto"),
					Package:        proto.String("example"),
					Dependency:     []string{},
					MessageType:    test.MsgDescs,
					Service:        []*descriptorpb.ServiceDescriptorProto{},
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
					},
				},
				GoPkg: descriptor.GoPackage{
					Path: "example.com/path/to/example/example.pb",
					Name: "example_pb",
				},
				Messages: msgs,
			}
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
			})
			it.NoError(err)

			message, err := reg.LookupMsg("", ".example."+test.Message)
			it.NoError(err)

			_, err = messageToQueryParameters(message, reg, []descriptor.Parameter{}, nil)
			it.NoError(err)
		})
	}
}

// TestMessagetoQueryParametersRecursive, is a check that cyclical references between messages
//  are handled gracefully. The goal is to insure that attempts to add messages with cyclical
//  references to query-parameters returns an error message.
func TestMessageToQueryParametersRecursive(t *testing.T) {
	it := require.New(t)

	type test struct {
		Name     string
		MsgDescs []*descriptorpb.DescriptorProto
		Message  string
	}

	tests := []test{
		// First test:
		// Here we test that a message that references it self through a field will return an error.
		// Example proto:
		// message DirectRecursiveMessage {
		//      DirectRecursiveMessage nested = 1;
		// }
		{
			Name: "reference it self return error",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("DirectRecursiveMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("nested"),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.DirectRecursiveMessage"),
							Number:   proto.Int32(1),
						},
					},
				},
			},
			Message: "DirectRecursiveMessage",
		},
		// Second test:
		// Here we test that a cycle through multiple messages is detected and that an error is returned.
		// Sample:
		// message Root { NodeMessage nested = 1; }
		// message NodeMessage { CycleMessage nested = 1; }
		// message CycleMessage { Root nested = 1; }
		{
			Name: "cycle through multiple messages return error",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("RootMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("nested"),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.NodeMessage"),
							Number:   proto.Int32(1),
						},
					},
				},
				{
					Name: proto.String("NodeMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("nested"),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.CycleMessage"),
							Number:   proto.Int32(1),
						},
					},
				},
				{
					Name: proto.String("CycleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("nested"),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.RootMessage"),
							Number:   proto.Int32(1),
						},
					},
				},
			},
			Message: "RootMessage",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			reg := descriptor.NewRegistry()
			msgs := []*descriptor.Message{}
			for _, msgdesc := range test.MsgDescs {
				msgs = append(msgs, &descriptor.Message{DescriptorProto: msgdesc})
			}
			file := descriptor.File{
				FileDescriptorProto: &descriptorpb.FileDescriptorProto{
					SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
					Name:           proto.String("example.proto"),
					Package:        proto.String("example"),
					Dependency:     []string{},
					MessageType:    test.MsgDescs,
					Service:        []*descriptorpb.ServiceDescriptorProto{},
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
					},
				},
				GoPkg: descriptor.GoPackage{
					Path: "example.com/path/to/example/example.pb",
					Name: "example_pb",
				},
				Messages: msgs,
			}
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
			})
			it.NoError(err)

			message, err := reg.LookupMsg("", ".example."+test.Message)
			it.NoError(err)
			_, err = messageToQueryParameters(message, reg, []descriptor.Parameter{}, nil)
			it.Error(err)
		})
	}
}

func TestMessageToQueryParametersWithJsonName(t *testing.T) {
	it := require.New(t)

	type test struct {
		Name           string
		MsgDescs       []*descriptorpb.DescriptorProto
		Message        string
		ExpectedParams []Parameter
	}

	tests := []test{
		{
			Name: "basic",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("test_field_a"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number:   proto.Int32(1),
							JsonName: proto.String("testFieldA"),
						},
					},
				},
			},
			Message: "ExampleMessage",
			ExpectedParams: []Parameter{
				{
					Name:     "testFieldA",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string"}},
				},
			},
		},
		{
			Name: "sub message",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("SubMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("test_field_a"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number:   proto.Int32(1),
							JsonName: proto.String("testFieldA"),
						},
					},
				},
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("sub_message"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".example.SubMessage"),
							Number:   proto.Int32(1),
							JsonName: proto.String("subMessage"),
						},
					},
				},
			},
			Message: "ExampleMessage",
			ExpectedParams: []Parameter{
				{
					Name:     "subMessage.testFieldA",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string"}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			reg := descriptor.NewRegistry()
			reg.SetUseJSONNamesForFields(true)
			msgs := []*descriptor.Message{}
			for _, msgdesc := range test.MsgDescs {
				msgs = append(msgs, &descriptor.Message{DescriptorProto: msgdesc})
			}
			file := descriptor.File{
				FileDescriptorProto: &descriptorpb.FileDescriptorProto{
					SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
					Name:           proto.String("example.proto"),
					Package:        proto.String("example"),
					Dependency:     []string{},
					MessageType:    test.MsgDescs,
					Service:        []*descriptorpb.ServiceDescriptorProto{},
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
					},
				},
				GoPkg: descriptor.GoPackage{
					Path: "example.com/path/to/example/example.pb",
					Name: "example_pb",
				},
				Messages: msgs,
			}
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
			})
			it.NoError(err)

			message, err := reg.LookupMsg("", ".example."+test.Message)
			it.NoError(err)
			params, err := messageToQueryParameters(message, reg, []descriptor.Parameter{}, nil)
			it.NoError(err)
			it.Equal(test.ExpectedParams, params)
		})
	}
}

func TestMessageToQueryParametersWellKnownTypes(t *testing.T) {
	it := require.New(t)

	type test struct {
		Name              string
		MsgDescs          []*descriptorpb.DescriptorProto
		WellKnownMsgDescs []*descriptorpb.DescriptorProto
		Message           string
		ExpectedParams    []Parameter
	}

	tests := []test{
		{
			Name: "basic",
			MsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ExampleMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("a_field_mask"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".google.protobuf.FieldMask"),
							Number:   proto.Int32(1),
						},
						{
							Name:     proto.String("a_timestamp"),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".google.protobuf.Timestamp"),
							Number:   proto.Int32(2),
						},
					},
				},
			},
			WellKnownMsgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("FieldMask"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   proto.String("paths"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Label:  descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
							Number: proto.Int32(1),
						},
					},
				},
				{
					Name: proto.String("Timestamp"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   proto.String("seconds"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
							Number: proto.Int32(1),
						},
						{
							Name:   proto.String("nanos"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
							Number: proto.Int32(2),
						},
					},
				},
			},
			Message: "ExampleMessage",
			ExpectedParams: []Parameter{
				{
					Name:     "a_field_mask",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string"}},
				},
				{
					Name:     "a_timestamp",
					In:       "query",
					Required: false,
					Schema:   &SchemaRef{Value: &Schema{Type: "string", Format: "date-time"}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			reg := descriptor.NewRegistry()
			reg.SetEnumsAsInts(true)
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{
					{
						SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
						Name:           proto.String("google/well_known.proto"),
						Package:        proto.String("google.protobuf"),
						Dependency:     []string{},
						MessageType:    test.WellKnownMsgDescs,
						Service:        []*descriptorpb.ServiceDescriptorProto{},
						Options: &descriptorpb.FileOptions{
							GoPackage: proto.String("google/well_known"),
						},
					},
					{
						SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
						Name:           proto.String("acme/example.proto"),
						Package:        proto.String("example"),
						Dependency:     []string{"google/well_known.proto"},
						MessageType:    test.MsgDescs,
						Service:        []*descriptorpb.ServiceDescriptorProto{},
						Options: &descriptorpb.FileOptions{
							GoPackage: proto.String("acme/example"),
						},
					},
				},
			})
			it.NoError(err)

			message, err := reg.LookupMsg("", ".example."+test.Message)
			it.NoError(err)
			params, err := messageToQueryParameters(message, reg, []descriptor.Parameter{}, nil)
			it.NoError(err)
			it.Equal(test.ExpectedParams, params)
		})
	}
}

func TestApplyTemplate(t *testing.T) {
	it := require.New(t)

	msgdesc := &descriptorpb.DescriptorProto{
		Name: proto.String("ExampleMessage"),
	}
	meth := &descriptorpb.MethodDescriptorProto{
		Name:       proto.String("Example"),
		InputType:  proto.String("ExampleMessage"),
		OutputType: proto.String("ExampleMessage"),
	}

	// Create two services that have the same method name. We will test that the
	// operation IDs are different
	svc := &descriptorpb.ServiceDescriptorProto{
		Name:   proto.String("ExampleService"),
		Method: []*descriptorpb.MethodDescriptorProto{meth},
	}
	svc2 := &descriptorpb.ServiceDescriptorProto{
		Name:   proto.String("OtherService"),
		Method: []*descriptorpb.MethodDescriptorProto{meth},
	}

	msg := &descriptor.Message{
		DescriptorProto: msgdesc,
	}
	file := descriptor.File{
		FileDescriptorProto: &descriptorpb.FileDescriptorProto{
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
			Name:           proto.String("example.proto"),
			Package:        proto.String("example"),
			MessageType:    []*descriptorpb.DescriptorProto{msgdesc},
			Service:        []*descriptorpb.ServiceDescriptorProto{svc},
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
			},
		},
		GoPkg: descriptor.GoPackage{
			Path: "example.com/path/to/example/example.pb",
			Name: "example_pb",
		},
		Messages: []*descriptor.Message{msg},
		Services: []*descriptor.Service{
			{
				ServiceDescriptorProto: svc,
				Methods: []*descriptor.Method{
					{
						MethodDescriptorProto: meth,
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*descriptor.Binding{
							{
								HTTPMethod: "GET",
								Body:       &descriptor.Body{FieldPath: nil},
								PathTmpl: httprule.Template{
									Version:  1,
									OpCodes:  []int{0, 0},
									Template: "/v1/echo",
								},
							},
						},
					},
				},
			},
			{
				ServiceDescriptorProto: svc2,
				Methods: []*descriptor.Method{
					{
						MethodDescriptorProto: meth,
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*descriptor.Binding{
							{
								HTTPMethod: "POST",
								Body:       &descriptor.Body{FieldPath: nil},
								PathTmpl: httprule.Template{
									Version:  1,
									OpCodes:  []int{0, 0},
									Template: "/v1/ping",
								},
							},
						},
					},
				},
			},
		},
	}

	reg := descriptor.NewRegistry()
	reg.SetDisableDefaultErrors(true)
	err := AddErrorDefs(reg)
	it.NoError(err)

	fileCL := crossLinkFixture(&file)
	err = reg.Load(reqFromFile(fileCL))
	it.NoError(err)

	result, err := applyTemplate(param{File: fileCL, reg: reg})
	it.NoError(err)

	// Check that the two services have unique operation IDs even though they
	// have the same method name.
	it.Equal("ExampleService_Example", result.Paths["/v1/echo"].Get.OperationID)
	it.Equal("OtherService_Example", result.Paths["/v1/ping"].Post.OperationID)
	it.Equal(&PathItem{
		Get: &Operation{
			Tags:        []string{"ExampleService"},
			Summary:     "",
			Description: "",
			OperationID: "ExampleService_Example",
			Parameters: []*ParameterRef{
				{
					Value: &Parameter{
						Name:     "body",
						In:       "body",
						Required: true,
						Schema: &SchemaRef{
							Ref: "#/components/schemas/exampleExampleMessage",
						},
					},
				},
			},
			Responses: map[string]*ResponseRef{
				"200": {
					Ref: "",
					Value: &Response{
						Description: strPtr("A successful response."),
						Headers:     Headers{},
						Content: Content{
							"application/json": &MediaType{
								Schema: &SchemaRef{
									Ref: "#/components/schemas/exampleExampleMessage",
								},
							},
						},
					},
				},
			},
		},
		Parameters: nil,
	}, result.Paths["/v1/echo"])

	it.Equal(&PathItem{
		Post: &Operation{
			Tags:        []string{"OtherService"},
			Summary:     "",
			Description: "",
			OperationID: "OtherService_Example",
			Parameters: []*ParameterRef{
				{
					Value: &Parameter{
						Name:     "body",
						In:       "body",
						Required: true,
						Schema: &SchemaRef{
							Ref: "#/components/schemas/exampleExampleMessage",
						},
					},
				},
			},
			Responses: map[string]*ResponseRef{
				"200": {
					Ref: "",
					Value: &Response{
						Description: strPtr("A successful response."),
						Headers:     Headers{},
						Content: Content{
							"application/json": &MediaType{
								Schema: &SchemaRef{
									Ref: "#/components/schemas/exampleExampleMessage",
								},
							},
						},
					},
				},
			},
		},
		Parameters: nil,
	}, result.Paths["/v1/ping"])
}

func TestApplyTemplateOverrideOperationID(t *testing.T) {
	it := require.New(t)

	newFile := func() *descriptor.File {
		msgdesc := &descriptorpb.DescriptorProto{
			Name: proto.String("ExampleMessage"),
		}
		meth := &descriptorpb.MethodDescriptorProto{
			Name:       proto.String("Example"),
			InputType:  proto.String("ExampleMessage"),
			OutputType: proto.String("ExampleMessage"),
			Options:    &descriptorpb.MethodOptions{},
		}
		svc := &descriptorpb.ServiceDescriptorProto{
			Name:   proto.String("ExampleService"),
			Method: []*descriptorpb.MethodDescriptorProto{meth},
		}
		msg := &descriptor.Message{
			DescriptorProto: msgdesc,
		}
		return &descriptor.File{
			FileDescriptorProto: &descriptorpb.FileDescriptorProto{
				SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
				Name:           proto.String("example.proto"),
				Package:        proto.String("example"),
				MessageType:    []*descriptorpb.DescriptorProto{msgdesc},
				Service:        []*descriptorpb.ServiceDescriptorProto{svc},
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
				},
			},
			GoPkg: descriptor.GoPackage{
				Path: "example.com/path/to/example/example.pb",
				Name: "example_pb",
			},
			Messages: []*descriptor.Message{msg},
			Services: []*descriptor.Service{
				{
					ServiceDescriptorProto: svc,
					Methods: []*descriptor.Method{
						{
							MethodDescriptorProto: meth,
							RequestType:           msg,
							ResponseType:          msg,
							Bindings: []*descriptor.Binding{
								{
									HTTPMethod: "GET",
									Body:       &descriptor.Body{FieldPath: nil},
									PathTmpl: httprule.Template{
										Version:  1,
										OpCodes:  []int{0, 0},
										Template: "/v1/echo", // TODO(achew22): Figure out what this should really be
									},
								},
							},
						},
					},
				},
			},
		}
	}

	verifyTemplateFromReq := func(t *testing.T, reg *descriptor.Registry, file *descriptor.File, opts *openapiconfig.OpenAPIOptions) {
		if err := AddErrorDefs(reg); err != nil {
			t.Errorf("AddErrorDefs(%#v) failed with %v; want success", reg, err)
			return
		}
		fileCL := crossLinkFixture(file)
		err := reg.Load(reqFromFile(fileCL))
		it.NoError(err)
		if opts != nil {
			err = reg.RegisterOpenAPIOptions(opts)
			it.NoError(err)
		}
		result, err := applyTemplate(param{File: fileCL, reg: reg})
		it.NoError(err)
		it.Equal("MyExample", result.Paths["/v1/echo"].Get.OperationID)
	}

	openapiOperation := openapi_options.Operation{
		OperationId: "MyExample",
	}

	t.Run("verify override via method option", func(t *testing.T) {
		file := newFile()
		proto.SetExtension(proto.Message(file.Services[0].Methods[0].MethodDescriptorProto.Options),
			openapi_options.E_Openapiv3Operation, &openapiOperation)

		reg := descriptor.NewRegistry()
		verifyTemplateFromReq(t, reg, file, nil)
	})

	// TODO(anjmao): This one is hard. It requires touching internal/descriptor pkg which imports openapiv2.
	//t.Run("verify override options annotations", func(t *testing.T) {
	//	file := newFile()
	//	reg := descriptor.NewRegistry()
	//	opts := &openapiconfig.OpenAPIOptions{
	//		Method: []*openapiconfig.OpenAPIMethodOption{
	//			{
	//				Method: "example.ExampleService.Example",
	//				Option: &openapiOperation,
	//			},
	//		},
	//	}
	//	verifyTemplateFromReq(t, reg, file, opts)
	//})
}

func TestApplyTemplateHeaders(t *testing.T) {
	it := require.New(t)
	newFile := func() *descriptor.File {
		msgdesc := &descriptorpb.DescriptorProto{
			Name: proto.String("ExampleMessage"),
		}
		meth := &descriptorpb.MethodDescriptorProto{
			Name:       proto.String("Example"),
			InputType:  proto.String("ExampleMessage"),
			OutputType: proto.String("ExampleMessage"),
			Options:    &descriptorpb.MethodOptions{},
		}
		svc := &descriptorpb.ServiceDescriptorProto{
			Name:   proto.String("ExampleService"),
			Method: []*descriptorpb.MethodDescriptorProto{meth},
		}
		msg := &descriptor.Message{
			DescriptorProto: msgdesc,
		}
		return &descriptor.File{
			FileDescriptorProto: &descriptorpb.FileDescriptorProto{
				SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
				Name:           proto.String("example.proto"),
				Package:        proto.String("example"),
				MessageType:    []*descriptorpb.DescriptorProto{msgdesc},
				Service:        []*descriptorpb.ServiceDescriptorProto{svc},
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
				},
			},
			GoPkg: descriptor.GoPackage{
				Path: "example.com/path/to/example/example.pb",
				Name: "example_pb",
			},
			Messages: []*descriptor.Message{msg},
			Services: []*descriptor.Service{
				{
					ServiceDescriptorProto: svc,
					Methods: []*descriptor.Method{
						{
							MethodDescriptorProto: meth,
							RequestType:           msg,
							ResponseType:          msg,
							Bindings: []*descriptor.Binding{
								{
									HTTPMethod: "GET",
									Body:       &descriptor.Body{FieldPath: nil},
									PathTmpl: httprule.Template{
										Version:  1,
										OpCodes:  []int{0, 0},
										Template: "/v1/echo", // TODO(achew22): Figure out what this should really be
									},
								},
							},
						},
					},
				},
			},
		}
	}

	openapiOperation := openapi_options.Operation{
		Responses: &openapi_options.Responses{
			ResponseOrReference: []*openapi_options.NamedResponseOrReference{
				{
					Name: "200",
					Value: &openapi_options.ResponseOrReference{
						Oneof: &openapi_options.ResponseOrReference_Response{
							Response: &openapi_options.Response{
								Headers: &openapi_options.HeadersOrReferences{
									AdditionalProperties: []*openapi_options.NamedHeaderOrReference{
										{
											Name: "String",
											Value: &openapi_options.HeaderOrReference{
												Oneof: &openapi_options.HeaderOrReference_Header{
													Header: &openapi_options.Header{
														Description: "string header description",
														Schema: &openapi_options.SchemaOrReference{
															Oneof: &openapi_options.SchemaOrReference_Schema{
																Schema: &openapi_options.Schema{
																	Type:   "string",
																	Format: "uuid",
																},
															},
														},
													},
												},
											},
										},
										{
											Name: "Boolean",
											Value: &openapi_options.HeaderOrReference{
												Oneof: &openapi_options.HeaderOrReference_Header{
													Header: &openapi_options.Header{
														Description: "boolean header description",
														Schema: &openapi_options.SchemaOrReference{
															Oneof: &openapi_options.SchemaOrReference_Schema{
																Schema: &openapi_options.Schema{
																	Type: "boolean",
																	Default: &openapi_options.DefaultType{
																		Oneof: &openapi_options.DefaultType_Boolean{
																			Boolean: true,
																		},
																	},
																	Pattern: "^true|false$",
																},
															},
														},
													},
												},
											},
										},
										{
											Name: "Integer",
											Value: &openapi_options.HeaderOrReference{
												Oneof: &openapi_options.HeaderOrReference_Header{
													Header: &openapi_options.Header{
														Description: "integer header description",
														Schema: &openapi_options.SchemaOrReference{
															Oneof: &openapi_options.SchemaOrReference_Schema{
																Schema: &openapi_options.Schema{
																	Type: "integer",
																	Default: &openapi_options.DefaultType{
																		Oneof: &openapi_options.DefaultType_Number{
																			Number: 0,
																		},
																	},
																	Pattern: "^[0-9]$",
																},
															},
														},
													},
												},
											},
										},
										{
											Name: "Number",
											Value: &openapi_options.HeaderOrReference{
												Oneof: &openapi_options.HeaderOrReference_Header{
													Header: &openapi_options.Header{
														Description: "number header description",
														Schema: &openapi_options.SchemaOrReference{
															Oneof: &openapi_options.SchemaOrReference_Schema{
																Schema: &openapi_options.Schema{
																	Type: "number",
																	Default: &openapi_options.DefaultType{
																		Oneof: &openapi_options.DefaultType_Number{
																			Number: 1.2,
																		},
																	},
																	Pattern: "^[-+]?[0-9]*\\\\.?[0-9]+([eE][-+]?[0-9]+)?$",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	verifyTemplateHeaders := func(t *testing.T, reg *descriptor.Registry, file *descriptor.File, opts *openapiconfig.OpenAPIOptions) {
		err := AddErrorDefs(reg)
		it.NoError(err)
		fileCL := crossLinkFixture(file)
		err = reg.Load(reqFromFile(fileCL))
		it.NoError(err)
		if opts != nil {
			// TODO(anjmao): This one registers v2 options.
			err = reg.RegisterOpenAPIOptions(opts)
			it.NoError(err)
		}
		result, err := applyTemplate(param{File: fileCL, reg: reg})
		it.NoError(err)
		it.Equal("3.0.3", result.OpenAPI)

		expectedHeaders := Headers{
			"String": &HeaderRef{
				Value: &Header{
					Description: "string header description",
					Schema: &SchemaRef{
						Value: &Schema{
							Type:    "string",
							Format:  "uuid",
							Pattern: "",
						},
					},
				},
			},
			"Boolean": &HeaderRef{
				Value: &Header{
					Description: "boolean header description",
					Schema: &SchemaRef{
						Value: &Schema{
							Type:    "boolean",
							Default: true,
							Pattern: "^true|false$",
						},
					},
				},
			},
			"Integer": &HeaderRef{
				Value: &Header{
					Description: "integer header description",
					Schema: &SchemaRef{
						Value: &Schema{
							Type:    "integer",
							Default: float64(0),
							Pattern: "^[0-9]$",
						},
					},
				},
			},
			"Number": &HeaderRef{
				Value: &Header{
					Description: "number header description",
					Schema: &SchemaRef{
						Value: &Schema{
							Type:    "number",
							Default: float64(1.2),
							Pattern: `^[-+]?[0-9]*\\.?[0-9]+([eE][-+]?[0-9]+)?$`,
						},
					},
				},
			},
		}

		verifyHeader := func(a, b *Header) {
			s1, s2 := a.Schema.Value, b.Schema.Value
			it.Equal(s1.Type, s2.Type)
			it.Equal(s1.Default, s2.Default)
			it.Equal(s1.Pattern, s2.Pattern)
			it.Equal(s1.Format, s2.Format)
		}

		actualHeaders := result.Paths["/v1/echo"].Get.Responses["200"].Value.Headers
		verifyHeader(expectedHeaders["String"].Value, actualHeaders["String"].Value)
		verifyHeader(expectedHeaders["Boolean"].Value, actualHeaders["Boolean"].Value)
		verifyHeader(expectedHeaders["Integer"].Value, actualHeaders["Integer"].Value)
		verifyHeader(expectedHeaders["Number"].Value, actualHeaders["Number"].Value)
	}

	t.Run("verify template options set via proto options", func(t *testing.T) {
		file := newFile()
		proto.SetExtension(proto.Message(file.Services[0].Methods[0].Options), openapi_options.E_Openapiv3Operation, &openapiOperation)
		reg := descriptor.NewRegistry()
		verifyTemplateHeaders(t, reg, file, nil)
	})
}

func TestValidateHeaderType(t *testing.T) {
	type test struct {
		Type          string
		Format        string
		expectedError error
	}
	tests := []test{
		{
			"string",
			"date-time",
			nil,
		},
		{
			"boolean",
			"",
			nil,
		},
		{
			"integer",
			"uint",
			nil,
		},
		{
			"integer",
			"uint8",
			nil,
		},
		{
			"integer",
			"uint16",
			nil,
		},
		{
			"integer",
			"uint32",
			nil,
		},
		{
			"integer",
			"uint64",
			nil,
		},
		{
			"integer",
			"int",
			nil,
		},
		{
			"integer",
			"int8",
			nil,
		},
		{
			"integer",
			"int16",
			nil,
		},
		{
			"integer",
			"int32",
			nil,
		},
		{
			"integer",
			"int64",
			nil,
		},
		{
			"integer",
			"float64",
			errors.New("the provided format \"float64\" is not a valid extension of the type \"integer\""),
		},
		{
			"integer",
			"uuid",
			errors.New("the provided format \"uuid\" is not a valid extension of the type \"integer\""),
		},
		{
			"number",
			"uint",
			nil,
		},
		{
			"number",
			"uint8",
			nil,
		},
		{
			"number",
			"uint16",
			nil,
		},
		{
			"number",
			"uint32",
			nil,
		},
		{
			"number",
			"uint64",
			nil,
		},
		{
			"number",
			"int",
			nil,
		},
		{
			"number",
			"int8",
			nil,
		},
		{
			"number",
			"int16",
			nil,
		},
		{
			"number",
			"int32",
			nil,
		},
		{
			"number",
			"int64",
			nil,
		},
		{
			"number",
			"float",
			nil,
		},
		{
			"number",
			"float32",
			nil,
		},
		{
			"number",
			"float64",
			nil,
		},
		{
			"number",
			"complex64",
			nil,
		},
		{
			"number",
			"complex128",
			nil,
		},
		{
			"number",
			"double",
			nil,
		},
		{
			"number",
			"byte",
			nil,
		},
		{
			"number",
			"rune",
			nil,
		},
		{
			"number",
			"uintptr",
			nil,
		},
		{
			"number",
			"date",
			errors.New("the provided format \"date\" is not a valid extension of the type \"number\""),
		},
		{
			"array",
			"",
			errors.New("the provided header type \"array\" is not supported"),
		},
		{
			"foo",
			"",
			errors.New("the provided header type \"foo\" is not supported"),
		},
	}
	for _, v := range tests {
		err := validateHeaderTypeAndFormat(v.Type, v.Format)

		if v.expectedError == nil {
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
		} else {
			if err == nil {
				t.Fatal("expected header error not returned")
			}
			if err.Error() != v.expectedError.Error() {
				t.Errorf("expected error malformed, expected %q, got %q", v.expectedError.Error(), err.Error())
			}
		}
	}

}

func TestValidateDefaultValueType(t *testing.T) {
	type test struct {
		Type          string
		Value         string
		Format        string
		expectedError error
	}
	tests := []test{
		{
			"string",
			`"string"`,
			"",
			nil,
		},
		{
			"string",
			"\"2012-11-01T22:08:41+00:00\"",
			"date-time",
			nil,
		},
		{
			"string",
			"\"2012-11-01\"",
			"date",
			nil,
		},
		{
			"string",
			"0",
			"",
			errors.New("the provided default value \"0\" does not match provider type \"string\", or is not properly quoted with escaped quotations"),
		},
		{
			"string",
			"false",
			"",
			errors.New("the provided default value \"false\" does not match provider type \"string\", or is not properly quoted with escaped quotations"),
		},
		{
			"boolean",
			"true",
			"",
			nil,
		},
		{
			"boolean",
			"0",
			"",
			errors.New("the provided default value \"0\" does not match provider type \"boolean\""),
		},
		{
			"boolean",
			`"string"`,
			"",
			errors.New("the provided default value \"\\\"string\\\"\" does not match provider type \"boolean\""),
		},
		{
			"number",
			"1.2",
			"",
			nil,
		},
		{
			"number",
			"123",
			"",
			nil,
		},
		{
			"number",
			"nan",
			"",
			errors.New("the provided number \"nan\" is not a valid JSON number"),
		},
		{
			"number",
			"NaN",
			"",
			errors.New("the provided number \"NaN\" is not a valid JSON number"),
		},
		{
			"number",
			"-459.67",
			"",
			nil,
		},
		{
			"number",
			"inf",
			"",
			errors.New("the provided number \"inf\" is not a valid JSON number"),
		},
		{
			"number",
			"infinity",
			"",
			errors.New("the provided number \"infinity\" is not a valid JSON number"),
		},
		{
			"number",
			"Inf",
			"",
			errors.New("the provided number \"Inf\" is not a valid JSON number"),
		},
		{
			"number",
			"Infinity",
			"",
			errors.New("the provided number \"Infinity\" is not a valid JSON number"),
		},
		{
			"number",
			"false",
			"",
			errors.New("the provided default value \"false\" does not match provider type \"number\""),
		},
		{
			"number",
			`"string"`,
			"",
			errors.New("the provided default value \"\\\"string\\\"\" does not match provider type \"number\""),
		},
		{
			"integer",
			"2",
			"",
			nil,
		},
		{
			"integer",
			fmt.Sprint(math.MaxInt32),
			"int32",
			nil,
		},
		{
			"integer",
			fmt.Sprint(math.MaxInt32 + 1),
			"int32",
			errors.New("the provided default value \"2147483648\" does not match provided format \"int32\""),
		},
		{
			"integer",
			fmt.Sprint(math.MaxInt64),
			"int64",
			nil,
		},
		{
			"integer",
			"9223372036854775808",
			"int64",
			errors.New("the provided default value \"9223372036854775808\" does not match provided format \"int64\""),
		},
		{
			"integer",
			"18446744073709551615",
			"uint64",
			nil,
		},
		{
			"integer",
			"false",
			"",
			errors.New("the provided default value \"false\" does not match provided type \"integer\""),
		},
		{
			"integer",
			"1.2",
			"",
			errors.New("the provided default value \"1.2\" does not match provided type \"integer\""),
		},
		{
			"integer",
			`"string"`,
			"",
			errors.New("the provided default value \"\\\"string\\\"\" does not match provided type \"integer\""),
		},
	}
	for _, v := range tests {
		err := validateDefaultValueTypeAndFormat(v.Type, v.Value, v.Format)

		if v.expectedError == nil {
			if err != nil {
				t.Errorf("unexpected error '%v'", err)
			}
		} else {
			if err == nil {
				t.Error("expected update error not returned")
			}
			if err.Error() != v.expectedError.Error() {
				t.Errorf("expected error malformed, expected %q, got %q", v.expectedError.Error(), err.Error())
			}
		}
	}
}

func TestApplyTemplateRequestWithUnusedReferences(t *testing.T) {
	it := require.New(t)

	reqdesc := &descriptorpb.DescriptorProto{
		Name: proto.String("ExampleMessage"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("string"),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(1),
			},
		},
	}
	respdesc := &descriptorpb.DescriptorProto{
		Name: proto.String("EmptyMessage"),
	}
	meth := &descriptorpb.MethodDescriptorProto{
		Name:            proto.String("Example"),
		InputType:       proto.String("ExampleMessage"),
		OutputType:      proto.String("EmptyMessage"),
		ClientStreaming: proto.Bool(false),
		ServerStreaming: proto.Bool(false),
	}
	svc := &descriptorpb.ServiceDescriptorProto{
		Name:   proto.String("ExampleService"),
		Method: []*descriptorpb.MethodDescriptorProto{meth},
	}

	req := &descriptor.Message{
		DescriptorProto: reqdesc,
	}
	resp := &descriptor.Message{
		DescriptorProto: respdesc,
	}
	stringField := &descriptor.Field{
		Message:              req,
		FieldDescriptorProto: req.GetField()[0],
	}
	file := descriptor.File{
		FileDescriptorProto: &descriptorpb.FileDescriptorProto{
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
			Name:           proto.String("example.proto"),
			Package:        proto.String("example"),
			MessageType:    []*descriptorpb.DescriptorProto{reqdesc, respdesc},
			Service:        []*descriptorpb.ServiceDescriptorProto{svc},
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
			},
		},
		GoPkg: descriptor.GoPackage{
			Path: "example.com/path/to/example/example.pb",
			Name: "example_pb",
		},
		Messages: []*descriptor.Message{req, resp},
		Services: []*descriptor.Service{
			{
				ServiceDescriptorProto: svc,
				Methods: []*descriptor.Method{
					{
						MethodDescriptorProto: meth,
						RequestType:           req,
						ResponseType:          resp,
						Bindings: []*descriptor.Binding{
							{
								HTTPMethod: "GET",
								PathTmpl: httprule.Template{
									Version:  1,
									OpCodes:  []int{0, 0},
									Template: "/v1/example",
								},
							},
							{
								HTTPMethod: "POST",
								PathTmpl: httprule.Template{
									Version:  1,
									OpCodes:  []int{0, 0},
									Template: "/v1/example/{string}",
								},
								PathParams: []descriptor.Parameter{
									{
										FieldPath: descriptor.FieldPath([]descriptor.FieldPathComponent{
											{
												Name:   "string",
												Target: stringField,
											},
										}),
										Target: stringField,
									},
								},
								Body: &descriptor.Body{
									FieldPath: descriptor.FieldPath([]descriptor.FieldPathComponent{
										{
											Name:   "string",
											Target: stringField,
										},
									}),
								},
							},
						},
					},
				},
			},
		},
	}

	reg := descriptor.NewRegistry()
	err := AddErrorDefs(reg)
	it.NoError(err)
	err = reg.Load(&pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
	})
	it.NoError(err)
	result, err := applyTemplate(param{File: crossLinkFixture(&file), reg: reg})
	it.NoError(err)

	for k := range result.Components.Schemas {
		fmt.Println(k)
	}
	// Only EmptyMessage must be present, not ExampleMessage (plus error status)
	// TODO(anjmao): Check why protoAny is not returned as in v2 tests.
	it.Len(result.Components.Schemas, 2)
	_, ok := result.Components.Schemas["rpcStatus"]
	it.True(ok)
	_, ok = result.Components.Schemas["exampleEmptyMessage"]
	it.True(ok)
}

func TestApplyTemplateRequestWithBodyQueryParameters(t *testing.T) {
	bookDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("Book"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("name"),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(1),
			},
			{
				Name:   proto.String("id"),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(2),
			},
		},
	}
	createDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("CreateBookRequest"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("parent"),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(1),
			},
			{
				Name:   proto.String("book"),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(2),
			},
			{
				Name:   proto.String("book_id"),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Number: proto.Int32(3),
			},
		},
	}
	meth := &descriptorpb.MethodDescriptorProto{
		Name:       proto.String("CreateBook"),
		InputType:  proto.String("CreateBookRequest"),
		OutputType: proto.String("Book"),
	}
	svc := &descriptorpb.ServiceDescriptorProto{
		Name:   proto.String("BookService"),
		Method: []*descriptorpb.MethodDescriptorProto{meth},
	}

	bookMsg := &descriptor.Message{
		DescriptorProto: bookDesc,
	}
	createMsg := &descriptor.Message{
		DescriptorProto: createDesc,
	}

	parentField := &descriptor.Field{
		Message:              createMsg,
		FieldDescriptorProto: createMsg.GetField()[0],
	}
	bookField := &descriptor.Field{
		Message:              createMsg,
		FieldMessage:         bookMsg,
		FieldDescriptorProto: createMsg.GetField()[1],
	}
	bookIDField := &descriptor.Field{
		Message:              createMsg,
		FieldDescriptorProto: createMsg.GetField()[2],
	}

	createMsg.Fields = []*descriptor.Field{parentField, bookField, bookIDField}

	file := descriptor.File{
		FileDescriptorProto: &descriptorpb.FileDescriptorProto{
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
			Name:           proto.String("book.proto"),
			MessageType:    []*descriptorpb.DescriptorProto{bookDesc, createDesc},
			Service:        []*descriptorpb.ServiceDescriptorProto{svc},
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
			},
		},
		GoPkg: descriptor.GoPackage{
			Path: "example.com/path/to/book.pb",
			Name: "book_pb",
		},
		Messages: []*descriptor.Message{bookMsg, createMsg},
		Services: []*descriptor.Service{
			{
				ServiceDescriptorProto: svc,
				Methods: []*descriptor.Method{
					{
						MethodDescriptorProto: meth,
						RequestType:           createMsg,
						ResponseType:          bookMsg,
						Bindings: []*descriptor.Binding{
							{
								HTTPMethod: "POST",
								PathTmpl: httprule.Template{
									Version:  1,
									OpCodes:  []int{0, 0},
									Template: "/v1/{parent=publishers/*}/books",
								},
								PathParams: []descriptor.Parameter{
									{
										FieldPath: descriptor.FieldPath([]descriptor.FieldPathComponent{
											{
												Name:   "parent",
												Target: parentField,
											},
										}),
										Target: parentField,
									},
								},
								Body: &descriptor.Body{
									FieldPath: descriptor.FieldPath([]descriptor.FieldPathComponent{
										{
											Name:   "book",
											Target: bookField,
										},
									}),
								},
							},
						},
					},
				},
			},
		},
	}
	reg := descriptor.NewRegistry()
	if err := AddErrorDefs(reg); err != nil {
		t.Errorf("AddErrorDefs(%#v) failed with %v; want success", reg, err)
		return
	}
	err := reg.Load(&pluginpb.CodeGeneratorRequest{ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto}})
	if err != nil {
		t.Errorf("Registry.Load() failed with %v; want success", err)
		return
	}
	result, err := applyTemplate(param{File: crossLinkFixture(&file), reg: reg})
	if err != nil {
		t.Errorf("applyTemplate(%#v) failed with %v; want success", file, err)
		return
	}

	if _, ok := result.Paths["/v1/{parent=publishers/*}/books"].Post.Responses["200"]; !ok {
		t.Errorf("applyTemplate(%#v).%s = expected 200 response to be defined", file, `result.Paths["/v1/{parent=publishers/*}/books"].Post.Responses["200"]`)
	} else {
		if want, got, name := 3, len(result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters), `len(result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters)`; !reflect.DeepEqual(got, want) {
			t.Errorf("applyTemplate(%#v).%s = %d want to be %d", file, name, got, want)
		}

		type param struct {
			Name     string
			In       string
			Required bool
		}

		p0 := result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters[0]
		if want, got, name := (param{"parent", "path", true}), (param{p0.Value.Name, p0.Value.In, p0.Value.Required}), `result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters[0]`; !reflect.DeepEqual(got, want) {
			t.Errorf("applyTemplate(%#v).%s = %v want to be %v", file, name, got, want)
		}
		p1 := result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters[1]
		if want, got, name := (param{"body", "body", true}), (param{p1.Value.Name, p1.Value.In, p1.Value.Required}), `result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters[1]`; !reflect.DeepEqual(got, want) {
			t.Errorf("applyTemplate(%#v).%s = %v want to be %v", file, name, got, want)
		}
		p2 := result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters[2]
		if want, got, name := (param{"book_id", "query", false}), (param{p2.Value.Name, p2.Value.In, p2.Value.Required}), `result.Paths["/v1/{parent=publishers/*}/books"].Post.Parameters[1]`; !reflect.DeepEqual(got, want) {
			t.Errorf("applyTemplate(%#v).%s = %v want to be %v", file, name, got, want)
		}
	}

	// If there was a failure, print out the input and the json result for debugging.
	if t.Failed() {
		t.Errorf("had: %s", file)
		t.Errorf("got: %s", fmt.Sprint(result))
	}
}

func generateFieldsForJSONReservedName() []*descriptor.Field {
	fields := make([]*descriptor.Field, 0)
	fieldName := string("json_name")
	fieldJSONName := string("jsonNAME")
	fieldDescriptor := descriptorpb.FieldDescriptorProto{Name: &fieldName, JsonName: &fieldJSONName}
	field := &descriptor.Field{FieldDescriptorProto: &fieldDescriptor}
	return append(fields, field)
}

func generateMsgsForJSONReservedName() []*descriptor.Message {
	result := make([]*descriptor.Message, 0)
	// The first message, its field is field_abc and its type is NewType
	// NewType field_abc
	fieldName := "field_abc"
	fieldJSONName := "fieldAbc"
	messageName1 := "message1"
	messageType := "pkg.a.NewType"
	pfd := descriptorpb.FieldDescriptorProto{Name: &fieldName, JsonName: &fieldJSONName, TypeName: &messageType}
	result = append(result,
		&descriptor.Message{
			DescriptorProto: &descriptorpb.DescriptorProto{
				Name: &messageName1, Field: []*descriptorpb.FieldDescriptorProto{&pfd},
			},
		})
	// The second message, its name is NewName, its type is string
	// message NewType {
	//    string field_newName [json_name = RESERVEDJSONNAME]
	// }
	messageName := "NewType"
	field := "field_newName"
	fieldJSONName2 := "RESERVEDJSONNAME"
	pfd2 := descriptorpb.FieldDescriptorProto{Name: &field, JsonName: &fieldJSONName2}
	result = append(result, &descriptor.Message{
		DescriptorProto: &descriptorpb.DescriptorProto{
			Name: &messageName, Field: []*descriptorpb.FieldDescriptorProto{&pfd2},
		},
	})
	return result
}

func TestTemplateWithJsonCamelCase(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"/test/{test_id}", "/test/{testId}"},
		{"/test1/{test1_id}/test2/{test2_id}", "/test1/{test1Id}/test2/{test2Id}"},
		{"/test1/{test1_id}/{test2_id}", "/test1/{test1Id}/{test2Id}"},
		{"/test1/test2/{test1_id}/{test2_id}", "/test1/test2/{test1Id}/{test2Id}"},
		{"/test1/{test1_id1_id2}", "/test1/{test1Id1Id2}"},
		{"/test1/{test1_id1_id2}/test2/{test2_id3_id4}", "/test1/{test1Id1Id2}/test2/{test2Id3Id4}"},
		{"/test1/test2/{test1_id1_id2}/{test2_id3_id4}", "/test1/test2/{test1Id1Id2}/{test2Id3Id4}"},
		{"test/{a}", "test/{a}"},
		{"test/{ab}", "test/{ab}"},
		{"test/{a_a}", "test/{aA}"},
		{"test/{ab_c}", "test/{abC}"},
		{"test/{json_name}", "test/{jsonNAME}"},
		{"test/{field_abc.field_newName}", "test/{fieldAbc.RESERVEDJSONNAME}"},
	}
	reg := descriptor.NewRegistry()
	reg.SetUseJSONNamesForFields(true)
	for _, data := range tests {
		actual := templateToOpenAPIPath(data.input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		if data.expected != actual {
			t.Errorf("Expected templateToOpenAPIPath(%v) = %v, actual: %v", data.input, data.expected, actual)
		}
	}
}

func TestTemplateWithoutJsonCamelCase(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"/test/{test_id}", "/test/{test_id}"},
		{"/test1/{test1_id}/test2/{test2_id}", "/test1/{test1_id}/test2/{test2_id}"},
		{"/test1/{test1_id}/{test2_id}", "/test1/{test1_id}/{test2_id}"},
		{"/test1/test2/{test1_id}/{test2_id}", "/test1/test2/{test1_id}/{test2_id}"},
		{"/test1/{test1_id1_id2}", "/test1/{test1_id1_id2}"},
		{"/test1/{test1_id1_id2}/test2/{test2_id3_id4}", "/test1/{test1_id1_id2}/test2/{test2_id3_id4}"},
		{"/test1/test2/{test1_id1_id2}/{test2_id3_id4}", "/test1/test2/{test1_id1_id2}/{test2_id3_id4}"},
		{"test/{a}", "test/{a}"},
		{"test/{ab}", "test/{ab}"},
		{"test/{a_a}", "test/{a_a}"},
		{"test/{json_name}", "test/{json_name}"},
		{"test/{field_abc.field_newName}", "test/{field_abc.field_newName}"},
	}
	reg := descriptor.NewRegistry()
	reg.SetUseJSONNamesForFields(false)
	for _, data := range tests {
		actual := templateToOpenAPIPath(data.input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		if data.expected != actual {
			t.Errorf("Expected templateToOpenAPIPath(%v) = %v, actual: %v", data.input, data.expected, actual)
		}
	}
}

func TestTemplateToOpenAPIPath(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"/test", "/test"},
		{"/{test}", "/{test}"},
		{"/{test=prefix/*}", "/{test}"},
		{"/{test=prefix/that/has/multiple/parts/to/it/*}", "/{test}"},
		{"/{test1}/{test2}", "/{test1}/{test2}"},
		{"/{test1}/{test2}/", "/{test1}/{test2}/"},
		{"/{name=prefix/*}", "/{name=prefix/*}"},
		{"/{name=prefix1/*/prefix2/*}", "/{name=prefix1/*/prefix2/*}"},
		{"/{user.name=prefix/*}", "/{user.name=prefix/*}"},
		{"/{user.name=prefix1/*/prefix2/*}", "/{user.name=prefix1/*/prefix2/*}"},
		{"/{parent=prefix/*}/children", "/{parent=prefix/*}/children"},
		{"/{name=prefix/*}:customMethod", "/{name=prefix/*}:customMethod"},
		{"/{name=prefix1/*/prefix2/*}:customMethod", "/{name=prefix1/*/prefix2/*}:customMethod"},
		{"/{user.name=prefix/*}:customMethod", "/{user.name=prefix/*}:customMethod"},
		{"/{user.name=prefix1/*/prefix2/*}:customMethod", "/{user.name=prefix1/*/prefix2/*}:customMethod"},
		{"/{parent=prefix/*}/children:customMethod", "/{parent=prefix/*}/children:customMethod"},
	}
	reg := descriptor.NewRegistry()
	reg.SetUseJSONNamesForFields(false)
	for _, data := range tests {
		actual := templateToOpenAPIPath(data.input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		if data.expected != actual {
			t.Errorf("Expected templateToOpenAPIPath(%v) = %v, actual: %v", data.input, data.expected, actual)
		}
	}
	reg.SetUseJSONNamesForFields(true)
	for _, data := range tests {
		actual := templateToOpenAPIPath(data.input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		if data.expected != actual {
			t.Errorf("Expected templateToOpenAPIPath(%v) = %v, actual: %v", data.input, data.expected, actual)
		}
	}
}

func BenchmarkTemplateToOpenAPIPath(b *testing.B) {
	const input = "/{user.name=prefix1/*/prefix2/*}:customMethod"

	b.Run("with JSON names", func(b *testing.B) {
		reg := descriptor.NewRegistry()
		reg.SetUseJSONNamesForFields(false)

		for i := 0; i < b.N; i++ {
			_ = templateToOpenAPIPath(input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		}
	})

	b.Run("without JSON names", func(b *testing.B) {
		reg := descriptor.NewRegistry()
		reg.SetUseJSONNamesForFields(true)

		for i := 0; i < b.N; i++ {
			_ = templateToOpenAPIPath(input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		}
	})
}

func TestResolveFullyQualifiedNameToOpenAPIName(t *testing.T) {
	var tests = []struct {
		input                string
		output               string
		listOfFQMNs          []string
		useFQNForOpenAPIName bool
	}{
		{
			".a.b.C",
			"C",
			[]string{
				".a.b.C",
			},
			false,
		},
		{
			".a.b.C",
			"abC",
			[]string{
				".a.C",
				".a.b.C",
			},
			false,
		},
		{
			".a.b.C",
			"abC",
			[]string{
				".C",
				".a.C",
				".a.b.C",
			},
			false,
		},
		{
			".a.b.C",
			"a.b.C",
			[]string{
				".C",
				".a.C",
				".a.b.C",
			},
			true,
		},
	}

	for _, data := range tests {
		names := resolveFullyQualifiedNameToOpenAPINames(data.listOfFQMNs, data.useFQNForOpenAPIName)
		output := names[data.input]
		if output != data.output {
			t.Errorf("Expected fullyQualifiedNameToOpenAPIName(%v) to be %s but got %s",
				data.input, data.output, output)
		}
	}
}

func TestFQMNtoOpenAPIName(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"/test", "/test"},
		{"/{test}", "/{test}"},
		{"/{test=prefix/*}", "/{test}"},
		{"/{test=prefix/that/has/multiple/parts/to/it/*}", "/{test}"},
		{"/{test1}/{test2}", "/{test1}/{test2}"},
		{"/{test1}/{test2}/", "/{test1}/{test2}/"},
	}
	reg := descriptor.NewRegistry()
	reg.SetUseJSONNamesForFields(false)
	for _, data := range tests {
		actual := templateToOpenAPIPath(data.input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		if data.expected != actual {
			t.Errorf("Expected templateToOpenAPIPath(%v) = %v, actual: %v", data.input, data.expected, actual)
		}
	}
	reg.SetUseJSONNamesForFields(true)
	for _, data := range tests {
		actual := templateToOpenAPIPath(data.input, reg, generateFieldsForJSONReservedName(), generateMsgsForJSONReservedName())
		if data.expected != actual {
			t.Errorf("Expected templateToOpenAPIPath(%v) = %v, actual: %v", data.input, data.expected, actual)
		}
	}
}

func TestSchemaOfField(t *testing.T) {
	type test struct {
		field          *descriptor.Field
		refs           refMap
		expected       SchemaRef
		openAPIOptions *openapiconfig.OpenAPIOptions
	}

	jsonSchema := &openapi_options.Schema{
		Title:       "field title",
		Description: "field description",
	}

	var fieldOptions = new(descriptorpb.FieldOptions)
	proto.SetExtension(fieldOptions, openapi_options.E_Openapiv3Field, jsonSchema)

	var requiredField = []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED}
	var requiredFieldOptions = new(descriptorpb.FieldOptions)
	proto.SetExtension(requiredFieldOptions, annotations.E_FieldBehavior, requiredField)

	var outputOnlyField = []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY}
	var outputOnlyOptions = new(descriptorpb.FieldOptions)
	proto.SetExtension(outputOnlyOptions, annotations.E_FieldBehavior, outputOnlyField)

	tests := []test{
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name: proto.String("primitive_field"),
					Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:  proto.String("repeated_primitive_field"),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "array", Items: &SchemaRef{Value: &Schema{Type: "string"}}},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.FieldMask"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.Timestamp"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string", Format: "date-time"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.Duration"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.StringValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("repeated_wrapped_field"),
					TypeName: proto.String(".google.protobuf.StringValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "array", Items: &SchemaRef{Value: &Schema{Type: "string"}}},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.BytesValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string", Format: "byte"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.Int32Value"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "integer", Format: "int32"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.UInt32Value"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "integer", Format: "int64"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.Int64Value"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string", Format: "int64"},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.UInt64Value"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:   "string",
					Format: "uint64",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.FloatValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:   "number",
					Format: "float",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.DoubleValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:   "number",
					Format: "double",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.BoolValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type: "boolean",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.Struct"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type: "object",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.Value"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type: "object",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.ListValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:  "array",
					Items: &SchemaRef{Value: &Schema{Type: "object"}},
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("wrapped_field"),
					TypeName: proto.String(".google.protobuf.NullValue"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type: "string",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("message_field"),
					TypeName: proto.String(".example.Message"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				},
			},
			refs: refMap{".example.Message": struct{}{}},
			expected: SchemaRef{
				Value: &Schema{},
				Ref: "#/components/schemas/exampleMessage",
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("map_field"),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".example.Message.MapFieldEntry"),
					Options:  fieldOptions,
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:                 "object",
					AdditionalProperties: &SchemaRef{Value: &Schema{Type: "string"}},
					Title:                "field title",
					Description:          "field description",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:    proto.String("array_field"),
					Label:   descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Options: fieldOptions,
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:        "array",
					Items:       &SchemaRef{Value: &Schema{Type: "string"}},
					Title:       "field title",
					Description: "field description",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:    proto.String("primitive_field"),
					Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:    descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					Options: fieldOptions,
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{
					Type:        "integer",
					Format:      "int32",
					Title:       "field title",
					Description: "field description",
				},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("message_field"),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".example.Empty"),
					Options:  fieldOptions,
				},
			},
			refs: refMap{".example.Empty": struct{}{}},
			expected: SchemaRef{
				Ref: "#/components/schemas/exampleEmpty",
				Value: &Schema{
					Title:       "field title",
					Description: "field description",
				},
			},
		},
		//{
		//	field: &descriptor.Field{
		//		FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
		//			Name:     proto.String("map_field"), // should be called map_field_option but it's not valid map field name
		//			Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
		//			Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		//			TypeName: proto.String(".example.Message.MapFieldEntry"),
		//		},
		//	},
		//	openAPIOptions: &openapiconfig.OpenAPIOptions{
		//		Field: []*openapiconfig.OpenAPIFieldOption{
		//			{
		//				Field:  "example.Message.map_field",
		//				Option: jsonSchema,
		//			},
		//		},
		//	},
		//	refs: make(refMap),
		//	expected: SchemaRef{
		//		Value: &Schema{
		//			Type:                 "object",
		//			AdditionalProperties: &SchemaRef{Value: &Schema{Type: "string"}},
		//			Title:                "field title",
		//			Description:          "field description",
		//		},
		//	},
		//},
		//{
		//	field: &descriptor.Field{
		//		FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
		//			Name:  proto.String("array_field_option"),
		//			Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
		//			Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		//		},
		//	},
		//	openAPIOptions: &openapiconfig.OpenAPIOptions{
		//		Field: []*openapiconfig.OpenAPIFieldOption{
		//			{
		//				Field:  "example.Message.array_field_option",
		//				Option: jsonSchema,
		//			},
		//		},
		//	},
		//	refs: make(refMap),
		//	expected: SchemaRef{
		//		Value: &Schema{
		//			Type:        "array",
		//			Items:       &SchemaRef{Value: &Schema{Type: "string"}},
		//			Title:       "field title",
		//			Description: "field description",
		//		},
		//	},
		//},
		//{
		//	field: &descriptor.Field{
		//		FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
		//			Name:  proto.String("primitive_field_option"),
		//			Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		//			Type:  descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
		//		},
		//	},
		//	openAPIOptions: &openapiconfig.OpenAPIOptions{
		//		Field: []*openapiconfig.OpenAPIFieldOption{
		//			{
		//				Field:  "example.Message.primitive_field_option",
		//				Option: jsonSchema,
		//			},
		//		},
		//	},
		//	refs: make(refMap),
		//	expected: SchemaRef{
		//		Value: &Schema{
		//			Type:        "integer",
		//			Format:      "int32",
		//			Title:       "field title",
		//			Description: "field description",
		//		},
		//	},
		//},
		//{
		//	field: &descriptor.Field{
		//		FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
		//			Name:     proto.String("message_field_option"),
		//			Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		//			Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		//			TypeName: proto.String(".example.Empty"),
		//		},
		//	},
		//	openAPIOptions: &openapiconfig.OpenAPIOptions{
		//		Field: []*openapiconfig.OpenAPIFieldOption{
		//			{
		//				Field:  "example.Message.message_field_option",
		//				Option: jsonSchema,
		//			},
		//		},
		//	},
		//	refs: refMap{".example.Empty": struct{}{}},
		//	expected: SchemaRef{
		//		Ref:   "#/components/exampleEmpty",
		//		Value: &Schema{Title: "field title", Description: "field description"},
		//	},
		//},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:    proto.String("required_via_field_behavior_field"),
					Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Options: requiredFieldOptions,
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string", Required: []string{"required_via_field_behavior_field"}},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:    proto.String("readonly_via_field_behavior_field"),
					Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Options: outputOnlyOptions,
				},
			},
			refs: make(refMap),
			expected: SchemaRef{
				Value: &Schema{Type: "string", ReadOnly: true},
			},
		},
		{
			field: &descriptor.Field{
				FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("message_field"),
					TypeName: proto.String(".example.Message"),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					Options:  requiredFieldOptions,
				},
			},
			refs: refMap{".example.Message": struct{}{}},
			expected: SchemaRef{
				Value: &Schema{},
				Ref: "#/components/schemas/exampleMessage",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.field.FieldDescriptorProto.GetName(), func(t *testing.T) {
			reg := descriptor.NewRegistry()
			req := &pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{
					{
						Name:    proto.String("third_party/google.proto"),
						Package: proto.String("google.protobuf"),
						Options: &descriptorpb.FileOptions{
							GoPackage: proto.String("third_party/google"),
						},
						MessageType: []*descriptorpb.DescriptorProto{
							protodesc.ToDescriptorProto((&structpb.Struct{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&structpb.Value{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&structpb.ListValue{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&field_mask.FieldMask{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&timestamppb.Timestamp{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&durationpb.Duration{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.StringValue{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.BytesValue{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.Int32Value{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.UInt32Value{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.Int64Value{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.UInt64Value{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.FloatValue{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor()),
							protodesc.ToDescriptorProto((&wrapperspb.BoolValue{}).ProtoReflect().Descriptor()),
						},
						EnumType: []*descriptorpb.EnumDescriptorProto{
							protodesc.ToEnumDescriptorProto(structpb.NullValue(0).Descriptor()),
						},
					},
					{
						SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
						Name:           proto.String("example.proto"),
						Package:        proto.String("example"),
						Dependency:     []string{"third_party/google.proto"},
						Options: &descriptorpb.FileOptions{
							GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
						},
						MessageType: []*descriptorpb.DescriptorProto{
							{
								Name: proto.String("Message"),
								Field: []*descriptorpb.FieldDescriptorProto{
									{
										Name:   proto.String("value"),
										Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
										Number: proto.Int32(1),
									},
									func() *descriptorpb.FieldDescriptorProto {
										fd := test.field.FieldDescriptorProto
										fd.Number = proto.Int32(2)
										return fd
									}(),
								},
								NestedType: []*descriptorpb.DescriptorProto{
									{
										Name:    proto.String("MapFieldEntry"),
										Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
										Field: []*descriptorpb.FieldDescriptorProto{
											{
												Name:   proto.String("key"),
												Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
												Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
												Number: proto.Int32(1),
											},
											{
												Name:   proto.String("value"),
												Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
												Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
												Number: proto.Int32(2),
											},
										},
									},
								},
							},
							{
								Name: proto.String("Empty"),
							},
						},
						EnumType: []*descriptorpb.EnumDescriptorProto{
							{
								Name: proto.String("MessageType"),
								Value: []*descriptorpb.EnumValueDescriptorProto{
									{
										Name:   proto.String("MESSAGE_TYPE_1"),
										Number: proto.Int32(0),
									},
								},
							},
						},
						Service: []*descriptorpb.ServiceDescriptorProto{},
					},
				},
			}
			err := reg.Load(req)
			if err != nil {
				t.Errorf("failed to reg.Load(req): %v", err)
			}

			// set field's parent message pointer to message so field can resolve its FQFN
			test.field.Message = &descriptor.Message{
				DescriptorProto: req.ProtoFile[1].MessageType[0],
				File: &descriptor.File{
					FileDescriptorProto: req.ProtoFile[1],
				},
			}

			if test.openAPIOptions != nil {
				if err := reg.RegisterOpenAPIOptions(test.openAPIOptions); err != nil {
					t.Fatalf("failed to register OpenAPI options: %s", err)
				}
			}

			refs := make(refMap)
			actual := schemaOfField(test.field, reg, refs)
			expectedSchemaObject := test.expected
			if e, a := expectedSchemaObject, actual; !reflect.DeepEqual(a, e) {
				t.Errorf("Expected schemaOfField(%v) = \n%#+v, actual: \n%#+v", test.field, e, a)
			}
			if !reflect.DeepEqual(refs, test.refs) {
				t.Errorf("Expected schemaOfField(%v) to add refs %v, not %v", test.field, test.refs, refs)
			}
		})
	}
}

func TestRenderMessagesAsComponentSchemas(t *testing.T) {
	it := require.New(t)
	requiredFieldSchema := &openapi_options.Schema{
		Title:       "field title",
		Description: "field description",
		Required:    []string{"aRequiredField"},
	}
	var requiredField = new(descriptorpb.FieldOptions)
	proto.SetExtension(requiredField, openapi_options.E_Openapiv3Field, requiredFieldSchema)

	var fieldBehaviorRequired = []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED}
	var requiredFieldOptions = new(descriptorpb.FieldOptions)
	proto.SetExtension(requiredFieldOptions, annotations.E_FieldBehavior, fieldBehaviorRequired)

	var fieldBehaviorOutputOnlyField = []annotations.FieldBehavior{annotations.FieldBehavior_OUTPUT_ONLY}
	var fieldBehaviorOutputOnlyOptions = new(descriptorpb.FieldOptions)
	proto.SetExtension(fieldBehaviorOutputOnlyOptions, annotations.E_FieldBehavior, fieldBehaviorOutputOnlyField)

	tests := []struct {
		descr           string
		msgDescs        []*descriptorpb.DescriptorProto
		schema          map[string]openapi_options.Schema // per-message schema to add
		expectedSchemas Schemas
		openAPIOptions  *openapiconfig.OpenAPIOptions
		excludedFields  []*descriptor.Field
	}{
		{
			descr: "no OpenAPI options",
			msgDescs: []*descriptorpb.DescriptorProto{
				{Name: proto.String("Message")},
			},
			schema: map[string]openapi_options.Schema{},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type: "object",
					},
				},
			},
		},
		{
			descr: "example option",
			msgDescs: []*descriptorpb.DescriptorProto{
				{Name: proto.String("Message")},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					Example: &openapi_options.Any{
						Value: &anypb.Any{
							Value: []byte(`{"foo":"bar"}`),
						},
					},
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type:    "object",
						Example: json.RawMessage(`{"foo":"bar"}`),
					},
				},
			},
		},
		{
			descr: "example option with something non-json",
			msgDescs: []*descriptorpb.DescriptorProto{
				{Name: proto.String("Message")},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					Example: &openapi_options.Any{
						Value: &anypb.Any{
							Value: []byte(`XXXX anything goes XXXX`),
						},
					},
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type:    "object",
						Example: json.RawMessage((`XXXX anything goes XXXX`)),
					},
				},
			},
		},
		{
			descr: "external docs option",
			msgDescs: []*descriptorpb.DescriptorProto{
				{Name: proto.String("Message")},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					ExternalDocs: &openapi_options.ExternalDocs{
						Description: "glorious docs",
						Url:         "https://nada",
					},
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type: "object",
						ExternalDocs: &ExternalDocs{
							Description: "glorious docs",
							URL:         "https://nada",
						},
					},
				},
			},
		},
		{
			descr: "JSONSchema options",
			msgDescs: []*descriptorpb.DescriptorProto{
				{Name: proto.String("Message")},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					Title:            "title",
					Description:      "desc",
					MultipleOf:       100,
					Maximum:          101,
					ExclusiveMaximum: true,
					Minimum:          1,
					ExclusiveMinimum: true,
					MaxLength:        10,
					MinLength:        3,
					Pattern:          "[a-z]+",
					MaxItems:         20,
					MinItems:         2,
					UniqueItems:      true,
					MaxProperties:    33,
					MinProperties:    22,
					Required:         []string{"req"},
					ReadOnly:         true,
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type:             "object",
						Title:            "title",
						Description:      "desc",
						MultipleOf:       100,
						Maximum:          101,
						ExclusiveMaximum: true,
						Minimum:          1,
						ExclusiveMinimum: true,
						MaxLength:        10,
						MinLength:        3,
						Pattern:          "[a-z]+",
						MaxItems:         20,
						MinItems:         2,
						UniqueItems:      true,
						MaxProperties:    33,
						MinProperties:    22,
						Required:         []string{"req"},
						ReadOnly:         true,
					},
				},
			},
		},
		//{
		//	descr: "JSONSchema options from registry",
		//	msgDescs: []*descriptorpb.DescriptorProto{
		//		{Name: proto.String("Message")},
		//	},
		//	openAPIOptions: &openapiconfig.OpenAPIOptions{
		//		Message: []*openapiconfig.OpenAPIMessageOption{
		//			{
		//				Message: "example.Message",
		//				Option: &openapi_options.Schema{
		//					JsonSchema: &openapi_options.JSONSchema{
		//						Title:            "title",
		//						Description:      "desc",
		//						MultipleOf:       100,
		//						Maximum:          101,
		//						ExclusiveMaximum: true,
		//						Minimum:          1,
		//						ExclusiveMinimum: true,
		//						MaxLength:        10,
		//						MinLength:        3,
		//						Pattern:          "[a-z]+",
		//						MaxItems:         20,
		//						MinItems:         2,
		//						UniqueItems:      true,
		//						MaxProperties:    33,
		//						MinProperties:    22,
		//						Required:         []string{"req"},
		//						ReadOnly:         true,
		//					},
		//				},
		//			},
		//		},
		//	},
		//	expectedSchemas: map[string]openapiSchemaObject{
		//		"Message": {
		//			schemaCore: schemaCore{
		//				Type: "object",
		//			},
		//			Title:            "title",
		//			Description:      "desc",
		//			MultipleOf:       100,
		//			Maximum:          101,
		//			ExclusiveMaximum: true,
		//			Minimum:          1,
		//			ExclusiveMinimum: true,
		//			MaxLength:        10,
		//			MinLength:        3,
		//			Pattern:          "[a-z]+",
		//			MaxItems:         20,
		//			MinItems:         2,
		//			UniqueItems:      true,
		//			MaxProperties:    33,
		//			MinProperties:    22,
		//			Required:         []string{"req"},
		//			ReadOnly:         true,
		//		},
		//	},
		//},
		{
			descr: "JSONSchema with required properties",
			msgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("Message"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:    proto.String("aRequiredField"),
							Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number:  proto.Int32(1),
							Options: requiredField,
						},
					},
				},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					Title:       "title",
					Description: "desc",
					Required:    []string{"req"},
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type:        "object",
						Title:       "title",
						Description: "desc",
						Required:    []string{"req", "aRequiredField"},
						Properties: Schemas{
							"aRequiredField": &SchemaRef{
								Value: &Schema{
									Type:        "string",
									Description: "field description",
									Title:       "field title",
									Required:    []string{"aRequiredField"},
								},
							},
						},
					},
				},
			},
		},
		{
			descr: "JSONSchema with excluded fields",
			msgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("Message"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:    proto.String("aRequiredField"),
							Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number:  proto.Int32(1),
							Options: requiredField,
						},
						{
							Name:   proto.String("anExcludedField"),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number: proto.Int32(2),
						},
					},
				},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					Title:       "title",
					Description: "desc",
					Required:    []string{"req"},
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type:        "object",
						Title:       "title",
						Description: "desc",
						Required:    []string{"req", "aRequiredField"},
						Properties: Schemas{
							"aRequiredField": &SchemaRef{
								Value: &Schema{

									Type:        "string",
									Description: "field description",
									Title:       "field title",
									Required:    []string{"aRequiredField"},
								},
							},
						},
					},
				},
			},
			excludedFields: []*descriptor.Field{
				{
					FieldDescriptorProto: &descriptorpb.FieldDescriptorProto{
						Name: strPtr("anExcludedField"),
					},
				},
			},
		},
		{
			descr: "JSONSchema with one of fields",
			msgDescs: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("Message"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:       proto.String("line_num"),
							Type:       descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
							Number:     proto.Int32(1),
							OneofIndex: proto.Int32(0),
						},
						{
							Name:       proto.String("lang"),
							Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							Number:     proto.Int32(2),
							OneofIndex: proto.Int32(0),
						},
					},
					OneofDecl: []*descriptorpb.OneofDescriptorProto{
						{
							Name: proto.String("code"),
						},
					},
				},
			},
			schema: map[string]openapi_options.Schema{
				"Message": {
					Title:       "title",
					Description: "desc",
				},
			},
			expectedSchemas: map[string]*SchemaRef{
				"Message": {
					Value: &Schema{
						Type:        "object",
						Title:       "title",
						Description: "desc",
						Properties: Schemas{
							"line_num": &SchemaRef{
								Value: &Schema{
									Type:   "string",
									Format: "int64",
								},
							},
							"lang": &SchemaRef{
								Value: &Schema{
									Type: "string",
								},
							},
						},
					},
				},
			},
		},
		//{
		//	descr: "JSONSchema with required properties via field_behavior",
		//	msgDescs: []*descriptorpb.DescriptorProto{
		//		{
		//			Name: proto.String("Message"),
		//			Field: []*descriptorpb.FieldDescriptorProto{
		//				{
		//					Name:    proto.String("aRequiredField"),
		//					Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		//					Number:  proto.Int32(1),
		//					Options: requiredFieldOptions,
		//				},
		//				{
		//					Name:    proto.String("aOutputOnlyField"),
		//					Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		//					Number:  proto.Int32(2),
		//					Options: fieldBehaviorOutputOnlyOptions,
		//				},
		//			},
		//		},
		//	},
		//	schema: map[string]openapi_options.Schema{
		//		"Message": {
		//			JsonSchema: &openapi_options.JSONSchema{
		//				Title:       "title",
		//				Description: "desc",
		//				Required:    []string{"req"},
		//			},
		//		},
		//	},
		//	expectedSchemas: map[string]openapiSchemaObject{
		//		"Message": {
		//			schemaCore: schemaCore{
		//				Type: "object",
		//			},
		//			Title:       "title",
		//			Description: "desc",
		//			Required:    []string{"req", "aRequiredField"},
		//			Properties: &openapiSchemaObjectProperties{
		//				{
		//					Key: "aRequiredField",
		//					Value: openapiSchemaObject{
		//						schemaCore: schemaCore{
		//							Type: "string",
		//						},
		//						Required: []string{"aRequiredField"},
		//					},
		//				},
		//				{
		//					Key: "aOutputOnlyField",
		//					Value: openapiSchemaObject{
		//						schemaCore: schemaCore{
		//							Type: "string",
		//						},
		//						ReadOnly: true,
		//					},
		//				},
		//			},
		//		},
		//	},
		//},
	}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {

			msgs := []*descriptor.Message{}
			for _, msgdesc := range test.msgDescs {
				msgdesc.Options = &descriptorpb.MessageOptions{}
				msgs = append(msgs, &descriptor.Message{DescriptorProto: msgdesc})
			}

			reg := descriptor.NewRegistry()
			file := descriptor.File{
				FileDescriptorProto: &descriptorpb.FileDescriptorProto{
					SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
					Name:           proto.String("example.proto"),
					Package:        proto.String("example"),
					Dependency:     []string{},
					MessageType:    test.msgDescs,
					EnumType:       []*descriptorpb.EnumDescriptorProto{},
					Service:        []*descriptorpb.ServiceDescriptorProto{},
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String("github.com/grpc-ecosystem/grpc-gateway/runtime/internal/examplepb;example"),
					},
				},
				Messages: msgs,
			}
			err := reg.Load(&pluginpb.CodeGeneratorRequest{
				ProtoFile: []*descriptorpb.FileDescriptorProto{file.FileDescriptorProto},
			})
			if err != nil {
				t.Fatalf("failed to load code generator request: %v", err)
			}

			msgMap := map[string]*descriptor.Message{}
			for _, d := range test.msgDescs {
				name := d.GetName()
				msg, err := reg.LookupMsg("example", name)
				if err != nil {
					t.Fatalf("lookup message %v: %v", name, err)
				}
				msgMap[msg.FQMN()] = msg

				if schema, ok := test.schema[name]; ok {
					proto.SetExtension(d.Options, openapi_options.E_Openapiv3Schema, &schema)
				}
			}

			if test.openAPIOptions != nil {
				if err := reg.RegisterOpenAPIOptions(test.openAPIOptions); err != nil {
					t.Fatalf("failed to register OpenAPI options: %s", err)
				}
			}

			refs := make(refMap)
			actual := Components{
				Schemas: make(Schemas),
			}
			err = renderMessagesToComponentsSchemas(msgMap, &actual, reg, refs, test.excludedFields)
			it.NoError(err)

			it.Equal(test.expectedSchemas, actual.Schemas)
		})
	}
}

func crossLinkFixture(f *descriptor.File) *descriptor.File {
	for _, m := range f.Messages {
		m.File = f
	}
	for _, svc := range f.Services {
		svc.File = f
		for _, m := range svc.Methods {
			m.Service = svc
			for _, b := range m.Bindings {
				b.Method = m
				for _, param := range b.PathParams {
					param.Method = m
				}
			}
		}
	}
	return f
}

func reqFromFile(f *descriptor.File) *pluginpb.CodeGeneratorRequest {
	return &pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			f.FileDescriptorProto,
		},
		FileToGenerate: []string{f.GetName()},
	}
}

func strPtr(s string) *string {
	return &s
}