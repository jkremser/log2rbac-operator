#!/bin/bash
CHART=${CHART:-"k8gb/k8gb"}

# deploy k8gb w/o the RBAC
kubectl create ns k8gb
helm repo add --force-update k8gb https://www.k8gb.io
helm -n k8gb upgrade -i k8gb $CHART --set k8gb.deployRbac=false
kubectl create sa k8gb

# make sure the role and rolebinding are not there
kubectl delete clusterrole new-k8gb-role
kubectl delete clusterrolebinding new-k8gb-role-binding

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


# watch how the role grows
echo "watch kubectl describe clusterrole new-k8gb-role"