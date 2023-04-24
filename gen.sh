protoc -I /usr/local/include/ -I. --go_out=module=github.com/emcfarlane/starlarkproto:. testpb/*.proto
