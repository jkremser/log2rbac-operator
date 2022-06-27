package internal

// Various constants used across the log2rbac
const (
	// CreatedByAnnotationKey https://kubernetes.io/docs/reference/labels-annotations-taints/#app-kubernetes-io-created-by
	CreatedByAnnotationKey = "app.kubernetes.io/created-by"

	// CreatedByAnnotationValue https://kubernetes.io/docs/reference/labels-annotations-taints/#app-kubernetes-io-created-by
	CreatedByAnnotationValue = "log2rbac"
)
