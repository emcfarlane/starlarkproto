# Tests of Starlark 'proto' extension.

load("assert.star", "assert")

s = struct(body = "hello")
assert.eq(s, s)
print(s)

# Prefer load by import path for dynamic protobuf support
#m = proto("starlarkproto.test.Message", body="Hello, world!")
#test = proto.package("starlarkproto.test")
#test = 1
test = proto.file("github.com/afking/starlarkproto/testpb/star.proto")
#print("loaded!!!!")
#print(dir(test))
m = test.Message(body="Hello, world!")
assert.eq(m, m)
assert.eq(dir(m), ["body", "maps", "nested", "one_number", "one_string", "oneofs", "strings", "type"])
print(m)
assert.eq(m.body, "Hello, world!")

# Setting value asserts types
def set_field_invalid():
	m.body = 2
assert.fails(set_field_invalid, "proto: *")


# Enums
enum = proto.new("starlarkproto.test.Enum")
enum_a = enum(0)
enum_a_alt = enum("ENUM_A")
assert.eq(enum_a, enum_a_alt)

enum_file = test.Enum
enum_b = enum_file(1)
enum_b_alt = enum_file("ENUM_B")
assert.eq(enum_b, enum_b_alt)
assert.ne(enum_a, enum_b)
#print("ENUMS", enum_a, enum_b)

# Nested Enums
message_unknown = test.Message.Type.UNKNOWN
message_greeting = test.Message.Type.GREETING
assert.ne(message_unknown, message_greeting)

# Enums can be assigned by String or Ints
assert.eq(m.type, message_unknown)
m.type = "GREETING"
assert.eq(m.type, message_greeting)
m.type = 0
assert.eq(m.type, message_unknown)
m.type = message_greeting
assert.eq(m.type, message_greeting)

# Lists are references
b = m.strings
b.append("hello")
assert.eq(m.strings[0], "hello")
print(m)

# Structs init on attr
#m.nested.body = "nested"
#assert.eq(m.nested.body, "nested")

# Message can be created from structs
m.nested = struct(body = "struct", type = "GREETING")
assert.eq(m.nested.body, "struct")
assert.eq(m.nested.type, message_greeting)
print(m)

# Messages can be assigned None to delete
m.nested = None
#assert.eq(m.nested, test.Message(None))  # None creates typed nil
assert.true(not m.nested, msg="Nil RO type is falsy")  # 

# Maps shallow copy Dicts on set
m.maps = {
	"hello": struct(body = "world!", type = "GREETING"),
}
print(m)

# Oneofs
m.one_string = "one dream"
assert.eq(m.one_string, "one dream")
assert.eq(m.one_number, 0)
assert.eq(m.oneofs, "one dream")
m.one_number = 1
assert.eq(m.one_string, "")
assert.eq(m.one_number, 1)
assert.eq(m.oneofs, 1)
print(m)

# Marshal/Unmarshal
data = proto.marshal(m)
m2 = test.Message()
proto.unmarshal(data, m2)
assert.eq(m, m2)

