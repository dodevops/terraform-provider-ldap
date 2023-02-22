package provider

import (
	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

func TestLDAPObjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testCreateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "dn", "cn=test,dc=example,dc=com"),
					resource.TestCheckResourceAttr("ldap_object.test", "object_classes.0", "person"),
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.sn.0", "test"),
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.userPassword.0", "password"),
				),
			},
			{
				Config: testUpdateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.sn.0", "test2"),
				),
			},
			{
				Config: testUpdateIgnoreConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.userPassword.0", "password"),
				),
			},
			{
				Config:    testUpdateIgnoreConfig,
				PreConfig: testChangePasswordExternally,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.userPassword.0", "password"),
				),
			},
			{
				Config:        testImport,
				PreConfig:     testImportPreConfig,
				ImportState:   true,
				ImportStateId: "cn=importtest,dc=example,dc=com",
				ResourceName:  "ldap_object.importtest",
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.importtest", "attributes.sn.0", "test"),
				),
			},
		}},
	)
}

func testChangePasswordExternally() {
	ldapUrl := os.Getenv("LDAP_URL")
	ldapBindDN := os.Getenv("LDAP_BIND_DN")
	ldapBindPassword := os.Getenv("LDAP_BIND_PASSWORD")

	if conn, err := ldap.DialURL(ldapUrl); err != nil {
		return
	} else {
		if err := conn.Bind(ldapBindDN, ldapBindPassword); err != nil {
			return
		}
		r := ldap.NewModifyRequest("cn=test,dc=example,dc=com", []ldap.Control{})
		r.Replace("userPassword", []string{"password2"})
		if err := conn.Modify(r); err != nil {
			return
		}
	}
}

const testCreateConfig = `
resource "ldap_object" "test" {
	dn = "cn=test,dc=example,dc=com"
	object_classes = ["person"]
	attributes = {
		"cn" = ["test"]
		"sn" = ["test"]
		"userPassword" = ["password"]
	}
	ignore_changes = ["userPassword"]
}
`

const testUpdateConfig = `
resource "ldap_object" "test" {
	dn = "cn=test,dc=example,dc=com"
	object_classes = ["person"]
	attributes = {
		"cn" = ["test"]
		"sn" = ["test2"]
		"userPassword" = ["password"]
	}
	ignore_changes = ["userPassword"]
}
`

const testUpdateIgnoreConfig = `
resource "ldap_object" "test" {
	dn = "cn=test,dc=example,dc=com"
	object_classes = ["person"]
	attributes = {
		"cn" = ["test"]
		"sn" = ["test2"]
		"userPassword" = ["password2"]
	}
	ignore_changes = ["userPassword"]
}
`

const testImport = `
resource "ldap_object" "importtest" {
}
`

func testImportPreConfig() {
	ldapUrl := os.Getenv("LDAP_URL")
	ldapBindDN := os.Getenv("LDAP_BIND_DN")
	ldapBindPassword := os.Getenv("LDAP_BIND_PASSWORD")

	if conn, err := ldap.DialURL(ldapUrl); err != nil {
		return
	} else {
		if err := conn.Bind(ldapBindDN, ldapBindPassword); err != nil {
			return
		}
		r := ldap.NewAddRequest("cn=importtest,dc=example,dc=com", []ldap.Control{})
		r.Attribute("objectClass", []string{"person"})
		r.Attribute("sn", []string{"test"})
		if err := conn.Add(r); err != nil {
			return
		}
	}
}
