# Terraform LDAP provider

## Using the provider

Add the following Terraform code to start using the provider:

```terraform
terraform {
  required_providers {
    ldap = {
      source  = "dodevops/ldap"
      version = "~> 1.0"
    }
  }
}

provider "ldap" {
  ldap_url      = "ldap://localhost:389"
  bind_user     = "cn=admin,dc=example,dc=com"
  bind_password = "admin"
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, first make sure to have a running LDAP server and set the environment
variables LDAP_URL, LDAP_BIND_DN and LDAP_BIND_PASSWORD accordingly. We've included a docker-compose file to quickly
start a matching test server.

    cd contrib/test-ldap-server
    docker-compose up -d

Afterwards run `make testacc`.
