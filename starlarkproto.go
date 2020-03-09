package starlarkproto

import (
	"fmt"
	"sort"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

//func NewModule(files *protoregistry.Files) *starlarkstruct.Module {
func NewModule(resolver protodesc.Resolver) *starlarkstruct.Module {
	p := &proto{resolver: resolver}

	return &starlarkstruct.Module{
		Name: "proto",
		Members: starlark.StringDict{
			"file":      starlark.NewBuiltin("proto.file", p.file),
			"new":       starlark.NewBuiltin("proto.new", p.new),
			"marshal":   starlark.NewBuiltin("proto.marshal", marshal),
			"unmarshal": starlark.NewBuiltin("proto.unmarshal", unmarshal),
		},
	}
}

type proto struct {
	//files *protoregistry.Files
	resolver protodesc.Resolver
}

func (p *proto) file(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackPositionalArgs("proto.package", args, kwargs, 1, &name); err != nil {
		return nil, err
	}

	fileDesc, err := p.resolver.FindFileByPath(name)
	if err != nil {
		return nil, err
	}
	return &Descriptor{desc: fileDesc}, nil
}

func (p *proto) new(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackPositionalArgs("proto.package", args, kwargs, 1, &name); err != nil {
		return nil, err
	}
	fullname := protoreflect.FullName(name)

	desc, err := p.resolver.FindDescriptorByName(fullname)
	if err != nil {
		return nil, err
	}
	return &Descriptor{desc: desc}, nil
}

func marshal(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	//starlark.UnpackPositionalArgs
	return nil, fmt.Errorf("TODO: marshal")
}

func unmarshal(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, fmt.Errorf("TODO: unmarshal")
}

type Descriptor struct {
	desc protoreflect.Descriptor
	//attrs map[string]
}

func (d *Descriptor) String() string        { return string(d.desc.Name()) }
func (d *Descriptor) Type() string          { return "proto.desc" }
func (d *Descriptor) Freeze()               {}
func (d *Descriptor) Truth() starlark.Bool  { return d.desc != nil }
func (d *Descriptor) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: proto.desc") }
func (d *Descriptor) Name() string          { return string(d.desc.Name()) } // TODO
func (d *Descriptor) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	switch v := d.desc.(type) {
	case protoreflect.FileDescriptor:
		return nil, fmt.Errorf("proto: file descriptor not callable")

	case protoreflect.EnumDescriptor:
		// TODO: unreachable?
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("unexpected kwargs")
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("unexpected number of args")
		}
		vals := v.Values()
		return NewEnum(vals, args[0])

	case protoreflect.MessageDescriptor:
		msg := dynamicpb.NewMessage(v)
		return NewMessage(msg, args, kwargs)

	default:
		return nil, fmt.Errorf("proto: desc missing call type %T", v)
	}
}

// TODO: can this just use the resolver?
func (d *Descriptor) Attr(name string) (starlark.Value, error) {
	// TODO: cache?
	switch v := d.desc.(type) {
	case protoreflect.FileDescriptor:
		fmt.Println("file")
		eds := v.Enums()
		for i := 0; i < eds.Len(); i++ {
			ed := eds.Get(i)
			if string(ed.Name()) == name {
				return &Descriptor{desc: ed}, nil
			}
		}

		mds := v.Messages()
		for i := 0; i < mds.Len(); i++ {
			md := mds.Get(i)
			if string(md.Name()) == name {
				return &Descriptor{desc: md}, nil
			}
		}
		return nil, nil

	case protoreflect.EnumDescriptor:
		evdesc := v.Values().ByName(protoreflect.Name(name))
		if evdesc == nil {
			return nil, fmt.Errorf("proto: enum not found")
		}
		return Enum{edesc: evdesc}, nil
		//vals.ByName
		//for i := 0; i < vals.Len(); i++ {
		//	evdesc := vals.Get(i)
		//	evdesc.
		//	if string(evdesc.Name()) == name {
		//		return &Descriptor{desc: evdesc}, nil
		//	}
		//}
		//return nil, nil

	case protoreflect.MessageDescriptor:
		eds := v.Enums()
		for i := 0; i < eds.Len(); i++ {
			ed := eds.Get(i)
			if string(ed.Name()) == name {
				return &Descriptor{desc: ed}, nil
			}
		}
		mds := v.Messages()
		for i := 0; i < mds.Len(); i++ {
			md := mds.Get(i)
			if string(md.Name()) == name {
				return &Descriptor{desc: md}, nil
			}
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("proto: desc missing attr type %T", v)
	}
}

func (d *Descriptor) AttrNames() []string {
	var names []string
	switch v := d.desc.(type) {
	case protoreflect.FileDescriptor:
		eds := v.Enums()
		for i := 0; i < eds.Len(); i++ {
			ed := eds.Get(i)
			names = append(names, string(ed.Name()))
		}

		mds := v.Messages()
		for i := 0; i < mds.Len(); i++ {
			md := mds.Get(i)
			names = append(names, string(md.Name()))
		}

	case protoreflect.EnumDescriptor:
		vals := v.Values()
		for i := 0; i < vals.Len(); i++ {
			evdesc := vals.Get(i)
			names = append(names, string(evdesc.Name()))
		}

	case protoreflect.MessageDescriptor:
		eds := v.Enums()
		for i := 0; i < eds.Len(); i++ {
			ed := eds.Get(i)
			names = append(names, string(ed.Name()))
		}

		mds := v.Messages()
		for i := 0; i < mds.Len(); i++ {
			md := mds.Get(i)
			names = append(names, string(md.Name()))
		}
	}
	sort.Strings(names)
	return names
}

// Message represents a proto.Message as a starlark.Value.
type Message struct {
	msg protoreflect.Message
}

// ProtoReflect implements proto.Message
func (m *Message) ProtoReflect() protoreflect.Message { return m.msg }

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
		evdesc := fd.Enum().Values().ByNumber(v)
		if evdesc == nil {
			evdesc = fd.DefaultEnumValue() // TODO: error?
		}
		return Enum{edesc: evdesc}
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
		// TODO: freeze
		if !v.IsValid() {
			return starlark.None
		}
		return &Message{
			msg: v,
		}
	case protoreflect.Map:
		// TODO: freeze
		if !v.IsValid() {
			return starlark.None
		}
		return &Map{
			m:     v,
			keyfd: fd.MapKey(),
			valfd: fd.MapValue(),
		}
	default:
		panic(fmt.Sprintf("unhandled proto type %s %T", v, v))
	}
}

func starToProto(v starlark.Value, fd protoreflect.FieldDescriptor, val *protoreflect.Value) error {
	switch kind := fd.Kind(); kind {
	case protoreflect.BoolKind:
		if b, ok := v.(starlark.Bool); ok {
			//return protoreflect.ValueOfBool(bool(b)), nil
			//return protoreflect.ValueOfBool(bool(b)), nil
			*val = protoreflect.ValueOfBool(bool(b))
			return nil
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Int64()
			//return protoreflect.ValueOfInt32(int32(v)), nil
			*val = protoreflect.ValueOfInt32(int32(v))
			return nil
		}

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Int64()
			//return protoreflect.ValueOfInt64(int64(v)), nil
			*val = protoreflect.ValueOfInt64(int64(v))
			return nil
		}

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Uint64()
			//return protoreflect.ValueOfUint32(uint32(v)), nil
			*val = protoreflect.ValueOfUint32(uint32(v))
			return nil

		}

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if x, ok := v.(starlark.Int); ok {
			v, _ := x.Uint64()
			//return protoreflect.ValueOfUint64(uint64(v)), nil
			*val = protoreflect.ValueOfUint64(uint64(v))
			return nil
		}

	case protoreflect.FloatKind:
		if x, ok := v.(starlark.Float); ok {
			*val = protoreflect.ValueOfFloat32(float32(x))
			return nil
		}

	case protoreflect.DoubleKind:
		if x, ok := v.(starlark.Float); ok {
			*val = protoreflect.ValueOfFloat64(float64(x))
			return nil
		}

	case protoreflect.StringKind:
		if x, ok := v.(starlark.String); ok {
			*val = protoreflect.ValueOfString(string(x))
			return nil
		}

	case protoreflect.BytesKind:
		if x, ok := v.(starlark.String); ok {
			*val = protoreflect.ValueOfBytes([]byte(x))
			return nil
		}

	case protoreflect.EnumKind:
		switch v := v.(type) {
		case starlark.String:
			enumVal := fd.Enum().Values().ByName(protoreflect.Name(string(v)))
			if enumVal == nil {
				return fmt.Errorf("proto: enum has no %s value", v)
			}
			*val = protoreflect.ValueOfEnum(enumVal.Number())
			return nil

		case starlark.Int:
			x, ok := v.Int64()
			if !ok {
				return fmt.Errorf("proto: enum has no %s value", v)
			}
			*val = protoreflect.ValueOfEnum(protoreflect.EnumNumber(int32(x)))
			return nil
		}

	case protoreflect.MessageKind:
		if fd.IsMap() {
			//mval := parent.NewField(fd)
			mm := val.Map()
			kfd := fd.MapKey()
			vfd := fd.MapValue()

			iter, ok := v.(starlark.IterableMapping)
			if !ok {
				break
			}

			items := iter.Items()
			for _, item := range items {
				mval := mm.NewValue()
				if err := starToProto(item[0], kfd, &mval); err != nil {
					return err
				}
				mkey := mval.MapKey()

				vval := mm.Mutable(mkey)
				if err := starToProto(item[1], vfd, &vval); err != nil {
					return err
				}

				mm.Set(mkey, vval)
			}
			return nil
		}

		switch v := v.(type) {
		case *starlarkstruct.Struct:
			//msg := dynamicpb.NewMessage(fd.Message())
			msg := val.Message()
			m := &Message{msg: msg} // wrap for set

			names := v.AttrNames()
			for _, name := range names {
				val, err := v.Attr(name)
				if err != nil {
					return err
				}
				if err := m.SetField(name, val); err != nil {
					return err
				}
			}
			return nil
			//return protoreflect.ValueOfMessage(msg), nil

		default:
			fmt.Println("undefined message kind")
		}

	default:
		return fmt.Errorf("proto: unsupported kind %q", kind)
	}

	return fmt.Errorf("proto: unknown type conversion %s", v.Type())
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
//func FromKeywords(msg protoreflect.Message, kwargs []starlark.Tuple) (*Message, error) {
func NewMessage(msg protoreflect.Message, args starlark.Tuple, kwargs []starlark.Tuple) (*Message, error) {
	hasArgs := len(args) > 0
	hasKwargs := len(kwargs) > 0

	if hasArgs && len(args) > 1 {
		return nil, fmt.Errorf("unexpected number of args")
	}

	if hasArgs && hasKwargs {
		return nil, fmt.Errorf("unxpected args and kwargs")
	}

	// TODO: clear?
	m := &Message{
		msg: msg,
	}
	if hasArgs {
		arg, ok := args[0].(starlark.HasAttrs)
		if !ok {
			return nil, fmt.Errorf("arg has no attributes")
		}

		names := arg.AttrNames()
		for _, name := range names {
			v, err := arg.Attr(name)
			if err != nil {
				return nil, err
			}
			if err := m.SetField(name, v); err != nil {
				return nil, err
			}
		}
		return m, nil
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
	return 0, fmt.Errorf("unhashable type: proto.message")
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

	if val == starlark.None {
		m.msg.Clear(fd)
		return nil
	}

	v := m.msg.NewField(fd)
	if err := starToProto(val, fd, &v); err != nil {
		return err
	}

	m.msg.Set(fd, v)
	return nil
}

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
	return 0, fmt.Errorf("unhashable type: proto.list")
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

	val := l.list.NewElement()
	if err := starToProto(v, l.fd, &val); err != nil {
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
	val := l.list.NewElement()
	if err := starToProto(v, l.fd, &val); err != nil {
		return err
	}
	l.list.Append(val)
	return nil
}

// Enum is the type of a protobuf enum.
type Enum struct {
	edesc protoreflect.EnumValueDescriptor
	//i   int32
	//str string // string representation
	//typ string
}

func NewEnum(enum protoreflect.EnumValueDescriptors, arg starlark.Value) (Enum, error) {
	switch v := arg.(type) {
	case starlark.String:
		edesc := enum.ByName(protoreflect.Name(v))
		if edesc == nil {
			return Enum{}, fmt.Errorf("proto: enum not found")
		}
		return Enum{edesc: edesc}, nil

	case starlark.Int:
		n, _ := v.Int64() // TODO: checks?
		edesc := enum.ByNumber(protoreflect.EnumNumber(n))
		return Enum{edesc: edesc}, nil

	case Enum:
		return Enum{edesc: v.edesc}, nil

	default:
		return Enum{}, fmt.Errorf("unsupported type %s", arg.Type())
	}
}

func (e Enum) String() string        { return string(e.edesc.Name()) }
func (e Enum) Type() string          { return "proto.enum" }
func (e Enum) Freeze()               {} // immutable
func (e Enum) Truth() starlark.Bool  { return e.edesc.Number() > 0 }
func (e Enum) Hash() (uint32, error) { return uint32(e.edesc.Number()), nil }
func (x Enum) CompareSameType(op syntax.Token, y_ starlark.Value, depth int) (bool, error) {
	y := y_.(Enum)
	if x.edesc.Parent().FullName() != y.edesc.Parent().FullName() {
		return false, fmt.Errorf("proto enum: type error %s and %s", x.edesc.FullName(), y.edesc.FullName())
	}
	i, j := x.edesc.Number(), y.edesc.Number()
	switch op {
	case syntax.EQL:
		return i == j, nil
	case syntax.NEQ:
		return i != j, nil
	case syntax.LE:
		return i <= j, nil
	case syntax.LT:
		return i < j, nil
	case syntax.GE:
		return i >= j, nil
	case syntax.GT:
		return i > 0, nil
	}
	panic(op)
}

type Map struct {
	m     protoreflect.Map
	keyfd protoreflect.FieldDescriptor
	valfd protoreflect.FieldDescriptor

	frozen    bool
	itercount uint32
}

func (m *Map) Clear() error {
	m.m.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
		m.m.Clear(key)
		return true
	})
	return nil
}
func (m *Map) Delete(k starlark.Value) (v starlark.Value, found bool, err error) {
	var keyval protoreflect.Value
	if err := starToProto(k, m.keyfd, &keyval); err != nil {
		return nil, false, err
	}
	key := keyval.MapKey()
	val := m.m.Get(key)
	if !val.IsValid() {
		return starlark.None, false, nil
	}
	m.m.Clear(key)
	return protoToStar(val, m.valfd), true, nil
}
func (m *Map) Get(k starlark.Value) (v starlark.Value, found bool, err error) {
	var keyval protoreflect.Value
	if err := starToProto(k, m.keyfd, &keyval); err != nil {
		return nil, false, err
	}
	key := keyval.MapKey()
	val := m.m.Get(key)
	if !val.IsValid() {
		return starlark.None, false, nil
	}
	return protoToStar(val, m.valfd), true, nil
}

type byTuple []starlark.Tuple

func (a byTuple) Len() int      { return len(a) }
func (a byTuple) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byTuple) Less(i, j int) bool {
	c := a[i][0].(starlark.Comparable)
	ok, err := c.CompareSameType(syntax.LT, a[j][0], 1)
	if err != nil {
		panic(err)
	}
	return ok
}

func (m *Map) Items() []starlark.Tuple {
	v := make([]starlark.Tuple, 0, m.Len())
	m.m.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
		v = append(v, starlark.Tuple{
			protoToStar(key.Value(), m.keyfd),
			protoToStar(val, m.valfd),
		})
		return true
	})
	sort.Sort(byTuple(v))
	return v
}

type byValue []starlark.Value

func (a byValue) Len() int      { return len(a) }
func (a byValue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byValue) Less(i, j int) bool {
	c := a[i].(starlark.Comparable)
	ok, err := c.CompareSameType(syntax.LT, a[j], 1)
	if err != nil {
		panic(err)
	}
	return ok
}

func (m *Map) Keys() []starlark.Value {
	v := make([]starlark.Value, 0, m.Len())
	m.m.Range(func(key protoreflect.MapKey, _ protoreflect.Value) bool {
		v = append(v, protoToStar(key.Value(), m.keyfd))
		return true
	})
	sort.Sort(byValue(v))
	return v
}
func (m *Map) Len() int {
	return m.m.Len()
}

type keyIterator struct {
	m    *Map
	keys []starlark.Value
	i    int
}

func (ki *keyIterator) Next(k *starlark.Value) bool {
	if ki.i < len(ki.keys) {
		*k = ki.keys[ki.i]
		ki.i++
		return true
	}
	return false
}

func (ki *keyIterator) Done() {
	if !ki.m.frozen {
		ki.m.itercount--
	}
}

func (m *Map) Iterate() starlark.Iterator {
	if !m.frozen {
		m.itercount--
	}
	return &keyIterator{m: m, keys: m.Keys()}
}
func (m *Map) SetKey(k, v starlark.Value) error {
	var keyval protoreflect.Value
	if err := starToProto(k, m.keyfd, &keyval); err != nil {
		return err
	}
	key := keyval.MapKey()

	val := m.m.NewValue()
	if err := starToProto(k, m.valfd, &val); err != nil {
		return err
	}
	m.m.Set(key, val)
	return nil
}
func (m *Map) String() string {
	buf := new(strings.Builder)
	buf.WriteByte('{')
	for i, item := range m.Items() {
		if i > 0 {
			buf.WriteString(", ")
		}
		k, v := item[0], item[1]

		buf.WriteString(k.String())
		buf.WriteString(": ")
		buf.WriteString(v.String())
	}
	buf.WriteByte('}')
	return buf.String()
}
func (m *Map) Type() string {
	return "TODO"
}
func (m *Map) Freeze() {
	if !m.frozen {
		m.frozen = true
		// TODO: keys are immutable
		//       values need checking...
	}
}
func (m *Map) Truth() starlark.Bool  { return m.Len() > 0 }
func (m *Map) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: map") }
func (m *Map) Attr(name string) (starlark.Value, error) {
	return nil, nil
}
func (m *Map) AttrNames() []string {
	return nil
}
