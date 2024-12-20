export VERSION=v0.9.0
make build-installer IMG=quay.io/orc/openstack-resource-controller:$VERSION
git add dist
git commit -m "Release $VERSION"
git tag -s -a $VERSION -m $VERSION
git push origin
git push origin tag $VERSION

gh release create -d --generate-notes --verify-tag -t "Release $VERSION" $VERSION
gh release upload $VERSION dist/install.yaml

Sanity check generated release
* Monitor image build at: https://github.com/k-orc/openstack-resource-controller/actions/workflows/release_image.yaml
* `podman pull quay.io/orc/openstack-resource-controller:$VERSION` should succeed

Edit -> Publish
