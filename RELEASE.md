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

## Helm Charts and ArtifactHub

Publishing new Helm chart versions to [ArtifactHub](https://artifacthub.io)
is fully automated as part of the `release` workflow so no manual steps are
required.

### How it works

When a tag is pushed, the `helm` job in the
[release workflow](.github/workflows/release.yaml) runs after the image
push and performs the following:

**Version stamping** — The `Chart.yaml` files in `charts/orc` and
`charts/orc-crds` contain development placeholders (`0.0.0-dev`).
The workflow derives the release version from the git tag (stripping
the `v` prefix for `version`, keeping it for `appVersion`) and stamps
both charts using `sed`.

**Packaging** — Both charts are packaged with `helm package`.

**Pushing** — The packaged charts are pushed to the OCI registry at
`oci://quay.io/orc/helm`:

```bash
helm push "orc-crds-${VERSION}.tgz" oci://quay.io/orc/helm
helm push "orc-${VERSION}.tgz" oci://quay.io/orc/helm
```

ArtifactHub automatically discovers new chart versions from the OCI
registry. The `artifacthub.io/*` annotations in each `Chart.yaml`
provide metadata (license, category, links) that ArtifactHub uses for
the listing.

### Verifying after a release

After the release workflow completes, confirm the new charts are
available:

```bash
# Pull and verify the chart versions
helm show chart oci://quay.io/orc/helm/orc --version X.Y.Z
helm show chart oci://quay.io/orc/helm/orc-crds --version X.Y.Z
```

Check that the new versions appear on ArtifactHub:
- [orc](https://artifacthub.io/packages/helm/orc/orc)
- [orc-crds](https://artifacthub.io/packages/helm/orc-crds/orc-crds)