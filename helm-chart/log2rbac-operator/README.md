# log2rbac-operator

![Version: 0.0.1](https://img.shields.io/badge/Version-0.0.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.0.3](https://img.shields.io/badge/AppVersion-v0.0.3-informational?style=flat-square)

A Helm chart for log2rbac Kubernetes operator

```
 _             ____       _
| | ___   __ _|___ \ _ __| |__   __ _  ___
| |/ _ \ / _` | __) | '__| '_ \ / _` |/ __|
| | (_) | (_| |/ __/| |  | |_) | (_| | (__
|_|\___/ \__, |_____|_|  |_.__/ \__,_|\___|
         |___/
```

**Homepage:** <https://jkremser.github.io/log2rbac-operator>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Jiri Kremser | jiri.kremser@gmail.com |  |

## Source Code

* <https://github.com/jkremser/log2rbac-operator>

## Requirements

Kubernetes: `>= 1.14.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ |
| deploy | object | `{"jaeger":false,"operator":true,"rbac":true,"service":true}` | what should be deployed |
| deploy.jaeger | bool | `false` | whether the jaeger should be deployed with the operator (use together with `tracing.enabled = true`) |
| deploy.operator | bool | `true` | whether the operator itself should be deployed (Deployment) |
| deploy.rbac | bool | `true` | whether the rbac resources should be also deployed (ServiceAccount, ClusterRole, ClusterRoleBinding) |
| deploy.service | bool | `true` | whether the service for metrics and open-telemetry should be deployed (Service) |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"Always"` | translates to pod's `spec.imagePullPolicy` |
| image.repository | string | `"jkremser/log2rbac"` | container image repo (can be prepended by image registry) |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion. |
| imagePullSecrets | list | `[]` |  |
| metrics.enabled | bool | `true` | should the metrics be enabled (additional arg for log2rbac binary) |
| metrics.nodePort | int | `30081` | Port on node that will be used for metrics. This make sense only for serviceType = NodePort, otherwise it's ignored |
| metrics.port | int | `8080` | on which port the metrics server should listen |
| metrics.serviceType | string | `"NodePort"` | typeof the service for metrics (ClusterIP, NodePort, LoadBalancer). Consult https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` | https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ |
| operator | object | `{"colors":true,"noBanner":false,"restartPods":true,"syncIntervals":{"afterNoLogs":30,"afterNoRbacEntry":5,"afterPodRestart":20}}` | operator specific settings |
| operator.colors | bool | `true` | should the logs be colorcul (env var `COLORS`) |
| operator.noBanner | bool | `false` | should the ascii logo be printed in the logs (env var `NO_BANNER`) |
| operator.restartPods | bool | `true` | whether the operator should be restarting the pods after modifying the role (env var `SHOULD_RESTART_APP_PODS`) if not set defaults to `true` |
| operator.syncIntervals.afterNoLogs | int | `30` | if it was not possible to get the logs, how long to wait for the next check (env var `SYNC_INTERVAL_AFTER_NO_LOGS_SECONDS`) value represents the number of seconds |
| operator.syncIntervals.afterNoRbacEntry | int | `5` | if no rbac related entry was found in logs, how long to wait for the next check (env var `SYNC_INTERVAL_AFTER_NO_RBAC_ENTRY_MINUTES`) value represents the number of minutes |
| operator.syncIntervals.afterPodRestart | int | `20` | how long to wait after rbac entry was added and pod was restarted by the operator (env var `SYNC_INTERVAL_AFTER_POD_RESTART_SECONDS`) value represents the number of seconds |
| podAnnotations | object | `{}` | additional annotations that will be applied on operator's pod |
| podLabels | object | `{}` | additional labels that will be applied on operator's pod |
| podSecurityContext | string | `nil` |  |
| resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | resource definitions for operator's pod see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities | object | `{"drop":["ALL"]}` | For more options consult https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#securitycontext-v1-core |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| tolerations | list | `[]` | https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ |
| tracing.enabled | bool | `false` | if the application should be sending the traces to OTLP collector (env var `TRACING_ENABLED`) |
| tracing.endpoint | string | `"localhost:4318"` | `host:port` where the spans (traces) should be sent, sets the `OTEL_EXPORTER_OTLP_ENDPOINT` env var |
| tracing.jaegerImage | object | `{"pullPolicy":"Always","repository":"jaegertracing/all-in-one","tag":"1.33.0"}` | if `deploy.jaeger==true` this image will be used in the deployment for Jaeger |
| tracing.samplingRatio | string | `nil` | float representing the ratio of how often the span should be kept/dropped (env var `TRACING_SAMPLING_RATIO`) if not specified, the AlwaysSample will be used which is the same as 1.0. `0.1` would mean that 10% of samples will be kept |
| tracing.sidecarImage | object | `{"pullPolicy":"Always","repository":"otel/opentelemetry-collector","tag":"0.48.0"}` | OpenTelemetry collector into which the log2rbac operator sends the spans. It can be further configured to send its data to somewhere else using exporters (Jaeger for instance) |
