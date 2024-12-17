This kustomize module is for testing a deployment via `config/default` using a
locally built image.

The deployment in config/manager specifies an image of `controller:latest`. The
problem with this is that k8s implicitly applies a pull policy of `Always` to
an image with the latest tag. This means that even if we push the image to our
kind node the cluster will still try to pull the image, which will fail. To
circumvent this we rename the image to `controller:devel`.

To use this module, from the root directory:

```
$ docker build -t controller:devel .
$ kind create cluster
$ kind load docker-image controller:devel
$ kustomize build hack/local-deploy | kubectl apply -f - --server-side
```
