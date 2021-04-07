# Create the K8S cluster on vSphere the hard way

## Provision nodes

- Provision a vSphere environment ready with 3 EXSi hosts
- Create a datacenter named "Datacenter"
- Create a cluster named "Cluster"
- Create a resource pool named "resource"
- Deploy an ubuntu 20 ova template [link](https://cloud-images.ubuntu.com/focal/current/)
- Create control plan vm named "control" from the ova template
  - CPUs: 2; Mem: 2GiB
- Create 3 worker vm named "worker-x" from the ova template
  - CPUs: 2; Mem: 4GiB

## Configure nodes

### Change the hostname on each node accordingly

- hostname on control plane vm: control
- hostname on worker vm: worker-x

### Apply the following configurations to all nodes

Run the following script as root user

``` bash
# Disable firewall
ufw disable

# Disable swap if there is one
swapoff -a

# Letting iptables see bridged traffic
modprobe br_netfilter
lsmod | grep br_netfilter
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
sysctl --system

# Install Docker CE
## Set up the repository:
### Install packages to allow apt to use a repository over HTTPS
apt-get update && apt-get install -y \
  apt-transport-https ca-certificates curl software-properties-common gnupg2

### Add Docker’s official GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

### Add Docker apt repository.
add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"

## Install Docker CE.
apt-get update && apt-get install -y \
  containerd.io=1.2.13-1 \
  docker-ce=5:19.03.8~3-0~ubuntu-$(lsb_release -cs) \
  docker-ce-cli=5:19.03.8~3-0~ubuntu-$(lsb_release -cs)

# Setup daemon.
cat > /etc/docker/daemon.json <<EOF
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF

mkdir -p /etc/systemd/system/docker.service.d

# Restart docker.
systemctl daemon-reload
systemctl restart docker

# Install kubeadmin, kubelet, kubectl
apt-get update && apt-get install -y apt-transport-https curl

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

cat <<EOF | sudo tee /etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF

apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl
```

Update the `/etc/hosts` to map the IP address to the hostname

``` bash
10.184.109.237 control
10.184.99.219  worker-0
...
```

### Run kubeadm on control plane node

``` bash
# kubeadm init
# By default, Calico uses 192.168.0.0/16 as the Pod network CIDR, though this can be configured in the calico.yaml file.
# For Calico to work correctly, you need to pass this same CIDR to the kubeadm init command using the
kubeadm init --pod-network-cidr=192.168.0.0/16 # root user
```

After above command, you should be able to see the output like below:

``` text
Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 10.184.109.237:6443 --token tmst40.4hqgq5hzpb6a50so \
    --discovery-token-ca-cert-hash sha256:d61e19ee70bfb62085c589fd2a1a50997e4a006ada011b61781e6b310a3ee46b

```

### Install pod networking add-on on control plane node

- Switch to regular user
- Get the kubeconfig ready

``` bash
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

- Install the calico add-on

``` bash
kubectl apply -f https://docs.projectcalico.org/v3.11/manifests/calico.yaml
```

### Join cluster on worker nodes

Run the join command from `kubeadm init` output on each worker nodes

``` bash
# root user
kubeadm join 10.184.109.237:6443 --token tmst40.4hqgq5hzpb6a50so \
    --discovery-token-ca-cert-hash sha256:d61e19ee70bfb62085c589fd2a1a50997e4a006ada011b61781e6b310a3ee46b
```

Now you shuold have a running 1 master + 3 worker nodes cluster.