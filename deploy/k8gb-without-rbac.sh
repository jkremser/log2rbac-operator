#!/bin/bash
CHART=${CHART:-"k8gb/k8gb"}

# apply CRD
make install

# deploy k8gb w/o the RBAC
kubectl create ns k8gb 2> /dev/null || true
kubectl create sa k8gb -n k8gb 2> /dev/null || true
helm repo add --force-update k8gb https://www.k8gb.io
helm -n k8gb upgrade -i k8gb $CHART --set k8gb.deployRbac=false

# make sure the role and rolebinding are not there
kubectl delete clusterrole new-k8gb-role 2> /dev/null || true
kubectl delete clusterrolebinding new-k8gb-role-binding 2> /dev/null || true

# create RbacNegotiation for k8gb
cat <<CustomResource | kubectl apply -n k8gb -f -
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
