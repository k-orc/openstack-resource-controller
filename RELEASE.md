# Release procedure

Generate the new `install.yaml` manifest, and tag the release:
```bash
export VERSION=vX.Y.Z
make build-installer IMG=quay.io/orc/openstack-resource-controller:$VERSION
git add dist
git commit -m "Release $VERSION"
git tag -s -a $VERSION -m $VERSION
git push origin
git push origin tag $VERSION
```

Pushing the tag will trigger the [release
workflow](https://github.com/k-orc/openstack-resource-controller/actions/workflows/release.yaml)
which builds the container image and publishes the Helm charts. When it is
done, check that you can successfully pull the artifacts:
```bash
podman pull quay.io/orc/openstack-resource-controller:$VERSION
helm pull oci://quay.io/orc/helm/orc --version ${VERSION#v}
helm pull oci://quay.io/orc/helm/orc-crds --version ${VERSION#v}
```

Finally, create the release in github. We must attach the generated `install.yaml` to the release artifacts:
```bash
gh release create -d --generate-notes --verify-tag -t "Release $VERSION" $VERSION
gh release upload $VERSION dist/install.yaml
```

Edit the release draft from github, and publish.

## Manually publishing to OperatorHub 

After the release images have been pushed to Quay (see above steps), submit the new version to OperatorHub by opening a PR to the [community-operators](https://github.com/k8s-operatorhub/community-operators) repository.

### Steps

From the ORC repository, check out the release tag and generate the bundle:
```bash
export VERSION=vX.Y.Z
git checkout $VERSION
rm -rf bundle
make generate-bundle
```

This produces a `bundle/` directory containing `manifests/` and `metadata/` subdirectories.

In your community-operators fork, create the new version directory, see [Github Pages](https://k8s-operatorhub.github.io/community-operators/#submitting-your-pr). 

**Note:** community-operators uses bare semver without the `v` prefix.

```bash
# Strip the leading 'v' for the directory name
export OH_VERSION=${VERSION#v}
mkdir -p operators/orc/${OH_VERSION}
```

Copy the generated bundle contents:

```bash
cp -r /path/to/orc/bundle/manifests operators/orc/${OH_VERSION}/
cp -r /path/to/orc/bundle/metadata  operators/orc/${OH_VERSION}/
```

Verify the directory structure matches the expected layout:

```
operators/orc/
├── ci.yaml
└── X.Y.Z/
    ├── manifests/
    │   ├── orc.clusterserviceversion.yaml
    │   ├── openstack.k-orc.cloud_*.yaml   (CRDs)
    │   └── ...
    └── metadata/
        └── annotations.yaml
```

Commit, push, and open a PR against `k8s-operatorhub/community-operators:main`.

Wait for community-operators CI to validate the bundle. The CI runs `operator-sdk bundle validate` and other checks.

**Note:** The [`ci.yaml`](https://k8s-operatorhub.github.io/community-operators/operator-ci-yaml/) in `operators/orc/` uses `updateGraph: semver-mode`, meaning OLM determines upgrade paths based on
semantic versioning automatically. 