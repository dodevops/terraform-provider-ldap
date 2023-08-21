package provider

import (
	"context"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/thoas/go-funk"
)

// GetEntry returns a specific entry and is a shortcut around the search function.
func GetEntry(conn *ldap.Conn, dn string, attrs ...string) (ldap.Entry, error) {
	s := ldap.NewSearchRequest(dn, ldap.ScopeBaseObject, 0, 0, 0, false, "(&)", attrs, []ldap.Control{})

	if result, err := conn.Search(s); err != nil {
		return ldap.Entry{}, err
	} else {
		if len(result.Entries) != 1 {
			return ldap.Entry{}, fmt.Errorf("search returned %d results", len(result.Entries))
		}
		return *result.Entries[0], nil
	}
}

// ToLDIF converts the given ldap entry into an LDIF representation.
func ToLDIF(entry interface{}) string {
	if l, err := ldif.ToLDIF(entry); err == nil {
		if m, err := ldif.Marshal(l); err == nil {
			return m
		}
	}
	return ""
}

// MaskAttributes searches attributes of an LDAP entry for sensitive data and masks the values.
func MaskAttributes(ctx context.Context, attributes map[string][]string) context.Context {
	for attributeType, values := range attributes {
		if attributeType == "userPassword" {
			funk.ForEach(values, func(value string) {
				ctx = tflog.MaskLogStrings(ctx, value)
			})
		}
	}
	return ctx
}

// MaskAttributesFromArray is a MaskAttributes adapter for ldap.EntryAttribute-Arrays.
func MaskAttributesFromArray(ctx context.Context, attributes []*ldap.EntryAttribute) context.Context {
	var attributesHash = funk.Reduce(
		attributes,
		func(acc map[string][]string, a *ldap.EntryAttribute) map[string][]string {
			acc[a.Name] = a.Values
			return acc
		},
		make(map[string][]string),
	)
	if h, ok := attributesHash.(map[string][]string); !ok {
		return ctx
	} else {
		return MaskAttributes(ctx, h)
	}
}
