package provider

import (
	"context"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"os"
	"testing"
)

func TestLDAPObjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create test
			{
				Config: testCreateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "dn", "cn=test,dc=example,dc=com"),
					resource.TestCheckResourceAttr("ldap_object.test", "object_classes.0", "person"),
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.sn.0", "test"),
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.userPassword.0", "password"),
				),
			},
			// Update test
			{
				Config: testUpdateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.sn.0", "test2"),
				),
			},
			// Update object class
			{
				Config: testUpdateObjectClassConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "object_classes.0", "person"),
					resource.TestCheckResourceAttr("ldap_object.test", "object_classes.1", "uidObject"),
				),
			},
			// Update an ignored attribute
			{
				Config: testUpdateIgnoreConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.userPassword.0", "password"),
				),
			},
			// Update an ignored attribute which was changed externally.
			{
				Config:    testUpdateIgnoreConfig,
				PreConfig: testChangePasswordExternally,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "attributes.userPassword.0", "password"),
				),
			},
			// Update DN. Destroy afterwards
			{
				Config: testUpdateDN,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ldap_object.test", "dn", "cn=test2,dc=example,dc=com"),
				),
			},
			{
				Config:  testCreateConfig,
				Destroy: true,
			},
		}},
	)
}

func TestImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test import
			{
				Config:           testImport,
				PreConfig:        testImportPreConfig,
				ImportState:      true,
				ImportStateId:    "cn=importtest,dc=example,dc=com",
				ImportStateCheck: testImportStateCheck,
				ResourceName:     "ldap_object.importtest",
			},
			// Test that ignored attributes are not imported
			{
				Config:             testImportIgnored,
				PreConfig:          testImportIgnoredPreConfig,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      "cn=importtestignore,dc=example,dc=com",
				ResourceName:       "ldap_object.importtestignore",
			},
			{
				Config:       testImportIgnored,
				ResourceName: "ldap_object.importtestignore",
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{ignorePlanCheck()},
				},
			},
		},
	})
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

const testUpdateObjectClassConfig = `
resource "ldap_object" "test" {
	dn = "cn=test,dc=example,dc=com"
	object_classes = ["person", "uidObject"]
	attributes = {
		"cn" = ["test"]
		"sn" = ["test2"]
		"userPassword" = ["password"]
		"uid" = ["test"]
	}
	ignore_changes = ["userPassword"]
}
`

const testUpdateIgnoreConfig = `
resource "ldap_object" "test" {
	dn = "cn=test,dc=example,dc=com"
	object_classes = ["person", "uidObject"]
	attributes = {
		"cn" = ["test"]
		"sn" = ["test2"]
		"userPassword" = ["password2"]
		"uid" = ["test"]
	}
	ignore_changes = ["userPassword"]
}
`

const testUpdateDN = `
resource "ldap_object" "test" {
	dn = "cn=test2,dc=example,dc=com"
	object_classes = ["person", "uidObject"]
	attributes = {
		"cn" = ["test2"]
		"sn" = ["test2"]
		"userPassword" = ["password"]
		"uid" = ["test"]
	}
	ignore_changes = ["userPassword"]
}
`

const testImport = `
resource "ldap_object" "importtest" {
	dn = "cn=importtest,dc=example,dc=com"
	object_classes = ["person"]
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

const testImportIgnored = `
resource "ldap_object" "importtestignore" {
	dn = "cn=importtestignore,dc=example,dc=com"
	object_classes = ["person"]

	attributes = {
		"cn" = ["importtestignore"]
		"sn" = ["test"]
	}

	ignore_changes = ["description"]
}
`

func testImportIgnoredPreConfig() {
	ldapUrl := os.Getenv("LDAP_URL")
	ldapBindDN := os.Getenv("LDAP_BIND_DN")
	ldapBindPassword := os.Getenv("LDAP_BIND_PASSWORD")

	if conn, err := ldap.DialURL(ldapUrl); err != nil {
		return
	} else {
		if err := conn.Bind(ldapBindDN, ldapBindPassword); err != nil {
			return
		}
		r := ldap.NewAddRequest("cn=importtestignore,dc=example,dc=com", []ldap.Control{})
		r.Attribute("objectClass", []string{"person"})
		r.Attribute("sn", []string{"test"})
		r.Attribute("description", []string{"test"})
		if err := conn.Add(r); err != nil {
			return
		}
	}
}

func testImportStateCheck(state []*terraform.InstanceState) error {
	if state[0].ID != "cn=importtest,dc=example,dc=com" {
		return fmt.Errorf("wrong ID found")
	}
	if state[0].Attributes["object_classes.0"] != "person" {
		return fmt.Errorf("can not find object class person")
	}
	if state[0].Attributes["object_classes.#"] != "1" {
		return fmt.Errorf("invalid object classes")
	}
	if state[0].Attributes["attributes.sn.0"] != "test" {
		return fmt.Errorf("invalid sn")
	}
	return nil
}

type IgnorePlanCheck struct{}

func (IgnorePlanCheck) CheckPlan(ctx context.Context, request plancheck.CheckPlanRequest, response *plancheck.CheckPlanResponse) {
	if request.Plan.ResourceChanges[0].Name == "importtestignore" {
		var bA map[string]interface{}
		var aA map[string]interface{}
		if bC, ok := request.Plan.ResourceChanges[0].Change.Before.(map[string]interface{}); ok {
			if a, ok := bC["attributes"].(map[string]interface{}); ok {
				bA = a
			}
		}
		if aC, ok := request.Plan.ResourceChanges[0].Change.After.(map[string]interface{}); ok {
			if a, ok := aC["attributes"].(map[string]interface{}); ok {
				aA = a
			}
		}

		if len(bA) != len(aA) {
			response.Error = fmt.Errorf("attributes would've been changed")
		}
	}
}

func ignorePlanCheck() plancheck.PlanCheck {
	return IgnorePlanCheck{}
}
