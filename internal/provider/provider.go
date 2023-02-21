package provider

import (
	"context"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"
)

// Ensure LDAPProvider satisfies various provider interfaces.
var _ provider.Provider = &LDAPProvider{}

// LDAPProvider defines the provider implementation.
type LDAPProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// LDAPProviderModel describes the provider data model.
type LDAPProviderModel struct {
	LDAPURL          types.String `tfsdk:"ldap_url"`
	LDAPBindDN       types.String `tfsdk:"ldap_bind_dn"`
	LDAPBindPassword types.String `tfsdk:"ldap_bind_password"`
}

func (p *LDAPProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ldap"
	resp.Version = p.version
}

func (p *LDAPProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Terraform provider to manage and read entries in an LDAP directory.

Inspired by [elastic-infra/ldap](https://registry.terraform.io/providers/elastic-infra/ldap/latest), but updated to
Terraform Framework and including ignoring attributes and a data source.
`,
		Attributes: map[string]schema.Attribute{
			"ldap_url": schema.StringAttribute{
				MarkdownDescription: "LDAP URL to managed server (can be managed using the environment variable LDAP_URL)",
				Optional:            true,
			},
			"ldap_bind_dn": schema.StringAttribute{
				MarkdownDescription: "Bind DN used to manage directory (can be managed using the environment variable LDAP_BIND_DN)",
				Optional:            true,
			},
			"ldap_bind_password": schema.StringAttribute{
				MarkdownDescription: "Bind password  (can be managed using the environment variable LDAP_BIND_PASSWORD)",
				Optional:            true,
			},
		},
	}
}

func (p *LDAPProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	ldapUrl := os.Getenv("LDAP_URL")
	ldapBindDN := os.Getenv("LDAP_BIND_DN")
	ldapBindPassword := os.Getenv("LDAP_BIND_PASSWORD")

	var data LDAPProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.LDAPURL.ValueString() != "" {
		ldapUrl = data.LDAPURL.ValueString()
	}

	if data.LDAPBindDN.ValueString() != "" {
		ldapBindDN = data.LDAPBindDN.ValueString()
	}

	if data.LDAPBindPassword.ValueString() != "" {
		ldapBindPassword = data.LDAPBindPassword.ValueString()
	}

	if ldapUrl == "" {
		resp.Diagnostics.AddError(
			"No LDAP url specified",
			"Configure the ldap_url attribute or LDAP_URL environment variable for the provider",
		)
		return
	}

	if ldapBindDN == "" {
		resp.Diagnostics.AddError(
			"No LDAP bind dn specified",
			"Configure the ldap_bind_dn attribute or LDAP_BIND_DN environment variable for the provider",
		)
		return
	}

	if ldapBindPassword == "" {
		resp.Diagnostics.AddError(
			"No LDAP bind password specified",
			"Configure the ldap_bind_password attribute or LDAP_BIND_PASSWORD environment variable for the provider",
		)
		return
	}

	if conn, err := ldap.DialURL(ldapUrl); err != nil {
		resp.Diagnostics.AddError(
			"Can't connect to LDAP server",
			fmt.Sprintf("Error connecting to LDAP server: %s", err),
		)
		return
	} else {
		if err := conn.Bind(ldapBindDN, ldapBindPassword); err != nil {
			resp.Diagnostics.AddError(
				"Can't bind to LDAP server",
				fmt.Sprintf("Error binding to LDAP server: %s", err),
			)
		}
		resp.DataSourceData = conn
		resp.ResourceData = conn
	}
}

func (p *LDAPProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewLDAPObjectResource,
	}
}

func (p *LDAPProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewLDAPObjectDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LDAPProvider{
			version: version,
		}
	}
}
