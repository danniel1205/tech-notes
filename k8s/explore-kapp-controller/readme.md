# Explore kapp controller

## Goal

- Create a test app template
- Build the template image
- Deploy kapp controller
- Create an App which fetches the app template and applies the data values from secret

**Note:** You might need to create a dockerhub credential secret to make this demo work.

``` bash
kubectl create secret docker-registry regcred --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email>

where:
<your-registry-server> is your Private Docker Registry FQDN. (E.g., https://index.docker.io/v1 for DockerHub)
<your-name> is your Docker username.
<your-pword> is your Docker password.
<your-email> is your Docker email.
```