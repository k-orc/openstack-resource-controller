# orc-crds

Helm chart for deploying ORC CRDs.

## How to update the Chart

The Chart has the same version as the `ORC`s, try to keep them equal.

CRDs are located [here](https://github.com/k-orc/openstack-resource-controller/tree/main/config/crd/bases)

**Do not forget to change the branch version**

```
cd charts/orc-crds
repo="k-orc/openstack-resource-controller"
branch="v2.0.3"
folder="config/crd/bases"
files=$(curl -s "https://api.github.com/repos/$repo/contents/$folder?ref=$branch" | jq -r '.[].download_url')
for file in $files
do
    filename=$(basename "$file")
    curl -o "templates/$filename" -L "$file"
done
```
