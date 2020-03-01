# starlarkproto

[![GoDev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/mod/github.com/afking/starlarkproto)

Supports proto messages in starlark!

```python
m = proto("starlarkproto.test.Message", body="Hello, world!")
print(m) # Message(body = Hello, world!, type = 0)
m.type = "GREETING"
print(m) # Message(body = Hello, world!, type = 1)
```
