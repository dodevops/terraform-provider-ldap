package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestLDAPObjectDatasource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.ldap_object.test", "dn", "dc=example,dc=com"),
					resource.TestCheckResourceAttr("data.ldap_object.test", "object_classes.#", "3"),
					resource.TestCheckResourceAttr("data.ldap_object.test", "attributes.dc.0", "example"),
					resource.TestCheckResourceAttr("data.ldap_object.test", "attributes.creatorsName.0", "cn=admin,dc=example,dc=com"),
				),
			},
		},
	})
}

const testDataSource = `
data "ldap_object" "test" {
	dn = "dc=example,dc=com"
	additional_attributes = ["creatorsName"]
}`
