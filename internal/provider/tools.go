package provider

import (
	"fmt"
	"github.com/go-ldap/ldap/v3"
)

func GetEntry(conn *ldap.Conn, dn string) (ldap.Entry, error) {
	s := ldap.NewSearchRequest(dn, ldap.ScopeBaseObject, 0, 0, 0, false, "(&)", []string{}, []ldap.Control{})

	if result, err := conn.Search(s); err != nil {
		return ldap.Entry{}, err
	} else {
		if len(result.Entries) != 1 {
			return ldap.Entry{}, fmt.Errorf("search returned %d results", len(result.Entries))
		}
		return *result.Entries[0], nil
	}
}
