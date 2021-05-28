# Explore OpenLDAP

## Goals

- Create a CA cert which could be used to sign other certs
- Create `ldap.crt` and `ldap.key` pairs and get signed by the CA cert
- Create OpenLDAP + phpldapadmin by using docker-compose
  - Use the generated ldap certs file by updating the volume mounts in docker-compose file
- Create a sample user account `danielg` in LDAP
- Bring up a BookStack app and configure it to use LDAP
- Using `danielg` to log in BookStack should succeeds

## Lab

### Create certificates

``` bash
# Generate CA certs
brew-openssl genrsa -out ca.key 2048
brew-openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -subj "/CN=company.issuer" -out ca.crt

# Generate ldap certs
brew-openssl genrsa -out ldap.key 2048
brew-openssl req -new -key ldap.key -out ldap.csr
# When create the cert, I have used "dc=company,dc=cc" (company.cc as the domain name, Company Inc. as the org name)
# Above org info is needed when setting up LDAP

# Sign ldap cert with CA cert
brew-openssl x509 -req -in ldap.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out ldap.crt -days 1825
```

### Run docker-compose to bring up the LDAP service

``` bash
docker-compose up -d
```

docker compose file is under [link](openldap/docker-compose.yaml).

Once the service is up, you should be able to access the LDAP UI at `http://ip:8080`

### Create user account via UI

- Create `Organization Unit` named `developers`
- Create `Group` named `developers`
- Create a `Generic: User Account` named `danielg`

### Run Bookstack app

``` bash
cd ./bookstack
docker-compose up -d
```

docker compose file is under [link](bookstack/docker-compose.yaml). The [.env](bookstack/bookstack-ldap.env) file is the one has all LDAP configurations

Once Bookstack app is running, you should be able to login by using the user account you have created from OpenLDAP.

## More to be explored

- How to do the cert rotation on expiration ?
- How to backup/restore OpenLDAP ?