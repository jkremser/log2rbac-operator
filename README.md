[![License: MIT](https://img.shields.io/badge/License-Apache_2.0-yellow.svg)](https://opensource.org/licenses/Apache-2.0)

[comment]: <> ([![Build Status]&#40;https://github.com/jkremser/log2rbac-operator/workflows/build/badge.svg?branch=master&#41;]&#40;https://github.com/k8gb-io/log2rbac-operator/actions?query=workflow%3A%22Golang+lint+and+test%22+branch%3Amaster&#41;)

# log2rbac-operator
Kubernetes operator that helps you to set up the RBAC rules for your application. If requested, it scans the application's log files
for authorization errors and adds them as exceptions/rights to the associated `Role`. User have to allow this process by creating a
`RbacNegotiation` custom resource where they need to specify the app (only Deployments are currently supported) and `Role`.
Role can be either existing one or operator can create a new one for you and bind it to the service account that's configured with the deployment.


## Quick Start

```bash
make deploy
```

```bash
# create RbacNegotiation for k8gb
cat <<CustomResource | kubectl apply -f -
apiVersion: kremser.dev/v1
kind: RbacNegotiation
metadata:
  name: for-k8gb
spec:
  for:
    kind: Deployment
    name: k8gb
    namespace: k8gb
  role:
    name: new-k8gb-role
    isClusterRole: true
    createIfNotExist: true
CustomResource
```