# Explore cert manager
cert-manager is a native Kubernetes certificate management controller. It can help with issuing certificates from a variety of sources, such as Let’s Encrypt, HashiCorp Vault, Venafi, a simple signing key pair, or self signed.

It will ensure certificates are valid and up to date, and attempt to renew certificates at a configured time before expiry.

It is loosely based upon the work of kube-lego and has borrowed some wisdom from other similar projects such as kube-cert-manager.

![cert-manager](https://i.imgur.com/vaPg3bl.png)

## Goals
* Deploy a sample web app
* Deploy ingress controller
* Deploy cert manager
* Use cert manager in ingress controller

## HTTPs background
### Workflow of https request
![https-workflow](https://i.imgur.com/CPDgrMF.png)
### Workflow of CA signing
![ca-signing-workflow](https://i.imgur.com/AavkL64.png)


## Steps
### Deploy the sample web app and ingress controller
**Note:** Please refer to this [doc](https://hackmd.io/-YoGO4NrQaioK0W5HQXp8Q) for how to deploy a sample web app and ingress controller(contour) to a kind cluster
### Deploy cert manager
### Create CA secrete in k8s cluster
* Generate CA key pairs
```
alias brew-openssl=/usr/local/Cellar/openssl@1.1/1.1.1f/bin/openssl

# Generate ca key
brew-openssl genrsa -out ca.key 2048

# Generate ca cert
brew-openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -subj "/CN=local.daniel.issuer" -out ca.crt
```
* Create k8s secret for the ca key pairs
```
kubectl create secret tls local-daniel-ca-key-pair --key=ca.key --cert=ca.crt
```
### Create the issuer
```
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: ca-issuer
spec:
  ca:
    secretName: local-daniel-ca-key-pair
```
### Making cert manager work with HTTPProxy
https://projectcontour.io/guides/cert-manager/

HttpProxy currently cannot work with cert manager direclty. We have to make a detour.
* Create HttpProxy
```
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: cowsay
spec:
  virtualhost:
    fqdn: local.daniel.io
    tls:
      secretName: local-daniel-io-cert
  routes:
    - services:
      - name: cowsay-blue
        port: 1323
      conditions:
        - prefix: /
```
**Note:** The HTTPProxy instance will under invalid status until the Ingress instance is created

* Create Ingress
```
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    cert-manager.io/issuer: "ca-issuer"
    ingress.kubernetes.io/force-ssl-redirect: "true"
    kubernetes.io/tls-acme: "true"
  name: cowsay
spec:
  rules:
  - host: local.daniel.io
    http:
      paths:
      - backend:
          serviceName: cowsay-blue
          servicePort: 1323
  tls:
  - hosts:
    - local.daniel.io
    secretName: local-daniel-io-cert
```
Now, you should be able to see a secret named `local-daniel-io-cert` got created.

### Access to the service
```
curl -k https://local.daniel.io/
```
Or access on web browser at: `https://local.daniel.io/`

## References
https://cert-manager.io/docs/

https://projectcontour.io/guides/cert-manager/

https://cert-manager.io/docs/installation/kubernetes/

https://cert-manager.io/docs/configuration/ca/



