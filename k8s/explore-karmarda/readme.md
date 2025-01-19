# Explore Karmarda (WIP)

*Note*: This is still working in progress.

Official website: <https://karmada.io/>

## Propagate a deployment

Goal: Create a host cluster + member clusters, deploy Nginx across multiple clusters. Guide:
<https://karmada.io/docs/get-started/nginx-example>.

### Take a closer look into Karmarda control plane

```shell
k config get-contexts
CURRENT   NAME                CLUSTER             AUTHINFO            NAMESPACE
          karmada-apiserver   karmada-apiserver   karmada-apiserver
*         karmada-host        kind-karmada-host   kind-karmada-host
```

```shell
k get deployments -A
NAMESPACE            NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
karmada-system       karmada-aggregated-apiserver          2/2     2            2           27m
karmada-system       karmada-apiserver                     1/1     1            1           27m
karmada-system       karmada-controller-manager            2/2     2            2           27m
karmada-system       karmada-descheduler                   2/2     2            2           27m
karmada-system       karmada-kube-controller-manager       1/1     1            1           27m
karmada-system       karmada-metrics-adapter               1/1     1            1           27m
karmada-system       karmada-scheduler                     2/2     2            2           27m
karmada-system       karmada-scheduler-estimator-member1   2/2     2            2           27m
karmada-system       karmada-scheduler-estimator-member2   2/2     2            2           26m
karmada-system       karmada-scheduler-estimator-member3   2/2     2            2           26m
karmada-system       karmada-search                        2/2     2            2           27m
karmada-system       karmada-webhook                       2/2     2            2           27m
kube-system          coredns                               2/2     2            2           32m
local-path-storage   local-path-provisioner                1/1     1            1           32m

```

```shell

k config use-context karmada-apiserver
Switched to context "karmada-apiserver".

k get apiservice
NAME                                   SERVICE                                       AVAILABLE   AGE
v1.                                    Local                                         True        28m
v1.admissionregistration.k8s.io        Local                                         True        28m
v1.apiextensions.k8s.io                Local                                         True        28m
v1.apps                                Local                                         True        28m
v1.authentication.k8s.io               Local                                         True        28m
v1.authorization.k8s.io                Local                                         True        28m
v1.autoscaling                         Local                                         True        28m
v1.batch                               Local                                         True        28m
v1.certificates.k8s.io                 Local                                         True        28m
v1.coordination.k8s.io                 Local                                         True        28m
v1.discovery.k8s.io                    Local                                         True        28m
v1.events.k8s.io                       Local                                         True        28m
v1.flowcontrol.apiserver.k8s.io        Local                                         True        28m
v1.networking.k8s.io                   Local                                         True        28m
v1.node.k8s.io                         Local                                         True        28m
v1.policy                              Local                                         True        28m
v1.rbac.authorization.k8s.io           Local                                         True        28m
v1.scheduling.k8s.io                   Local                                         True        28m
v1.storage.k8s.io                      Local                                         True        28m
v1alpha1.apps.karmada.io               Local                                         True        28m
v1alpha1.autoscaling.karmada.io        Local                                         True        28m
v1alpha1.cluster.karmada.io            karmada-system/karmada-aggregated-apiserver   True        28m
v1alpha1.config.karmada.io             Local                                         True        28m
v1alpha1.multicluster.x-k8s.io         Local                                         True        28m
v1alpha1.networking.karmada.io         Local                                         True        28m
v1alpha1.policy.karmada.io             Local                                         True        28m
v1alpha1.remedy.karmada.io             Local                                         True        28m
v1alpha1.search.karmada.io             karmada-system/karmada-search                 True        28m
v1alpha1.work.karmada.io               Local                                         True        28m
v1alpha2.work.karmada.io               Local                                         True        28m
v1beta1.custom.metrics.k8s.io          karmada-system/karmada-metrics-adapter        True        28m
v1beta1.metrics.k8s.io                 karmada-system/karmada-metrics-adapter        True        28m
v1beta2.custom.metrics.k8s.io          karmada-system/karmada-metrics-adapter        True        28m
v1beta3.flowcontrol.apiserver.k8s.io   Local                                         True        28m
v2.autoscaling                         Local                                         True        28m
```

```shell
k apply -f resources/simple-nginx/deployment.yaml
k apply -f resources/simple-nginx/propagationpolicy.yaml
```

Key takeaways:

- Pods will not be created on control plane
- The karmada-scheduler on Control plane watches the events of deployment and CRUD of any member-cluster.deployment.
- PropagationPolicy specifies how replicas are splitted across multiple clusters.
- Changing PropagationPolicy will end up with updating the resources placement. (Running jobs will be terminated).

---

Questions:

- Why there are two different contexts on host cluster

TODOs:

- Pods to Pods communication across clusters
- Deep dive into karmada-scheduler-estimator


