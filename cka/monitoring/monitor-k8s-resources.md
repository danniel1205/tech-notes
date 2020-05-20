# Monitoring K8S resources

## metrics-server

metrics-server collects the cluster-wide metrics from kubelet on each node, and expose them in the K8S apiserver
through `Metrics API`.

metrics-server is not built-in, you have to deploy it. Github: <https://github.com/kubernetes-sigs/metrics-server>

### Deploy metrics-server on K8S cluster

- Before metrics-server is deployed, you could not get anything back from `kubectl top`

``` bash
ubuntu@control:~$ kubectl top node control
Error from server (NotFound): the server could not find the requested resource (get services http:heapster:)
ubuntu@control:~$ kubectl top pod static-nginx-worker-4
Error from server (NotFound): the server could not find the requested resource (get services http:heapster:)
```

- Deplopy metrics-server

  - [Need to make sure if the k8s sig github could be accessible in cka exam]: <https://github.com/kubernetes-sigs/metrics-server>
  - `kubectl apply <components.yaml which is downloaded from above link>`
  
- Run `kubectl top nodes`

``` bash
ubuntu@control:~/monitoring$ kubectl top nodes
NAME       CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%
control    343m         17%    1186Mi          62%
worker-0   126m         6%     641Mi           16%
worker-1   120m         6%     970Mi           25%
worker-2   108m         5%     746Mi           19%
```

### Deploy kube-state-metrics
