# Authentication in K8S

## Users

- Service accounts managed by K8S
- Normal users: Any user presents a valid certificate signed by cluster's CA is considered authenticated

## Authentication strategies

### X509 Client certs

![x509-client-cert](./resources/x509-client-cert.png)

- Using `openssl` to generate private key and csr file
- Using `CertificateSigningRequest` API to submit the cert signing request
- Using `kubectl certificate approve xxx` to approve the CSR
- Create `Role` and `RoleBinding` to the new user

See more details:

- <https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#normal-user>
- <https://kubernetes.io/docs/reference/access-authn-authz/authentication/#x509-client-certs>

### Static token file

![static-token-file](./resources/static-token-file.png)

- Provide a csv file as the token file to `--token-auth-file=SOMEFILE` of api-server
- Put a bearer token in https header when any http client wants to authenticate

### Bootstrap token

Bootstrap tokens are a simple bearer token that is meant to be used when creating new clusters or joining new nodes to an existing cluster. The `Bootstrap Token` is dynamically managed by K8S for above purposes. See [bootstrap-steps](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet-tls-bootstrapping/#bootstrap-initialization) for more details on how this token is being used.

See more details:

- <https://kubernetes.io/docs/reference/access-authn-authz/authentication/#bootstrap-tokens>
- <https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/>
- <https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet-tls-bootstrapping/#bootstrap-initialization>

### Service account tokens

A service account is an automatically enabled authenticator that uses signed bearer tokens to verify requests. When running a pod or deployment using a service account, the `secret` contains the bearer token will be mounted to the `pod`. The Bearer tokens are mounted into pods at well-known locations, and allow in-cluster processes to talk to the API server. If no service account specified, the default one will be used.

``` bash
$ kubectl run -it --rm testpod --restart=Never --image=alpine -- sh
If you don't see a command prompt, try pressing enter.
/ # ls -l /var/run/secrets/kubernetes.io/serviceaccount/
total 0
lrwxrwxrwx    1 root     root            13 Nov  8 11:45 ca.crt -> ..data/ca.crt
lrwxrwxrwx    1 root     root            16 Nov  8 11:45 namespace -> ..data/namespace
lrwxrwxrwx    1 root     root            12 Nov  8 11:45 token -> ..data/token

/ # cat /var/run/secrets/kubernetes.io/serviceaccount/token && echo
eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJkZWZhdWx0Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZWNyZXQubmFtZSI6ImRlZmF1bHQtdG9rZW4tcHh6ZjciLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiZGVmYXVsdCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6ImUzMzFmNTU1LTAxOTktMTFlYS1hYTVhLTQyMDEwYTgwMDBhYSIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpkZWZhdWx0OmRlZmF1bHQifQ.I3jFqRd7ANj4BidzoW-v_YO1N_z2p33Ex1biaz_J97QK9CRlIFmK2KC1k0RbQux3zo72xsLhbPjNH40pBzqr2TzVjZKFD7T8-ihDj1Og5L_BAud-CtT4e-0zcvC5rTKcWzkUn5a64TUSwF5Q6I3KjeaE3pmDFwG4I6XLuODCVOwRkfn0V_LzjqLy3nXfUK8FpIkeBrRd9QN68PG9YFH9lKMgGZLvuG_m6K6EFxqBpVMec8SPsG77GGRezC9Mjsyxp2Wie-j8vUVb5et2o4ShKa8sp-Nqum4bpBxGQ9NBo3qlefuqYGYivGZBnL8dienBLFbO5swSHUL6vxTgaRg2kA
/ #
```

### OpenID Connect Tokens

![oidc-token](./resources/oidc-token.png)

![oidc-token-2](./resources/oidc-token-2.png)

- Login to your identity provider
- Your identity provider will provide you with an access_token, id_token and a refresh_token
- When using kubectl, use your id_token with the --token flag or add it directly to your kubeconfig
- kubectl sends your id_token in a header called Authorization to the API server
- The API server will make sure the JWT signature is valid by checking against the certificate named in the configuration
- Check to make sure the id_token hasn't expired
- Make sure the user is authorized
- Once authorized the API server returns a response to kubectl
- kubectl provides feedback to the user

### Authenticating proxy

We could authenticate normal users by using an authenticating proxy, which sets the request header values, e.g. `X-Remote-User`. API server could be configured to identify users from the header values.

``` bash
// Example flags to api server
--requestheader-username-headers=X-Remote-User
--requestheader-group-headers=X-Remote-Group
--requestheader-extra-headers-prefix=X-Remote-Extra-
```

In order to prevent header spoofing, the authenticating proxy is required to present a valid client certificate to the API server for validation against the specified CA before the request headers are checked.

### Webhook token authentication

We could delegate the normal users authentication to a remote authn service via webhook as show in below. A `TokenReview` is sent to webhook. The webhook process the request and response a `TokenReview` back to indicate if the token is authenticated or not.

![webhook-authn](./resources/webhook-authn.png)

## References

- -<https://www.magalix.com/blog/kubernetes-authentication>
- <https://learnk8s.io/kubernetes-custom-authentication#using-the-webhook-token-authentication-plugin>
