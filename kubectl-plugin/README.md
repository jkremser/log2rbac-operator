# log2rbac kubectl plugin

Simple TUI based shell script to interface with the log2rbac operator. It can (un)deploy the operator and create the `RbacNegotiation` custom resources for various types of K8s kinds.


## Quick start

```bash
sudo ln -s $PWD/kubectl-log2rbac /usr/local/bin/kubectl-log2rbac
kubectl log2rbac
```

or install the latest released version using Krew

```bash
λ kubectl krew install --manifest-url=https://raw.githubusercontent.com/jkremser/log2rbac-operator/main/kubectl-plugin/log2rbac.yaml
Installing plugin: log2rbac
Installed plugin: log2rbac
\
 | Use this plugin:
 | 	kubectl log2rbac
 | Documentation:
 | 	https://github.com/jkremser/log2rbac-operator/tree/main/kubectl-plugin
/
```
