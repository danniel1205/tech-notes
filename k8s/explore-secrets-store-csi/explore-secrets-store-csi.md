# Explore secrets store csi

In K8S as of today, your secret objects are stored in ETCD as base64 encoded. Anyone who get the secret object or access
to the ETCD could get the secrets in plain text pretty easy. That is not safe !!! So people want to store their secrets
in external secret store which is out of K8S, but still want to access it within the Pod from K8S. That is how secrets
store csi and secret store provider could help.

Here are some great links talk about this topic:

- <https://www.youtube.com/watch?v=bIC4kLnrKN0>
- <https://www.youtube.com/watch?v=IznsHhKL428>
- <https://learn.hashicorp.com/vault/kubernetes/secret-store-driver#create-a-pod-with-secret-mounted>

## Lab

We are going to basically follow this [tutorial](https://learn.hashicorp.com/vault/kubernetes/secret-store-driver#create-a-pod-with-secret-mounted) from Hashicorp.

### Goals

- Using Hashcorp vault as the secret store
- Using secret store csi to make the external secret store available in K8S
- Create a Pod which consumes the secret in the secret store

### Steps

#### Create a kind cluster

``` bash
kind create cluster --config ./kind.yaml --image kindest/node:v1.18.0
```

#### Install the Vault Helm chart

``` bash
helm install vault \
    --set "server.dev.enabled=true" \
    --set "injector.enabled=false" \
    https://github.com/hashicorp/vault-helm/archive/v0.4.0.tar.gz
```

#### Create a secrete in Vault

``` bash
kubectl exec -it vault-0 -- /bin/sh
```

``` bash
/ $ vault kv put secret/demo-credential password="db-secret-password"
Key              Value
---              -----
created_time     2020-03-19T20:06:29.870406762Z
deletion_time    n/a
destroyed        false
version          1
```

#### Configure Kubernetes authentication

``` bash
# Enable Vault to use K8S AuthN
/ $ vault auth enable kubernetes
Success! Enabled kubernetes auth method at: kubernetes/
```

``` bash
# Configure the K8S AuthN mode to use service account token
vault write auth/kubernetes/config \
        token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
        kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443" \
        kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
```

``` bash
# Create Vault Policy
/ $ vault policy write demo-policy - <<EOF
path "sys/mounts" {
  capabilities = ["read"]
}

path "secret/data/demo-credential" {
  capabilities = ["read"]
}
EOF
Success! Uploaded policy: demo-policy
```

``` bash
# Create Vault role and bind to K8S service account
/ $ vault write auth/kubernetes/role/demo-role \
  bound_service_account_names=secrets-store-csi-driver \
  bound_service_account_namespaces=default \
  policies=demo-policy \
  ttl=20m
Success! Data written to: auth/kubernetes/role/demo-role
```

#### Install secrets store CSI driver

``` bash
git clone https://github.com/kubernetes-sigs/secrets-store-csi-driver.git

helm install csi secrets-store-csi-driver/charts/secrets-store-csi-driver

kubectl get pods
NAME                                 READY   STATUS    RESTARTS   AGE
csi-secrets-store-csi-driver-7p24r   3/3     Running   0          37s
csi-secrets-store-csi-driver-hp676   3/3     Running   0          37s
csi-secrets-store-csi-driver-ktt42   3/3     Running   0          37s
vault-0                              1/1     Running   0          15m
```

#### Apply the provider-vault executable and SecretProviderClass resource

Install the secrets store CSI driver is not done yet, it is just the interfaces. We need the actual implementations of it. So we have to install the provider-vault

``` bash
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: csi-secrets-store-provider-vault
  name: csi-secrets-store-provider-vault
spec:
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: csi-secrets-store-provider-vault
  template:
    metadata:
      labels:
        app: csi-secrets-store-provider-vault
    spec:
      serviceAccount: secrets-store-csi-driver
      tolerations:
      containers:
        - name: provider-vault-installer
          image: hashicorp/secrets-store-csi-driver-provider-vault:0.0.4
          imagePullPolicy: Always
          resources:
            requests:
              cpu: 50m
              memory: 100Mi
            limits:
              cpu: 50m
              memory: 100Mi
          env:
            - name: TARGET_DIR
              value: "/etc/kubernetes/secrets-store-csi-providers"
          volumeMounts:
            - mountPath: "/etc/kubernetes/secrets-store-csi-providers"
              name: providervol
      volumes:
        - name: providervol
          hostPath:
              path: "/etc/kubernetes/secrets-store-csi-providers"
      nodeSelector:
        beta.kubernetes.io/os: linux
EOF
```

Install `SecretProviderClass` which is similar to `StorageClass`, it could be used in `Pod` directly.

``` bash
cat <<EOF | kubectl apply -f -
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: vault-secrete-class
spec:
  provider: vault
  parameters:
    vaultAddress: "http://vault.default:8200"
    roleName: "demo-role"
    vaultSkipTLSVerify: "true"
    objects:  |
      array:
        - |
          objectPath: "/demo-credential"
          objectName: "password"
          objectVersion: ""
EOF
```

#### Create a pod with secret mounted

``` bash
cat <<EOF | kubectl apply -f -
kind: Pod
apiVersion: v1
metadata:
  name: nginx-secrets-store-inline
spec:
  containers:
  - image: nginx
    name: nginx
    volumeMounts:
    - name: secrets-store-inline
      mountPath: "/mnt/secrets-store"
      readOnly: true
  volumes:
    - name: secrets-store-inline
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: "vault-secrete-class"
EOF
```

``` bash
# Read the secret
kubectl exec nginx-secrets-store-inline -- cat /mnt/secrets-store/demo-credential
```
