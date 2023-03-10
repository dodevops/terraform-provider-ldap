package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestLDAPSearchDatasource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testSearchDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.ldap_search.test", "base_dn", "dc=example,dc=com"),
					resource.TestCheckResourceAttr("data.ldap_search.test", "results.0.dc.0", "example"),
					resource.TestCheckResourceAttr("data.ldap_search.test", "results.0.creatorsName.0", "cn=admin,dc=example,dc=com"),
				),
			},
		},
	})
}

const testSearchDataSource = `
data "ldap_search" "test" {
	base_dn = "dc=example,dc=com"
	additional_attributes = ["creatorsName"]
}`
