# Explore k14s

## What is k14s

k14s has been renamed to Carvel.

[Carvel tool official website](https://carvel.dev/)

## Lab

- Install Carvel tool
- Define the ytt template & values for [cow say app](https://github.com/danniel1205/sample-cowsay-web-app)
- Build the ytt template to be OCI image
- Using `App` from kapp-controller to deploy the cow say app

### Install Carvel tool

The official Carvel page has the installation guide: [link](https://carvel.dev/)

``` bash
brew tap k14s/tap
brew install ytt kbld kapp imgpkg kwt vendir
```

### Create the ytt template and values

[Templates](./ytt/templates)
[Values](./ytt/values.yaml)

### Build the ytt template by using docker

``` bash
docker build -f Dockerfile -t danielguo/cow-say-template:v1 .
docker push danielguo/cow-say-template:v1
```

### Deploy cow say app

- Switch to root dir of this project
- Install kapp-controller: `kubectl apply -f https://github.com/vmware-tanzu/carvel-kapp-controller/releases/latest/download/release.yml`
- Create RBAC for kapp-controller. [Link to the Yaml](./artifacts/rbac.yaml)
- Create Secret which contains the cow say ytt template values.
  `kubectl create secret generic cow-say-value --from-file=values.yaml=./ytt/values.yaml`
- Create App CR for cow-say app [Link to the Yaml](./artifacts/app.yaml)
- Double check the deployed app
  ```
    kapp list
    Target cluster 'https://127.0.0.1:32776' (nodes: kind-control-plane)
    
    Apps in namespace 'default'
    
    Name              Namespaces  Lcs   Lca
    cow-say-app-ctrl  default     true  22s
    
    Lcs: Last Change Successful
    Lca: Last Change Age
    
    1 apps
    
    Succeeded
  ```
- Enable port forwarding and access the app `kubectl port-forward service/cow-say 8081:1323`
- ``` bash
    curl localhost:8081
     _______
    < Hello >
    -------
            \   ^__^
             \  (oo)\_______
                (__)\       )\/\
                    ||----w |
                    ||     ||%
   ```

## References

- <https://tanzu.vmware.com/content/blog/introducing-k14s-kubernetes-tools-simple-and-composable-tools-for-application-deployment>
- <https://github.com/vmware-tanzu/carvel-ytt/blob/develop/docs/faq.md>
- <https://github.com/vmware-tanzu/carvel-ytt/blob/develop/docs/lang-ref-ytt.md>
- <https://github.com/vmware-tanzu/carvel-kapp-controller/blob/develop/docs/app-spec.md>
