# Explore custom controller leader election

## Goals

When working on the custom controller we found that one controller might have so many resources to be enqueued and
reconciled, e.g., the controller for K8S addons of all clusters. The question came out that if the controller could have
multiple replicas and shard the resources to different controller replicas so that the traffic could be balanced.

## What I found

After reading through the articles in the [Reference Section](#references) and the [controller-runtime#manager.go](https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/manager/manager.go),
the following are what I found:

- controller-runtime uses `ConfigMap` or `Endpoints` as the lock primitives for leader election. ([Code](https://github.com/kubernetes-sigs/controller-runtime/blob/197751df6040ec99414574e89f3fa73914ce335d/pkg/leaderelection/leader_election.go#L54))
  The default one uses `ConfigMap`.
- If leader election is enabled for a controller, there will be only one active controller instance to reconcile.
```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "6c48dc90.dg.k8s",
	})
```
- If leader election is NOT enabled, and you have multiple controller replicas, all the controller replicas will watch on
  the same event and reconcile concurrently. I have tried this by running a [custom controller](https://github.com/danniel1205/explore-k8s-custom-controller/blob/master/config/manager/manager.yaml#L29)
  I implemented for demo. When I create a `ConfigMap` all the replicas try to reconcile.
  
So the conclusion is: THERE IS NO CLIENT SIDE SHARDING SUPPORTED IN CONTROLLER RUNTIME.


## References

- <https://kubernetes.io/blog/2016/01/simple-leader-election-with-kubernetes/>
- <https://carlosbecker.com/posts/k8s-leader-election>
- <https://medium.com/michaelbi-22303/deep-dive-into-kubernetes-simple-leader-election-3712a8be3a99>