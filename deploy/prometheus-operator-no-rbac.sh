# create only a service account 
kubectl create sa prometheus-operator -n monitoring

kubectl apply -f https://github.com/prometheus-operator/kube-prometheus/raw/main/manifests/prometheusOperator-deployment.yaml

# create RbacNegotiation for the operator
cat <<CustomResource | kubectl apply -n monitoring -f -
apiVersion: kremser.dev/v1
kind: RbacNegotiation
metadata:
  name: for-prom
spec:
  for:
    namespace: monitoring
    kind: Deployment
    name: prometheus-operator
  role:
    name: new-prome-operator-role
    isClusterRole: true
    createIfNotExist: true
CustomResource
