package starlarkproto

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/starlarktest"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/emcfarlane/starlarkproto/testpb" // import side effect
)

func load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	if module == "assert.star" {
		return starlarktest.LoadAssertModule()
	}
	return nil, fmt.Errorf("unknown module %s", module)
}

func TestExecFile(t *testing.T) {
	protoregistry.GlobalFiles.RangeFiles(func(p protoreflect.FileDescriptor) bool {
		fmt.Println("file!", p.FullName())
		fmt.Println("     ", p.Path())
		fmt.Println("     ", p.Name())
		fmt.Println("     ", p.Package())
		return true
	})

	resolve.AllowGlobalReassign = true

	thread := &starlark.Thread{Load: load}
	starlarktest.SetReporter(thread, t)
	globals := starlark.StringDict{
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
		"proto":  NewModule(protoregistry.GlobalFiles),
	}

	files, err := filepath.Glob("testdata/*.star")
	if err != nil {
		t.Fatal(err)
	}

	for _, filename := range files {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}

		_, err = starlark.ExecFile(thread, filename, src, globals)
		switch err := err.(type) {
		case *starlark.EvalError:
			var found bool
			for i := range err.CallStack {
				posn := err.CallStack.At(i).Pos
				if posn.Filename() == filename {
					linenum := int(posn.Line)
					msg := err.Error()

					t.Errorf("\n%s:%d: unexpected error: %v", filename, linenum, msg)
					found = true
					break
				}
			}
			if !found {
				t.Error(err.Backtrace())
			}
		case nil:
			// success
		default:
			t.Errorf("\n%s", err)
		}

	}
}
