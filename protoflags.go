package protoflags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ApplyFlagsToMessage applies flag values from fs into msg.
func ApplyFlagsToMessage(
	fs *pflag.FlagSet,
	msg proto.Message,
) error {
	return applyFlags(fs, msg, "")
}

// applyFlags applies the values from the provided FlagSet to the given
// protobuf message. It assumes a naming scheme matching the one used
// by FlagsFromMessage (kebab-case field names, possibly prefixed for nested
// messages).
func applyFlags(
	fs *pflag.FlagSet,
	m proto.Message,
	prefix string,
) error {
	msg := m.ProtoReflect()
	md := msg.Descriptor()

	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)

		fieldKebab := toKebabCase(fd.TextName())
		flagName := prefix + fieldKebab

		switch {
		case fd.IsList():
			// If the flag wasn't set, skip.
			if !fs.Changed(flagName) {
				continue
			}

			rawVal, err := fs.GetString(flagName)
			if err != nil {
				return err
			}

			// Split the CLI input on commas, trim spaces
			items := strings.Split(rawVal, ",")
			for i := range items {
				items[i] = strings.TrimSpace(items[i])
			}

			// Prepare the list to store parsed values
			list := msg.Mutable(fd).List()
			// Clear existing contents (in case we are re-setting)
			list.Truncate(0)

			// Parse each item based on field kind
			switch fd.Kind() {
			case protoreflect.StringKind:
				for _, v := range items {
					list.Append(protoreflect.ValueOfString(v))
				}
			case protoreflect.Int32Kind, protoreflect.Int64Kind:
				for _, v := range items {
					num, parseErr := strconv.ParseInt(v, 10, 64)
					if parseErr != nil {
						return fmt.Errorf("invalid integer %q for field %s: %w", v, fd.FullName(), parseErr)
					}
					list.Append(protoreflect.ValueOfInt64(num))
				}
			case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
				for _, v := range items {
					num, parseErr := strconv.ParseUint(v, 10, 64)
					if parseErr != nil {
						return fmt.Errorf("invalid unsigned integer %q for field %s: %w", v, fd.FullName(), parseErr)
					}
					if fd.Kind() == protoreflect.Uint32Kind {
						list.Append(protoreflect.ValueOfUint32(uint32(num)))
					} else {
						list.Append(protoreflect.ValueOfUint64(num))
					}
				}
			case protoreflect.BoolKind:
				for _, v := range items {
					b, parseErr := strconv.ParseBool(v)
					if parseErr != nil {
						return fmt.Errorf("invalid bool %q for field %s: %w", v, fd.FullName(), parseErr)
					}
					list.Append(protoreflect.ValueOfBool(b))
				}
			case protoreflect.FloatKind, protoreflect.DoubleKind:
				for _, v := range items {
					f, parseErr := strconv.ParseFloat(v, 64)
					if parseErr != nil {
						return fmt.Errorf("invalid float %q for field %s: %w", v, fd.FullName(), parseErr)
					}
					if fd.Kind() == protoreflect.FloatKind {
						list.Append(protoreflect.ValueOfFloat32(float32(f)))
					} else {
						list.Append(protoreflect.ValueOfFloat64(f))
					}
				}
			case protoreflect.EnumKind:
				enumDesc := fd.Enum()
				for _, v := range items {
					// e.g. if user typed "RED,BLUE"
					ev := enumDesc.Values().ByName(protoreflect.Name(v))
					if ev == nil {
						return fmt.Errorf("unknown enum value %q for field %s", v, fd.FullName())
					}
					list.Append(protoreflect.ValueOfEnum(ev.Number()))
				}
			default:
				return fmt.Errorf("unsupported repeated field kind %q for field %s", fd.Kind(), fd.FullName())
			}

		case fd.Kind() == protoreflect.MessageKind && !fd.IsMap():
			// Recurse with prefix "field-name-"
			subPrefix := flagName + "-"
			subMsg := msg.Mutable(fd).Message()
			if err := applyFlags(fs, subMsg.Interface(), subPrefix); err != nil {
				return err
			}

		default:
			// If the flag wasn't set, skip.
			if !fs.Changed(flagName) {
				continue
			}

			switch fd.Kind() {
			case protoreflect.StringKind:
				val, err := fs.GetString(flagName)
				if err != nil {
					return err
				}
				msg.Set(fd, protoreflect.ValueOfString(val))

			case protoreflect.Int32Kind, protoreflect.Int64Kind:
				val, err := fs.GetInt(flagName)
				if err != nil {
					return err
				}
				msg.Set(fd, protoreflect.ValueOfInt64(int64(val)))

			case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
				val, err := fs.GetUint(flagName)
				if err != nil {
					return err
				}
				if fd.Kind() == protoreflect.Uint32Kind {
					msg.Set(fd, protoreflect.ValueOfUint32(uint32(val)))
				} else {
					msg.Set(fd, protoreflect.ValueOfUint64(uint64(val)))
				}

			case protoreflect.BoolKind:
				val, err := fs.GetBool(flagName)
				if err != nil {
					return err
				}
				msg.Set(fd, protoreflect.ValueOfBool(val))

			case protoreflect.FloatKind, protoreflect.DoubleKind:
				val, err := fs.GetFloat64(flagName)
				if err != nil {
					return err
				}
				if fd.Kind() == protoreflect.FloatKind {
					msg.Set(fd, protoreflect.ValueOfFloat32(float32(val)))
				} else {
					msg.Set(fd, protoreflect.ValueOfFloat64(val))
				}

			case protoreflect.EnumKind:
				enumValueName, err := fs.GetString(flagName)
				if err != nil {
					return err
				}
				enumDesc := fd.Enum()
				ev := enumDesc.Values().ByName(protoreflect.Name(enumValueName))
				if ev == nil {
					return fmt.Errorf("unknown enum value %q for field %s", enumValueName, fd.FullName())
				}
				msg.Set(fd, protoreflect.ValueOfEnum(ev.Number()))

			default:
				// If you have other types (bytes, etc.), handle them here.
				// Or return an error if they're not supported.
				return fmt.Errorf("unsupported field kind %q for field %s", fd.Kind(), fd.FullName())
			}
		}
	}

	return nil
}

// FlagsFromMessage returns a pflag.FlagSet containing flags for all scalar
// and repeated scalar fields in the given Protobuf message descriptor,
// recursing into nested messages (unless map, repeated message, or otherwise
// ignored). This is symmetrical with `applyFlags`.
func FlagsFromMessage(
	msg proto.Message,
	ignorePaths []string, // Dot-based paths to ignore (e.g. "nested.id", "nested.nested.id")
) (*pflag.FlagSet, error) {

	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	visited := make(map[protoreflect.FullName]bool)

	// We start with no prefix for both kebab and dot
	if err := addFlags(
		fs,
		msg.ProtoReflect().Descriptor(),
		/* kebabPrefix = */ "",
		/* dotPrefix   = */ "",
		visited,
		ignorePaths,
	); err != nil {
		return nil, err
	}

	return fs, nil
}

// addFlags recurses over each field in msgDesc, generating flags
// for scalars or repeated scalars, and recursing for nested messages.
//
// kebabPrefix is used to form flag names (e.g. "nested-id").
// dotPrefix is used to check ignored paths (e.g. "nested.id").
func addFlags(
	fs *pflag.FlagSet,
	msgDesc protoreflect.MessageDescriptor,
	kebabPrefix string,
	dotPrefix string,
	visited map[protoreflect.FullName]bool,
	ignorePaths []string,
) error {
	// Avoid infinite recursion
	if visited[msgDesc.FullName()] {
		return nil
	}
	visited[msgDesc.FullName()] = true

	for i := 0; i < msgDesc.Fields().Len(); i++ {
		field := msgDesc.Fields().Get(i)

		// Build up the doc comment (usage text)
		comment := field.ParentFile().
			SourceLocations().
			ByDescriptor(field).
			LeadingComments

		// Original field name (Protobuf `text_name`)
		fieldName := string(field.TextName()) // e.g. "id" or "nested"

		// Convert to kebab for the actual flag name
		fieldKebab := toKebabCase(fieldName)

		// Build the next kebab prefix
		// Example: if kebabPrefix == "nested", then nextKebabPrefix => "nested-id"
		// If kebabPrefix is "", just use fieldKebab
		nextKebabPrefix := kebabPrefix
		if nextKebabPrefix != "" {
			nextKebabPrefix += "-"
		}
		nextKebabPrefix += fieldKebab

		// Build the next dot prefix
		// Example: if dotPrefix == "nested", then nextDotPrefix => "nested.id"
		// If dotPrefix is "", just use fieldName
		nextDotPrefix := dotPrefix
		if nextDotPrefix != "" {
			nextDotPrefix += "."
		}
		nextDotPrefix += fieldName

		// If user wants to ignore certain paths, check our dot-based path
		if isIgnored(nextDotPrefix, ignorePaths) {
			// Skip this field entirely
			continue
		}

		// If repeated
		if field.IsList() {
			// Only handle repeated scalars/enums. Repeated messages are more complex.
			if field.Kind() == protoreflect.MessageKind {
				return fmt.Errorf("repeated messages are not supported for field %q", nextKebabPrefix)
			}
			// For repeated scalars, define a single CSV string flag
			fs.String(nextKebabPrefix, "", comment)
			continue
		}

		// If this is a sub-message (non-repeated, non-map)
		if field.Kind() == protoreflect.MessageKind && !field.IsMap() {
			// Recurse
			if err := addFlags(
				fs,
				field.Message(),
				nextKebabPrefix, // kebab prefix
				nextDotPrefix,   // dot prefix
				visited,
				ignorePaths,
			); err != nil {
				return err
			}
			continue
		}

		// Otherwise handle single scalar fields:
		switch field.Kind() {
		case protoreflect.StringKind:
			fs.String(nextKebabPrefix, "", comment)
		case protoreflect.Int32Kind, protoreflect.Int64Kind:
			fs.Int(nextKebabPrefix, 0, comment)
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			fs.Uint(nextKebabPrefix, 0, comment)
		case protoreflect.BoolKind:
			fs.Bool(nextKebabPrefix, false, comment)
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			fs.Float64(nextKebabPrefix, 0, comment)
		case protoreflect.EnumKind:
			fs.String(nextKebabPrefix, "", comment)
		default:
			return fmt.Errorf("unsupported field kind %q for field %q",
				field.Kind(), field.FullName())
		}
	}
	return nil
}

// isIgnored returns true if the given field path is listed in ignorePaths.
//
// Here, fieldPath is the dot-based path (e.g. "nested.id"), and
// ignorePaths is a list of dot-based patterns to ignore.
func isIgnored(fieldPath string, ignorePaths []string) bool {
	for _, ignored := range ignorePaths {
		if fieldPath == ignored {
			return true
		}
	}
	return false
}

// toKebabCase replaces underscores with hyphens.
// Must match the same function used by applyFlags.
func toKebabCase(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}
