# Create user accounts

## Lab

- Create client key pair for user Alice, and sign with K8S CA

``` bash
# Create private key for Alice
openssl genrsa -out alice.key 2048

# Create cert signing request for Alice's cert
openssl req -new -key alice.key -out alice.csr -subj "/CN=alice/O=bookstore"

# Sign Alice's cert with K8S CA
sudo openssl x509 -req -in alice.csr -CA /etc/kubernetes/pki/ca.crt -CAkey /etc/kubernetes/pki/ca.key -CAcreateserial -out alice.crt -days 180
```

- Create user entry in kubeconfig

``` bash
kubectl config set-credentials alice --client-certificate alice.crt --client-key alice.key
```

- Create context in kubeconfig

``` bash
kubectl config set-context alice-context --cluster=kubernetes --user=alice
```

- Create RBAC Roles for default namespace
  - Reader

    ``` yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      namespace: default
      name: reader
    rules:
    - apiGroups: ["", "apps"] # "" indicates the core API group
      resources: ["pods", "deployments", "services"]
      verbs: ["get", "watch", "list"]
    ```

  - Writer

    ``` yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      namespace: default
      name: writer
    rules:
    - apiGroups: ["", "apps"] # "" indicates the core API group
      resources: ["pods", "deployments", "services"]
      verbs: ["get", "watch", "list", "create", "update", "delete", "patch"]
    ```

- Create an RBAC Role binding
  - Bind Alice to Reader role and test it out

    ``` yaml
    apiVersion: rbac.authorization.k8s.io/v1
    # This role binding allows "jane" to read pods in the "default" namespace.
    # You need to already have a Role named "pod-reader" in that namespace.
    kind: RoleBinding
    metadata:
      name: reader-rolebinding
      namespace: default
    subjects:
      # You can specify more than one "subject"
    - kind: User
      name: alice # "name" is case sensitive
      apiGroup: rbac.authorization.k8s.io
    roleRef:
      # "roleRef" specifies the binding to a Role / ClusterRole
      kind: Role #this must be Role or ClusterRole
      name: reader # this must match the name of the Role or ClusterRole you wish to bind to
      apiGroup: rbac.authorization.k8s.io
    ```

  - Bind Alice to writer role and test it out

    ``` yaml
    apiVersion: rbac.authorization.k8s.io/v1
    # This role binding allows "jane" to read pods in the "default" namespace.
    # You need to already have a Role named "pod-reader" in that namespace.
    kind: RoleBinding
    metadata:
      name: writer-rolebinding
      namespace: default
    subjects:
      # You can specify more than one "subject"
    - kind: User
      name: alice # "name" is case sensitive
      apiGroup: rbac.authorization.k8s.io
    roleRef:
      # "roleRef" specifies the binding to a Role / ClusterRole
      kind: Role #this must be Role or ClusterRole
      name: writer # this must match the name of the Role or ClusterRole you wish to bind to
      apiGroup: rbac.authorization.k8s.io
    ```

- Create RBAC ClusterRoles
  - ClusterReader

    ``` yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
    name: cluster-reader
    rules:
    - apiGroups: ["", "apps"] # "" indicates the core API group
        resources: ["pods", "deployments", "services"]
        verbs: ["get", "watch", "list"]
    ```

- Create an RBAC ClusterRole binding

``` yaml
apiVersion: rbac.authorization.k8s.io/v1
# This role binding allows "jane" to read pods in the "default" namespace.
# You need to already have a Role named "pod-reader" in that namespace.
kind: ClusterRoleBinding
metadata:
  name: reader-clusterrole-binding
subjects:
# You can specify more than one "subject"
- kind: User
  name: alice # "name" is case sensitive
  apiGroup: rbac.authorization.k8s.io
roleRef:
  # "roleRef" specifies the binding to a Role / ClusterRole
  kind: ClusterRole
  name: cluster-reader # this must match the name of the Role or ClusterRole you wish to bind to
  apiGroup: rbac.authorization.k8s.io
```
