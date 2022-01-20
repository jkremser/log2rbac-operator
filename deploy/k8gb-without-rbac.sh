#!/bin/bash

# deploy k8gb w/o the RBAC
kubectl create ns k8gb
helm repo add --force-update k8gb https://www.k8gb.io
helm -n k8gb upgrade -i k8gb k8gb/k8gb --set k8gb.deployRbac=false

# create RbacNegotiation for k8gb
cat <<CustomResource | kubectl apply -f -
apiVersion: kremser.dev/v1
kind: RbacNegotiation
metadata:
  name: for-k8gb
spec:
  for:
    namespace: k8gb
    kind: Deployment
    name: k8gb
  role:
    name: new-k8gb-role
    isClusterRole: true
    createIfNotExist: true
CustomResource
