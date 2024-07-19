# Terraform LDAP provider

*THIS REPOSITORY IS ARCHIVED*

We're sorry, but we will discontinue this provider. If you'd like to take it further and support it, we're open to adoptions. Thanks!

Terraform provider to manage and read entries in an LDAP directory.

Inspired by [elastic-infra/ldap](https://registry.terraform.io/providers/elastic-infra/ldap/latest), but updated to
Terraform Framework and including ignoring attributes and a data source.

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

In order to run the full suite of Acceptance tests, first make sure to have a running LDAP server. We've included a 
docker-compose file to quickly start a matching test server.

    cd contrib/test-ldap-server
    docker-compose up -d

Then you can set the following environment variables:

- LDAP_NONTLS_URL: The non-TLS enabled URL to the LDAP server
- LDAP_BIND_DN: The bind DN to access the LDAP server
- LDAP_BIND_PASSWORD: The bind password to access the LDAP server
- LDAP_TLS_URL: The TLS enabled URL to access the LDAP server

The URL variables are used to test the non-tls, TLS and STARTTLS features of the provider.

If you use the provided test server, the variables are already set for you.

Afterwards run `make testacc`.
