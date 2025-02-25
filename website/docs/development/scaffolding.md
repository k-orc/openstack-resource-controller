# Scaffolding

The first step typically in writing a new controller is to generate the scaffolding for it. The scaffolding covers functionality which is common to all controllers. Its purpose is not only to reduce the boilerplate required to write a new controller, but also to guarantee consistency of behaviour across APIs.

> While it is possible to write this code manually, any controller requiring this is potentially stretching assumptions made throughout the project. If this is required, consider if changes can be made such that it is not required, or if further design work is required in the scaffolding or generic controller code.

The first step is to add the new resource to `allResources` in `cmd/resource-generator/main.go` and run:

```shell
make generate-resources
```

This will generate 3 files for you:

* `api/<version>/zz_generated.<resource>-resource.go`
* `internal/controllers/<resource>/zz_generated.adapter.go`
* `internal/controllers/<resource>/zz_generated.controller.go`

> These files are generated using a very simplistic text templating system in `cmd/resource-generator`. If you are wondering why we didn't use generics, which are widely used throughout the rest of the code, it's because `controller-gen`, which generates CRDs from the API, doesn't yet support them. Consequently we generate manually what generics would have generated implicitly. This code may be rewritten in the future if `controller-gen` gains support for generics.

The code will not compile at this point, as the generated code will refer to code you have not yet written.