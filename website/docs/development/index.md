# Developing controllers

This documentation goes over the different steps needed to write a new controller from scratch.

## Using the controller scaffolding generator

ORC comes with a controller scaffolding tool that does most of the heavy work when writing a new controller.
It asks a series of questions, then generates stubs for the API, controller implementation, tests and client code.

To run the scaffolding tool:
```bash
$ go run ./cmd/scaffold-controller
```

The tool can be run either interactively, if you don't pass it any option, or non-interactively if you pass it the `-interactive=false` flag and provide all of the required flags.

Once the tool returned successfully, there are still a few manual steps to integrate your new controller in ORC:

* Add the new resource to `cmd/resource-generator/main.go` and run `make generate`.
* Add the new client to `internal/scope/scope.go`, `internal/scope/provider.go`, and `internal/scope/mock.go`.
* Add the controller to `cmd/manager/main.go`.
* Implement all the TODOs. Search the code for `TODO(scaffolding)` to find them all.
* Iterate on your controller and get all the tests running.
* Run `make generate-bundle` to generate the OLM bundle manifests.
* Update the `README.md` file to list the new controller.

The following pages go into the details for writing controllers.
