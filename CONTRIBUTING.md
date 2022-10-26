## Local Setup
Assuming you have configured the connection to a Kubernetes cluster in your `$KUBECONFIG` or `~/.kube/config` all you need to do to run the latest
version of the project is:

```bash
make run
```

### Debugging
In your favorite IDE just run the main.go in the debug mode, all the environment variables should have the default fallback values so no further configuration is needed.


## Run e2e on the localhost

We are assuming the k3s & k3d combo here, but any Kubernetes should do.

```bash
# this image name will be build and deployed in the cluster (arbitrary)
export IMG=candidate
```

```bash
# build the container image
make container-img
```

```bash
# create cluster and import the image (if having multiple clusters, you may want to use -c clusterName)
k3d cluster create --no-lb --k3s-arg "--disable=traefik,servicelb,metrics-server,local-storage@server:*"
k3d image import ${IMG}:latest
```

```bash
# deploy the CRD and operator
make install deploy
kubectl wait deploy/log2rbac -n log2rbac --for condition=available --timeout=2m
```


```bash
# run the e2e tests
cd e2e-test/ && go mod download && gotest ./...
```
