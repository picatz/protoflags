package example_test

import (
	"fmt"
	"testing"

	"github.com/picatz/protoflags"
	"github.com/picatz/protoflags/internal/example"
	"github.com/shoenig/test/must"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

func TestExample(t *testing.T) {
	ex := &example.Example{
		Id:       1,
		Name:     "example",
		IsActive: true,
		Tags: []string{
			"tag1",
			"tag2",
		},
		Nested: &example.Example_Nested{
			Id:   2,
			Name: "nested",
		},
	}
	fmt.Println(ex)

	fs, err := protoflags.FlagsFromMessage(ex, nil)
	must.NoError(t, err)
	fs.VisitAll(func(f *pflag.Flag) {
		fmt.Printf("%s: %s %s\n", f.Name, f.Value.Type(), f.Usage)
	})
}

func TestFlagsFromMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         proto.Message
		ignorePaths []string
		check       func(t *testing.T, fs *pflag.FlagSet)
	}{
		{
			name: "no ignore paths",
			msg:  &example.Example{},
			check: func(t *testing.T, fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					fmt.Println(f.Name, f.Value.Type(), f.DefValue, f.Usage)
					must.SliceContains(t,
						[]string{
							"id",
							"name",
							"is-active",
							"tags",
							"nested-id",
							"nested-name",
							"nested-nested-id",
							"nested-nested-name",
						},
						f.Name,
					)
				})
			},
		},
		{
			name: "with ignore id and name",
			msg:  &example.Example{},
			ignorePaths: []string{
				"id",
				"name",
			},
			check: func(t *testing.T, fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					fmt.Println(f.Name, f.Value.Type(), f.DefValue, f.Usage)
					must.SliceContains(t,
						[]string{
							"is-active",
							"tags",
							"nested-id",
							"nested-name",
							"nested-nested-id",
							"nested-nested-name",
						},
						f.Name,
					)
				})
			},
		},
		{
			name: "with ignore nested",
			msg:  &example.Example{},
			ignorePaths: []string{
				"nested",
				"nested_nested",
			},
			check: func(t *testing.T, fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					fmt.Println(f.Name, f.Value.Type(), f.DefValue, f.Usage)
					must.SliceContains(t,
						[]string{
							"id",
							"name",
							"is-active",
							"tags",
						},
						f.Name,
					)
				})
			},
		},
		{
			name: "with ignore nested id",
			msg:  &example.Example{},
			ignorePaths: []string{
				"nested.id",
			},
			check: func(t *testing.T, fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					fmt.Println(f.Name, f.Value.Type(), f.DefValue, f.Usage)
					must.SliceContains(t,
						[]string{
							"id",
							"name",
							"is-active",
							"tags",
							"nested-name",
							"nested-nested-id",
							"nested-nested-name",
						},
						f.Name,
					)
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := protoflags.FlagsFromMessage(test.msg, test.ignorePaths)
			must.NoError(t, err)

			test.check(t, fs)
		})
	}
}

func TestApplyFlagsToMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         proto.Message
		ignorePaths []string
		flags       map[string]string
		check       func(t *testing.T, msg proto.Message)
	}{
		{
			name: "task",
			msg:  &example.Example{},
			flags: map[string]string{
				"id":        "2",
				"name":      "test",
				"is-active": "true",
				"tags":      "tag1,tag2",
			},
			check: func(t *testing.T, msg proto.Message) {
				e := msg.(*example.Example)
				must.Eq(t, 2, e.GetId())
				must.Eq(t, "test", e.GetName())
				must.Eq(t, true, e.GetIsActive())
				must.Eq(t, []string{"tag1", "tag2"}, e.GetTags())
			},
		},
		{
			name: "create task request",
			msg:  &example.Example{},
			flags: map[string]string{
				"name": "test",
			},
			check: func(t *testing.T, msg proto.Message) {
				e := msg.(*example.Example)
				must.Eq(t, "test", e.GetName())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := protoflags.FlagsFromMessage(test.msg, test.ignorePaths)
			must.NoError(t, err)

			args := []string{}
			for name, value := range test.flags {
				args = append(args, fmt.Sprintf("--%s=%s", name, value))
			}

			err = fs.Parse(args)
			must.NoError(t, err)

			err = protoflags.ApplyFlagsToMessage(fs, test.msg)
			must.NoError(t, err)

			test.check(t, test.msg)
		})
	}
}

/*
package cli_test

import (
	"fmt"
	"testing"

	"github.com/picatz/go-connect-aws-lambda-dynamodb/cmd/donezo/cli"
	v1 "github.com/picatz/go-connect-aws-lambda-dynamodb/pkg/donezo/v1"
	"github.com/shoenig/test/must"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

func TestFlagsFromMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         proto.Message
		ignorePaths []string
		check       func(t *testing.T, fs *pflag.FlagSet)
	}{
		{
			name: "task",
			msg:  &v1.Task{},
			check: func(t *testing.T, fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					fmt.Println(f.Name, f.Value.Type(), f.DefValue, f.Usage)
					must.SliceContains(t,
						[]string{
							"id",
							"project-id",
							"organization-id",
							"title",
							"description",
							"completed",
							"tags",
						},
						f.Name,
					)
				})
			},
		},
		{
			name:        "create task request",
			msg:         &v1.CreateTaskRequest{},
			ignorePaths: []string{"update_mask"},
			check: func(t *testing.T, fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					fmt.Println(f.Name, f.Value.Type(), f.DefValue, f.Usage)
					must.StrNotContains(t, f.Name, "update-mask")
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := cli.FlagsFromMessage(test.msg.ProtoReflect().Descriptor(), test.ignorePaths)
			must.NoError(t, err)

			test.check(t, fs)
		})
	}
}

func TestApplyFlagsToMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         proto.Message
		ignorePaths []string
		flags       map[string]string
		check       func(t *testing.T, msg proto.Message)
	}{
		{
			name: "task",
			msg:  &v1.Task{},
			flags: map[string]string{
				"id":              "123",
				"project-id":      "456",
				"organization-id": "789",
				"title":           "test",
				"description":     "test",
				"completed":       "true",
				"tags":            "tag1,tag2",
			},
			check: func(t *testing.T, msg proto.Message) {
				task := msg.(*v1.Task)
				must.Eq(t, "123", task.GetId())
				must.Eq(t, "456", task.GetProjectId())
				must.Eq(t, "789", task.GetOrganizationId())
				must.Eq(t, "test", task.GetTitle())
				must.Eq(t, "test", task.GetDescription())
				must.Eq(t, true, task.GetCompleted())
				must.Eq(t, []string{"tag1", "tag2"}, task.GetTags())
			},
		},
		{
			name: "create task request",
			msg:  &v1.CreateTaskRequest{},
			flags: map[string]string{
				"title": "test",
			},
			check: func(t *testing.T, msg proto.Message) {
				req := msg.(*v1.CreateTaskRequest)
				must.Eq(t, "test", req.GetTitle())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := cli.FlagsFromMessage(test.msg.ProtoReflect().Descriptor(), test.ignorePaths)
			must.NoError(t, err)

			args := []string{}
			for name, value := range test.flags {
				args = append(args, fmt.Sprintf("--%s=%s", name, value))
			}

			err = fs.Parse(args)
			must.NoError(t, err)

			err = cli.ApplyFlagsToMessage(fs, test.msg)
			must.NoError(t, err)

			test.check(t, test.msg)
		})
	}
}


*/
