// Package k8s provides utilities for interacting with Kubernetes clusters.
// It uses the official client-go library.
//
// To enable: add client-go to go.mod and uncomment the implementation below.
// Then add a "k8s_pods" tool in internal/tools/tools.go.
//
// go get k8s.io/client-go@latest
package k8s

// Stub — extend this to add real Kubernetes support.
// Example tools you could build here:
//
//   func ListPods(ctx, namespace) (string, error)     — list pod names + status
//   func GetLogs(ctx, pod, namespace) (string, error) — fetch pod logs
//   func Rollout(ctx, deploy, namespace) (string, error) — trigger rollout restart
//   func ScaleDeployment(ctx, deploy, ns, replicas) (string, error)
//
// See: https://github.com/kubernetes/client-go/tree/master/examples
