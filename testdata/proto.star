# Tests of Starlark 'proto' extension.

load("assert.star", "assert")

s = struct(body = "hello")
assert.eq(s, s)
print(s)

#m = proto("starlarkproto.test.Message", body="Hello, world!")
# Prefer load by import path for dynamic protobuf support
test = proto.load("github.com/afking/starlarkproto/testpb/star.proto")
#test = proto.package("starlarkproto.test")
print("loaded!!!!")
print(dir(test))
m = test.Message(body="Hello, world!")
assert.eq(m, m)
print(m)
assert.eq(m.body, "Hello, world!")

# Setting value asserts types
def set_field_invalid():
	m.body = 2
assert.fails(set_field_invalid, "proto: *")

# Enums can be assigned by String or Ints
assert.eq(m.type, 0)
m.type = "GREETING"
assert.eq(m.type, 1)
m.type = 0
assert.eq(m.type, 0)

# Lists are references
b = m.strings
b.append("hello")
assert.eq(m.strings[0], "hello")
print(m)

# Structs init on attr
m.nested.body = "nested"
assert.eq(m.nested.body, "nested")

# Message can be created from structs
m.nested = struct(body = "struct", type = "GREETING")
assert.eq(m.nested.body, "struct")
assert.eq(m.nested.type, 1)
print(m)

# Messages can be assigned None to delete
m.nested = None
assert.eq(m.nested, None)

# Maps init copy dicts
m.maps = {
	"hello": struct(body = "world!", type = "GREETING"),
}
print(m)

