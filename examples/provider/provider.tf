provider "ldap" {
  ldap_url           = "ldaps://ldap.company.com"
  ldap_bind_dn       = "cn=admin,dc=example,dc=com"
  ldap_bind_password = "verysecret"
}
