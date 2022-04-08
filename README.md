[![CI](https://github.com/jkremser/log2rbac-operator/workflows/CI/badge.svg?branch=master)](https://github.com/jkremser/log2rbac-operator/actions/workflows/ci.yaml?query=workflow%3A%22CI%22+branch%3Amaster)
[![GitHub release](https://img.shields.io/github/release/jkremser/log2rbac-operator/all.svg?style=flat-square)](https://github.com/jkremser/log2rbac-operator/releases) 
[![Docker Pulls](https://img.shields.io/docker/pulls/jkremser/log2rbac.svg)](https://hub.docker.com/r/jkremser/log2rbac)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkremser/log2rbac-operator)](https://goreportcard.com/report/github.com/jkremser/log2rbac-operator)
[![License: MIT](https://img.shields.io/badge/License-Apache_2.0-yellow.svg)](https://opensource.org/licenses/Apache-2.0)

# log2rbac-operator
Kubernetes operator that helps you to set up the RBAC rules for your application. If requested, it scans the application's log files
for authorization errors and adds them as exceptions/rights to the associated `Role`. User has to allow this process by creating a
`RbacNegotiation` custom resource where they need to specify the app[*](#clarify) and `Role`.
Role can be either existing one or operator can create a new one for you and bind it to the service account that's configured with the deployment. Again if the service account is not there, it will be created by the operator.

<a name="clarify"></a>* App can be one of the following:
- Deployment
- StatefulSet
- DaemonSet
- Service
- ReplicaSet
- or key-value pair specifying the pod selector


## Quick Start

```bash
# clone repo and
make deploy
```

alternatively install it using [all-in-one yaml](deploy/all-in-one.yaml)

```bash
kubectl apply -f http://bit.do/log2rbac
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

## Kubectl Plugin

Installation:
```bash
kubectl krew install log2rbac
```

It can help with creating those `RbacNegotiation` custom resources by interactive TUI api.

It's located in [this repo](./kubectl-plugin)
