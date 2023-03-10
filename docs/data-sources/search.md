---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ldap_search Data Source - terraform-provider-ldap"
subcategory: ""
description: |-
  Generic LDAP search datasource
---

# ldap_search (Data Source)

Generic LDAP search datasource

## Example Usage

```terraform
data "ldap_search" "example" {
  base_dn = "cn=entry,dc=example,dc=com"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `additional_attributes` (Set of String) Any additional attributes to request, such as constructed or operational attributes
- `base_dn` (String) Base DN to use to search for LDAP objects
- `filter` (String) Filter to search for LDAP objects with
- `scope` (String) Scope to use to search for LDAP objects

### Read-Only

- `id` (String) Datasource identifier
- `results` (List of Map of List of String) List of LDAP objects returned from the search