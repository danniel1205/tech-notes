# Pod volumes

## Lab

### Create a pod with two containers which are sharing the same volume

- Apply the following yaml

``` yaml
apiVersion: v1
kind: Pod
metadata:
  name: shared-volume-pod
spec:
  containers:
  - image: ubuntu
    name: ubuntu1
    command: ['sh', '-c', 'sleep 3600']
    volumeMounts:
    - mountPath: "/ubuntu1"
      name: myvolume
  - image: ubuntu
    name: ubuntu2
    command: ['sh', '-c', 'sleep 3600']
    volumeMounts:
    - mountPath: "/ubuntu2"
      name: myvolume
  volumes:
  - name: myvolume
    emptyDir: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Always
```

- Create a file in `ubuntu1` container

``` bash
kubectl exec -it shared-volume-pod -c ubuntu1 -- touch /ubuntu1/ubuntu1.txt
```

- ls the file created in `ubuntu2` container

``` bash
kubectl exec -it shared-volume-pod -c ubuntu1 -- ls
```

### Create hostPath PV

Question: What will happen if hostPath is mounted in ReadWriteMany mode ?

Apply the following yaml.

``` yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: hostpath-pv
spec:
  capacity:
    storage: 2Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: hostpath-sc
  hostPath:
    path: "/tmp/hostpath-pv"
```

### Create PVC refernce to the PV

``` yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hostpath-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
  storageClassName: hostpath-sc
```

### Create Pod using PVC

``` yaml
apiVersion: v1
kind: Pod
metadata:
  name: hostpath-pod
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts:
      - mountPath: "/var/www/html"
        name: hostpath
  volumes:
    - name: hostpath
      persistentVolumeClaim:
        claimName: hostpath-pvc
```

**Note:** NFS is temporarily not working on my ubuntu node.

``` text
  Warning  FailedMount  19s  kubelet, worker-2  MountVolume.SetUp failed for volume "nfs-pv" : mount failed: exit status 32
Mounting command: systemd-run
Mounting arguments: --description=Kubernetes transient mount for /var/lib/kubelet/pods/44ae9dd3-4343-4aa1-82f7-4cb7f73d2947/volumes/kub
ernetes.io~nfs/nfs-pv --scope -- mount -t nfs w3-dbc301.eng.vmware.com:/dbc/w3-dbc301/gdaniel/dbc-nfs-volume /var/lib/kubelet/pods/44ae
9dd3-4343-4aa1-82f7-4cb7f73d2947/volumes/kubernetes.io~nfs/nfs-pv
Output: Running scope as unit: run-r72ea452706274b3ab3b6caa6801a5dae.scope
mount: /var/lib/kubelet/pods/44ae9dd3-4343-4aa1-82f7-4cb7f73d2947/volumes/kubernetes.io~nfs/nfs-pv: bad option; for several filesystems
 (e.g. nfs, cifs) you might need a /sbin/mount.<type> helper program.
```
