package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"
	"strings"
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
	LDAPURL               types.String `tfsdk:"ldap_url"`
	LDAPBindDN            types.String `tfsdk:"ldap_bind_dn"`
	LDAPBindPassword      types.String `tfsdk:"ldap_bind_password"`
	LDAPTLSInsecureVerify types.Bool   `tfsdk:"ldap_tls_insecure_verify"`
	LDAPTLSUseStartTLS    types.Bool   `tfsdk:"ldap_tls_use_starttls"`
}

func (p *LDAPProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ldap"
	resp.Version = p.version
}

func (p *LDAPProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Terraform provider to manage and read entries in an LDAP directory.

Inspired by [elastic-infra/ldap](https://registry.terraform.io/providers/elastic-infra/ldap/latest), but updated to
Terraform Framework and including ignoring attributes and a data source.

All provider options can be set by the respective environment variables as well.
`,
		Attributes: map[string]schema.Attribute{
			"ldap_url": schema.StringAttribute{
				MarkdownDescription: "LDAP URL to managed server (`LDAP_URL`)",
				Optional:            true,
			},
			"ldap_bind_dn": schema.StringAttribute{
				MarkdownDescription: "Bind DN used to manage directory (`LDAP_BIND_DN`)",
				Optional:            true,
			},
			"ldap_bind_password": schema.StringAttribute{
				MarkdownDescription: "Bind password (`LDAP_BIND_PASSWORD`)",
				Optional:            true,
			},
			"ldap_tls_insecure_verify": schema.BoolAttribute{
				MarkdownDescription: "Whether to skip certificate verification (`LDAP_TLS_INSECURE_VERIFY`)",
				Optional:            true,
			},
			"ldap_tls_use_starttls": schema.BoolAttribute{
				MarkdownDescription: "Whether to connect using STARTTLS (`LDAP_TLS_USE_STARTTLS`)",
				Optional:            true,
			},
		},
	}
}

func (p *LDAPProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	ldapUrl := os.Getenv("LDAP_URL")
	ldapBindDN := os.Getenv("LDAP_BIND_DN")
	ldapBindPassword := os.Getenv("LDAP_BIND_PASSWORD")
	ldapTLSInsecureVerify := false
	if v := os.Getenv("LDAP_TLS_INSECURE_VERIFY"); v != "" {
		ldapTLSInsecureVerify = strings.ToUpper(v) == "TRUE"
	}

	ldapTLSUseStartTLS := false
	if v := os.Getenv("LDAP_TLS_USE_STARTTLS"); v != "" {
		ldapTLSUseStartTLS = strings.ToUpper(v) == "TRUE"
	}

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

	if !data.LDAPTLSInsecureVerify.IsNull() {
		ldapTLSInsecureVerify = data.LDAPTLSInsecureVerify.ValueBool()
	}

	if !data.LDAPTLSUseStartTLS.IsNull() {
		ldapTLSUseStartTLS = data.LDAPTLSUseStartTLS.ValueBool()
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

	var o []ldap.DialOpt

	if ldapTLSInsecureVerify {
		o = append(o, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	}

	if conn, err := ldap.DialURL(ldapUrl, o...); err != nil {
		resp.Diagnostics.AddError(
			"Can't connect to LDAP server",
			fmt.Sprintf("Error connecting to LDAP server: %s", err),
		)
		return
	} else {
		if ldapTLSUseStartTLS {
			c := tls.Config{}
			if ldapTLSInsecureVerify {
				c.InsecureSkipVerify = true
			}
			if err := conn.StartTLS(&c); err != nil {
				resp.Diagnostics.AddError(
					"Can't start TLS",
					fmt.Sprintf("Error starting TLS: %s", err),
				)
				return
			}
		}
		if err := conn.Bind(ldapBindDN, ldapBindPassword); err != nil {
			resp.Diagnostics.AddError(
				"Can't bind to LDAP server",
				fmt.Sprintf("Error binding to LDAP server: %s", err),
			)
			return
		}
		resp.DataSourceData = conn
		resp.ResourceData = conn
	}
}

func (p *LDAPProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewLDAPObjectResource,
	}
}

func (p *LDAPProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewLDAPObjectDataSource,
		NewLDAPSearchDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LDAPProvider{
			version: version,
		}
	}
}
