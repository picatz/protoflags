# protoflags

`protoflags` is a small Go library that lets you automatically generate 
[`github.com/spf13/pflag`](https://github.com/spf13/pflag) command-line flags 
for your protocol buffer (Protobuf) messages and then apply those flags back 
onto the Protobuf  message at runtime. This is especially handy for building 
CLI tools or microservices that accept user inputs to  populate proto messages.

- 🔧 No manual flag wiring — let the library handle the repetitive tasks consistently.
- 🔍 Deeply nested messages — it can recurse into submessages.
- 🔁 Supports repeated scalar fields with a CSV-like flag format.
- 🚫 Ignore fields or entire submessages by specifying dot-based ignore protopaths.

## Installation

```sh
go get github.com/picatz/protoflags@latest
```

## Quickstart

1. Define a Protobuf message.

```proto
syntax = "proto3";

package example;

message Example {
    int32 id = 1;
    string name = 2;
    bool is_active = 3;
}
```

2. Generate the Go code (via `buf` or `protoc` + your preferred plugins).
3. Generate flags from the message:

```go
import (
    "fmt"
    "github.com/picatz/protoflags"
    "github.com/spf13/pflag"
    "google.golang.org/protobuf/proto"
)

func main() {
    // Create a new proto message
    ex := &example.Example{}

    // Generate flags, ignoring no fields
    fs, err := protoflags.FlagsFromMessage(ex, nil)
    if err != nil {
        panic(err)
    }

    // Parse user input flags, e.g. from os.Args
    fs.Parse([]string{
        "--id 123",
        "--name example",
        "--is-active",
    })

    // Apply the parsed flags back onto ex
    err = protoflags.ApplyFlagsToMessage(fs, ex)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Populated example: %+v\n", ex)
}
```

4. Now `ex` has the fields set from flags. For example:

```proto
{
    id: 123,
    name: "example",
    is_active: true,
}
```

## Features

### 1. Automatically Generate pflag Flags

Every scalar (integer, string, bool, float, enum) field in your Protobuf 
message gets its own CLI flag. Repeated scalars are supported with a 
comma-separated format.

- Example: a repeated string `tags = 4;` field becomes `--tags="tag1,tag2"`

### 2. Nested Messages

If your proto message has nested submessages, protoflags can recurse into them. 
The generated flags are prefixed with "parent-field-name-".

For instance, if `Example` has a nested submessage `Nested` with its own fields:

- `nested.id`  =>  `--nested-id`
- `nested.name` =>  `--nested-name`

### 3. Ignoring Fields (or Submessages)

You can specify “ignore paths” to skip generating (and applying) flags for 
certain fields or entire submessages. 

## Examples

Below is a simple example of generating flags from a Protobuf message and then applying them back onto the message: 

```go
ex := &example.Example{
    Id:       1,
    Name:     "example",
    IsActive: true,
}

fs, _ := protoflags.FlagsFromMessage(ex, nil) // no ignore paths

// Print out the flags that got generated
fs.VisitAll(func(f *pflag.Flag) {
    fmt.Printf("%s: %s %s\n", f.Name, f.Value.Type(), f.Usage)
})
```

Which might output something like:

```
id: int  (Leading comments in the .proto, if any)
name: string
is-active: bool
```

Another example, ignoring certain paths:

```go
fs, _ := protoflags.FlagsFromMessage(
    &example.Example{},
    []string{"id", "name"},  // ignorePaths
)

// The 'id' and 'name' flags won't exist
fs.VisitAll(func(f *pflag.Flag) {
    fmt.Println(f.Name)
    // "is-active" ...
})
```

## Caveats and Limitations

There are lots of edge cases and limitations to consider when using protoflags. Here are a few:

1. **Repeated Messages**: Repeated submessages (e.g. `repeated Nested nested = 5;`) are not supported by default. The library currently returns an error if it encounters a repeated message, since mapping that to CLI flags is non-trivial.
2. **Maps**: Protobuf `map<K,V>` fields are also not handled. You may either skip them or implement a custom approach.
3. **Enum Values**: For enum fields, the user types the enum name in the CLI flags. For example, `--color=RED`.
4. **Infinite Recursion**: The library tracks which message descriptors it has visited (via a visited map) in case your proto messages reference themselves in cycles. This prevents an infinite recursion scenario.
5. **Field Name Collisions**: If two different fields become the same kebab-case name, you will have collisions. You can rename your fields or rename the generated flags.

Limitations may be addressed in future versions of the library, but for now, you may need to work around them.