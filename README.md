# starlarkproto

[![GoDev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/github.com/afking/starlarkproto?tab=doc)

Supports protobuffers in starlark!

```python
test = proto.file("github.com/afking/starlarkproto/testpb/star.proto")
m = test.Message(body="Hello, world!")
print(m)  # Message(body = Hello, world!, type = UNKNOWN, ...)
m.type = "GREETING" # Enums can be assigned by String, Int or proto.Enum
print(m)  # Message(body = Hello, world!, type = GREETING, ...)

greeting = test.Message.Type.GREETING
print(greeting)  # GREETING

data = proto.marshal(m) # Byte encoded string
m2 = test.Message()
proto.unmarshal(data, m2) # Unmarshal back to message
```
