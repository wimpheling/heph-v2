package main

import (
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestFieldCommentsAreRequired(t *testing.T) {
	for _, documented := range []bool{true, false} {
		file := &descriptorpb.FileDescriptorProto{Name: proto.String("test.proto"), Package: proto.String("test"), Syntax: proto.String("proto3"), Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/test;test")}, MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Input"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("value"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}}}, SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{Path: []int32{4, 0}, Span: []int32{0, 0, 0}, LeadingComments: proto.String(" Input is documented.")}}}}
		if documented {
			file.SourceCodeInfo.Location = append(file.SourceCodeInfo.Location, &descriptorpb.SourceCodeInfo_Location{Path: []int32{4, 0, 2, 0}, Span: []int32{0, 0, 0}, LeadingComments: proto.String(" Value is documented.")})
		}
		plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{FileToGenerate: []string{"test.proto"}, ProtoFile: []*descriptorpb.FileDescriptorProto{file}})
		if err != nil {
			t.Fatal(err)
		}
		err = fields(plugin.Files[0].Messages[0])
		if (err == nil) != documented {
			t.Fatalf("documented=%v, validation=%v", documented, err)
		}
	}
}
