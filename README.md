# starlarkproto

[![GoDev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/mod/github.com/afking/starlarkproto)

Supports proto messages in starlark!

```python
test = proto.file("github.com/afking/starlarkproto/testpb/star.proto")
m = test.Message(body="Hello, world!")
print(m) # Message(body = Hello, world!, type = 0)
m.type = "GREETING"
print(m) # Message(body = Hello, world!, type = 1)

greeting = test.Message.Type.GREETING
print(greeting) # GREETING

data = proto.marshal(m)
resp = proto.unmarshal(m)
```
