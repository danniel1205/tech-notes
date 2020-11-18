# How pod is created with network configured

- User creates a `Pod` directly or through `Deployment`. A `Pod` object will be created.
- `kube-scheduler` puts the `Pod` object in scheduling queue
- `kube-scheduler` dequeue the `Pod` object and schedule it to the node has enough resource by updaing `spec.nodeName`
- `kubelet` wakes up with the information of the `Pod` object
- `kubelet` talks to container runtime via `CRI`
- container runtime needs to interact with K8S network plugin via `CNI` to create the network for the pod.
  - Add: the `Pod` object is created, configure the network for that `Pod` object
  - Delete: the `Pod` object is deleted, cleanup the network resource for that `Pod object
  - Check: can be called periodically to make sure everthing is good
  ![cri-cni-interaction](./resources/cri-cni-interaction.png)
- `CNI` will create `veth` pair and assig IP address to that `Pod` object
  ![pod-to-pod-networking](./resources/pod-to-pod-network-same-node.gif)