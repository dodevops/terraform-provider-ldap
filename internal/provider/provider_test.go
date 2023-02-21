package provider

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ldap": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	assert.NotEmpty(t, os.Getenv("LDAP_URL"), "Please set LDAP_URL variable")
	assert.NotEmpty(t, os.Getenv("LDAP_BIND_DN"), "Please set LDAP_BIND_DN variable")
	assert.NotEmpty(t, os.Getenv("LDAP_BIND_PASSWORD"), "Please set LDAP_BIND_PASSWORD variable")
}
