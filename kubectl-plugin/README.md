# log2rbac kubectl plugin

Simple TUI based shell script to interface with the log2rbac operator. It can (un)deploy the operator and create the `RbacNegotiation` custom resources for various types of K8s kinds.


## Quick start

```bash
sudo ln -s $PWD/kubectl-log2rbac /usr/local/bin/kubectl-log2rbac
kubectl log2rbac
```
