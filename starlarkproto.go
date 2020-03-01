package starlarkproto

import (
	"fmt"
	"sort"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Message represents a proto.Message as a starlark.Value.
type Message struct {
	msg protoreflect.Message
}

// Make creates the implementation of a builtin function that instantiates a
// mutable message based on a protobuf Message descriptor.
//
// An application can add 'proto' to the Starlark envrionment like so:
//
// 	globals := starlark.StringDict{
// 		"proto": starlark.NewBuiltin("proto", starlarkproto.Make(
// 			protoregistry.GlobalFiles,
// 		)),
// 	}
//
func Make(resolver protodesc.Resolver) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {

	return func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("proto: unexpected positional arguments")
		}

		name, ok := starlark.AsString(args[0])
		if !ok {
			return nil, fmt.Errorf("proto: expected string type %v", args[0])
		}

		desc, err := resolver.FindDescriptorByName(protoreflect.FullName(name))
		if err != nil {
			return nil, err
		}

		md, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("proto: invalid descriptor type %T", desc)
		}

		msg := dynamicpb.NewMessage(md)

		return FromKeywords(msg, kwargs)
	}
}

// Type conversions rules:
//
//  ═══════════════╤════════════════════════════════════
//  Starlark type  │ Protobuf Type
//  ═══════════════╪════════════════════════════════════
//  NoneType       │ MessageKind, GroupKind
//  Bool           │ BoolKind
//  Int            │ Int32Kind, Sint32Kind, Sfixed32Kind,
//                 │ Int64Kind, Sint64Kind, Sfixed64Kind,
//                 │ Uint32Kind, Fixed32Kind,
//                 │ Uint64Kind, Fixed64Kind
//  Float          │ FloatKind, DoubleKind
//  String         │ StringKind, BytesKind
//  *List          │ []<Kind>
//  Tuple          │ n/a
//  *Dict          │ Map<Kind><Kind>
//  *Set           │ n/a
//
func protoToStar(v protoreflect.Value) starlark.Value {
	switch v := v.Interface().(type) {
	case nil:
		return starlark.None
	case bool:
		return starlark.Bool(v)
	case int32:
		return starlark.MakeInt(int(v))
	case int64:
		return starlark.MakeInt(int(v))
	case uint32:
		return starlark.MakeInt(int(v))
	case uint64:
		return starlark.MakeInt(int(v))
	case float32:
		return starlark.Float(float64(v))
	case float64:
		return starlark.Float(v)
	case string:
		return starlark.String(v)
	case []byte:
		return starlark.String(v)
	case protoreflect.EnumNumber:
		return starlark.MakeInt(int(v)) // TODO: strings?
	default:
		panic(fmt.Sprintf("unhandled proto type %s %T", v, v))
	}
}
func starToProto(v starlark.Value, fd protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	switch kind := fd.Kind(); kind {
	case protoreflect.BoolKind:
		if b, ok := v.(starlark.Bool); ok {
			return protoreflect.ValueOfBool(bool(b)), nil
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Int64()
			return protoreflect.ValueOfInt32(int32(v)), nil
		}

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Int64()
			return protoreflect.ValueOfInt64(int64(v)), nil
		}

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Uint64()
			return protoreflect.ValueOfUint32(uint32(v)), nil
		}

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Uint64()
			return protoreflect.ValueOfUint64(uint64(v)), nil
		}

	case protoreflect.FloatKind:
		if x, ok := v.(starlark.Float); ok {
			return protoreflect.ValueOfFloat32(float32(x)), nil
		}

	case protoreflect.DoubleKind:
		if x, ok := v.(starlark.Float); ok {
			return protoreflect.ValueOfFloat64(float64(x)), nil
		}

	case protoreflect.StringKind:
		if x, ok := v.(starlark.String); ok {
			return protoreflect.ValueOfString(string(x)), nil
		}

	case protoreflect.BytesKind:
		if x, ok := v.(starlark.String); ok {
			return protoreflect.ValueOfBytes([]byte(x)), nil
		}

	case protoreflect.EnumKind:
		switch v := v.(type) {
		case starlark.String:
			enumVal := fd.Enum().Values().ByName(protoreflect.Name(string(v)))
			if enumVal == nil {
				return protoreflect.Value{}, fmt.Errorf("enum has no %s value", v)
			}
			return protoreflect.ValueOfEnum(enumVal.Number()), nil

		case starlark.Int:
			x, ok := v.Int64()
			if !ok {
				return protoreflect.Value{}, fmt.Errorf("enum has no %s value", v)
			}
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(int32(x))), nil
		}

	default:
		return protoreflect.Value{}, fmt.Errorf("proto: unsupported kind %s", kind)
	}

	return protoreflect.Value{}, fmt.Errorf("proto: unknown type conversion %s", v.Type())
}

//func FromKeywords(md protoreflect.MessageDescriptor, kwargs []starlark.Tuple) (*Message, error) {
func FromKeywords(msg protoreflect.Message, kwargs []starlark.Tuple) (*Message, error) {
	// TODO: clear?
	m := &Message{
		msg: msg,
	}

	for _, kwarg := range kwargs {
		k := string(kwarg[0].(starlark.String))
		v := kwarg[1]

		if err := m.SetField(k, v); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *Message) String() string {
	desc := m.msg.Descriptor()
	buf := new(strings.Builder)
	buf.WriteString(string(desc.Name()))

	buf.WriteByte('(')
	fds := desc.Fields()
	for i := 0; i < fds.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		fd := fds.Get(i)
		buf.WriteString(string(fd.Name()))
		buf.WriteString(" = ")
		val := m.msg.Get(fd)
		buf.WriteString(val.String()) // TODO: convert to starlark first...
	}
	buf.WriteByte(')')
	return buf.String()
}

func (m *Message) Type() string         { return "proto" }
func (m *Message) Truth() starlark.Bool { return true }
func (m *Message) Hash() (uint32, error) {
	// Same algorithm as Tuple.hash, but with different primes.
	var x, h uint32 = 8731, 9839

	desc := m.msg.Descriptor()
	fds := desc.Fields()
	for i := 0; i < fds.Len(); i++ {
		fd := fds.Get(i)
		val := m.msg.Get(fd)

		namehash, _ := starlark.String(fd.Name()).Hash()
		x = x ^ 3*namehash

		v := protoToStar(val)
		y, err := v.Hash()
		if err != nil {
			return 0, err
		}
		x = x ^ y*h
		h += 7349
	}
	return x, nil
}
func (m *Message) Freeze() {
	// TODO: freezing...
}

// Attr returns the value of the specified field.
func (m *Message) Attr(name string) (starlark.Value, error) {
	fd, err := m.fieldDesc(name)
	if err != nil {
		return nil, err
	}

	v := m.msg.Get(fd)
	return protoToStar(v), nil
}

func (x *Message) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	return nil, nil // unhandled
}

// AttrNames returns a new sorted list of the struct fields.
func (m *Message) AttrNames() []string {
	desc := m.msg.Descriptor()
	fds := desc.Fields()
	names := make([]string, fds.Len())
	for i := range names {
		fd := fds.Get(i)
		names[i] = string(fd.Name())

	}
	sort.Strings(names)
	return names
}

func (m *Message) fieldDesc(name string) (protoreflect.FieldDescriptor, error) {
	desc := m.msg.Descriptor()
	fd := desc.Fields().ByName(protoreflect.Name(name))
	if fd == nil {
		return nil, starlark.NoSuchAttrError(
			fmt.Sprintf("%s has no .%s attribute", desc.Name(), name))
	}
	return fd, nil
}

func (m *Message) SetField(name string, val starlark.Value) error {
	fd, err := m.fieldDesc(name)
	if err != nil {
		return err
	}

	v, err := starToProto(val, fd)
	if err != nil {
		return err
	}

	m.msg.Set(fd, v)
	return nil
}
