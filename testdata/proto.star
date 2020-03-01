# Tests of Starlark 'proto' extension.

load("assert.star", "assert")

s = struct(body = "hello")
assert.eq(s, s)
print(s)

m = proto("starlarkproto.test.Message", body="Hello, world!")
assert.eq(m, m)
print(m)
assert.eq(m.body, "Hello, world!")
assert.eq(m.type, 0)
#assert.eq(m.type, "UNKNOWN")
m.type = "GREETING"
assert.eq(m.type, 1)
