# Release procedure

Generate the new `install.yaml` manifest, and tag the release:
```bash
export VERSION=vX.Y.Z
make build-installer IMG=quay.io/orc/openstack-resource-controller:$VERSION

... Update `Installation` in `README.md` to refer to the new release version.

git add dist README.md
git commit -m "Release $VERSION"
git tag -s -a $VERSION -m $VERSION
git push origin
git push origin tag $VERSION
```

Pushing the tag will trigger the tagged image build. Monitor the [release image
workflow](https://github.com/k-orc/openstack-resource-controller/actions/workflows/release_image.yaml)
and when it is done, check that you can successfully pull the image with:
```bash
podman pull quay.io/orc/openstack-resource-controller:$VERSION
```

Finally, create the release in github. We must attach the generated `install.yaml` to the release artifacts:
```bash
gh release create -d --generate-notes --verify-tag -t "Release $VERSION" $VERSION
gh release upload $VERSION dist/install.yaml
```

Edit the release draft from github, and publish.
