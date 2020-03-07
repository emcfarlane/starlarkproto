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
//  *List          │ List<Kind>
//  Tuple          │ n/a
//  *Dict          │ Map<Kind><Kind>
//  *Set           │ n/a
//
func protoToStar(v protoreflect.Value, fd protoreflect.FieldDescriptor) starlark.Value {
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
		//return Enum{i: int32(v), typ: fd.Kind().String()}
	case protoreflect.List:
		// TODO: freeze
		if !v.IsValid() {
			return starlark.None
		}
		return &List{
			list: v,
			fd:   fd,
		}
	case protoreflect.Message:
		if !v.IsValid() {
			return starlark.None
		}
		fmt.Println("---\n", v.IsValid(), "\n", fd)
		// TODO: freeze
		return &Message{
			msg: v,
		}
		//return starlark.None
	case protoreflect.Map:
		return starlark.None
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

func (m *Message) get(fd protoreflect.FieldDescriptor) protoreflect.Value {
	return m.msg.Get(fd)
}

func (m *Message) mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	if fd.IsMap() || fd.IsList() || fd.Kind() == protoreflect.MessageKind {
		return m.msg.Mutable(fd)
	}
	return m.msg.Get(fd)
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
		val := m.get(fd)

		// Method here should always be get?

		v := protoToStar(val, fd)
		buf.WriteString(v.String())
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
		val := m.get(fd)

		namehash, _ := starlark.String(fd.Name()).Hash()
		x = x ^ 3*namehash

		v := protoToStar(val, fd)
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

	pv := m.mutable(fd) // Attr can mutate
	return protoToStar(pv, fd), nil
}

func (x *Message) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	return nil, nil // unhandled
}

// AttrNames returns a new sorted list of the message fields.
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

//type method func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

var (
	listMethods = map[string]*starlark.Builtin{
		"append": starlark.NewBuiltin("append", list_append),
		//"clear":  list_clear,
		//"extend": list_extend,
		//"index":  list_index,
		//"insert": list_insert,
		//"pop":    list_pop,
		//"remove": list_remove,
	}
)

func bindAttr(recv starlark.Value, name string, methods map[string]*starlark.Builtin) (starlark.Value, error) {
	b := methods[name]
	if b == nil {
		return nil, nil // no such method
	}
	return b.BindReceiver(recv), nil
}

// https://github.com/google/starlark-go/blob/6677ee5c7211380ec7e6a1b50dc45287e40ca9e1/starlark/library.go#L153
func builtinAttrNames(methods map[string]*starlark.Builtin) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// List represents a repeated field as a starlark.List.
type List struct {
	//msg protoreflect.Message
	//fd  protoreflect.FieldDescriptor
	list protoreflect.List
	fd   protoreflect.FieldDescriptor

	frozen    bool
	itercount uint32
}

func (l *List) Attr(name string) (starlark.Value, error) { return bindAttr(l, name, listMethods) }
func (l *List) AttrNames() []string                      { return builtinAttrNames(listMethods) }

func (l *List) String() string {
	buf := new(strings.Builder)
	buf.WriteByte('[')
	for i := 0; i < l.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(l.Index(i).String())
	}
	buf.WriteByte(']')
	return buf.String()
}

func (l *List) Freeze() {
	if !l.frozen {
		l.frozen = true
		for i := 0; i < l.Len(); i++ {
			l.Index(i).Freeze()
		}
	}
}

func (l *List) Hash() (uint32, error) {
	// TODO: make hashable???
	return 0, fmt.Errorf("unhashable type: list")
}

func (l *List) checkMutable(verb string) error {
	if l.frozen {
		return fmt.Errorf("cannot %s frozen list", verb)
	}
	if l.itercount > 0 {
		return fmt.Errorf("cannot %s list during iteration", verb)
	}
	return nil
}

func (l *List) Index(i int) starlark.Value {
	return protoToStar(l.list.Get(i), l.fd)
}

type listIterator struct {
	l *List
	i int
}

func (it *listIterator) Next(p *starlark.Value) bool {
	if it.i < it.l.Len() {
		val := it.l.list.Get(it.i)
		*p = protoToStar(val, it.l.fd)
		it.i++
		return true
	}
	return false
}

func (it *listIterator) Done() {
	if !it.l.frozen {
		it.l.itercount--
	}
}

func (l *List) Iterate() starlark.Iterator {
	if !l.frozen {
		l.itercount++
	}
	return &listIterator{l: l}
}

func (l *List) Type() string         { return l.fd.Kind().String() }
func (l *List) Len() int             { return l.list.Len() }
func (l *List) Truth() starlark.Bool { return l.Len() > 0 }

func (l *List) SetIndex(i int, v starlark.Value) error {
	if err := l.checkMutable("assign to element of"); err != nil {
		return err
	}

	val, err := starToProto(v, l.fd)
	if err != nil {
		return err
	}

	l.list.Set(i, val)
	return nil
}

func list_append(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var object starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &object); err != nil {
		return nil, err
	}
	recv := b.Receiver().(*List)
	if err := recv.Append(object); err != nil {
		return nil, err
	}
	return starlark.None, nil
}

func (l *List) Append(v starlark.Value) error {
	if err := l.checkMutable("append to"); err != nil {
		return err
	}
	pv, err := starToProto(v, l.fd)
	if err != nil {
		return err
	}
	l.list.Append(pv)
	return nil
}

//// Enum is the type of a protobuf enum.
//type Enum struct {
//	i   int32
//	str string // string representation
//	typ string
//}
//
//func (e Enum) String() string {
//	if e.str != "" {
//		return e.str
//	}
//	return strconv.Itoa(int(e.i))
//}
//func (e Enum) Type() string          { return e.typ }
//func (e Enum) Freeze()               {} // immutable
//func (e Enum) Truth() starlark.Bool  { return e.i > 0 }
//func (e Enum) Hash() (uint32, error) { return uint32(e.i), nil }
//func (x Enum) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
//	return false, nil // TODO:...
//}
