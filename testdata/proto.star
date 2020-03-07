# Tests of Starlark 'proto' extension.

load("assert.star", "assert")

s = struct(body = "hello")
assert.eq(s, s)
print(s)

m = proto("starlarkproto.test.Message", body="Hello, world!")
assert.eq(m, m)
print(m)
assert.eq(m.body, "Hello, world!")

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
