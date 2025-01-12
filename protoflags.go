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
	msg protoreflect.MessageDescriptor,
	ignorePaths []string, // List of top-level or prefixed field names to ignore
) (*pflag.FlagSet, error) {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)

	// Track message descriptors we've visited to avoid infinite recursion.
	visited := make(map[protoreflect.FullName]bool)

	// Start recursion
	if err := addFlags(fs, msg, "", visited, ignorePaths); err != nil {
		return nil, err
	}
	return fs, nil
}

// addFlags recurses over each field in msg, generating flags
// for scalars or repeated scalars, and recursing for nested messages.
func addFlags(
	fs *pflag.FlagSet,
	msg protoreflect.MessageDescriptor,
	prefix string,
	visited map[protoreflect.FullName]bool,
	ignorePaths []string,
) error {
	// Avoid infinite recursion
	if visited[msg.FullName()] {
		return nil
	}
	visited[msg.FullName()] = true

	for i := 0; i < msg.Fields().Len(); i++ {
		field := msg.Fields().Get(i)
		comment := field.ParentFile().
			SourceLocations().
			ByDescriptor(field).
			LeadingComments

		// Convert field’s TextName to kebab-case
		fieldKebab := toKebabCase(field.TextName())
		flagName := prefix + fieldKebab

		// If user wants to ignore certain fields or paths, skip them
		if isIgnored(flagName, ignorePaths) {
			continue
		}

		// Check if repeated
		if field.IsList() {
			// Only handle repeated scalars/enums. Repeated messages are more complex.
			if field.Kind() == protoreflect.MessageKind {
				// We skip repeated messages or return an error (your choice).
				// Return an error for clarity:
				return fmt.Errorf("repeated messages are not supported for field %q", flagName)
			}
			// For repeated scalars, define a single CSV string flag
			// e.g. "1,2,3" for repeated int.
			fs.String(flagName, "", comment)

			// TODO?
			// fs.StringSlice(flagName, []string{}, comment)
			continue
		}

		// Handle nested single messages
		if field.Kind() == protoreflect.MessageKind && !field.IsMap() {
			// Recursively add flags for the sub-message
			if err := addFlags(fs, field.Message(), flagName+"-", visited, ignorePaths); err != nil {
				return err
			}
			continue
		}

		// Handle single scalar fields
		switch field.Kind() {
		case protoreflect.StringKind:
			fs.String(flagName, "", comment)
		case protoreflect.Int32Kind, protoreflect.Int64Kind:
			fs.Int(flagName, 0, comment)
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			fs.Uint(flagName, 0, comment)
		case protoreflect.BoolKind:
			fs.Bool(flagName, false, comment)
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			fs.Float64(flagName, 0, comment)
		case protoreflect.EnumKind:
			fs.String(flagName, "", comment)
		default:
			// If you have other types (bytes, etc.), handle them here or return error
			return fmt.Errorf("unsupported field kind %q for field %q",
				field.Kind(), field.FullName())
		}
	}
	return nil
}

// isIgnored returns true if the given field name is listed in ignorePaths.
func isIgnored(fieldName string, ignorePaths []string) bool {
	for _, ignored := range ignorePaths {
		if fieldName == ignored {
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
