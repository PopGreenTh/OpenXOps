package taxonomy

import (
	"testing"

	"github.com/PopGreenTh/OpenXOps/record/urn"
)

const cloudDoc = `tenant: { code: testtnt, name: Test tenant }
lzs:
  - code: main
    name: Main
    csp: aws
    currency: USD
    management_account: "111111111111"
    ou_paths:
      - code: checkout-prod
        name: Prod
        path: /Workloads/Checkout/Prod
        accounts:
          - { code: chk-prod, name: chk-prod, ref_id: "200000000001" }
      - { code: by-id, name: By id, ref_id: ou-7 }
      - { code: top, name: Top, path: / }
`

func TestCloudURNs(t *testing.T) {
	tax, err := Parse([]byte(cloudDoc))
	if err != nil {
		t.Fatal(err)
	}
	lz := urn.Tenant("testtnt").Child("lzc", "main")
	c := tax.Cloud("MAIN")
	prod := DiscoveredOU{ID: "ou-3", Path: "/Workloads/Checkout/Prod", Name: "Prod"}
	otherProd := DiscoveredOU{ID: "ou-4", Path: "/Workloads/Payments/Prod", Name: "Prod"}

	cases := []struct {
		name string
		got  urn.URN
		want string
	}{
		{"a listed OU, by path", c.OUURN(lz, prod), "urn:tnt:testtnt:lzc:main:org:checkout-prod"},
		{"a listed OU, by ref_id", c.OUURN(lz, DiscoveredOU{ID: "ou-7", Path: "/Anywhere", Name: "Seven"}), "urn:tnt:testtnt:lzc:main:org:by-id"},
		{"an unlisted OU is the slug of its name", c.OUURN(lz, otherProd), "urn:tnt:testtnt:lzc:main:org:prod"},
		{"an unlisted OU with a CamelCase name", c.OUURN(lz, DiscoveredOU{ID: "ou-5", Path: "/Shared Services", Name: "SharedServices"}), "urn:tnt:testtnt:lzc:main:org:shared-services"},
		{"the root, listed", c.OUURN(lz, DiscoveredOU{ID: "r-1", Path: "/", Name: "Root"}), "urn:tnt:testtnt:lzc:main:org:top"},
		{"a listed account", c.AccountURN(lz, prod, DiscoveredAccount{ID: "200000000001", Name: "anything"}), "urn:tnt:testtnt:lzc:main:org:checkout-prod:acc:chk-prod"},
		{"an unlisted account is the slug of its name", c.AccountURN(lz, prod, DiscoveredAccount{ID: "200000000002", Name: "Chk NonProd"}), "urn:tnt:testtnt:lzc:main:org:checkout-prod:acc:chk-non-prod"},
		{"an unlisted account without a name", c.AccountURN(lz, prod, DiscoveredAccount{ID: "200000000003"}), "urn:tnt:testtnt:lzc:main:org:checkout-prod:acc:200000000003"},
		{"an account outside the inventory, listed", c.UnknownAccountURN(lz, "200000000001"), "urn:tnt:testtnt:lzc:main:org:unknown:acc:chk-prod"},
		{"an account outside the inventory, unlisted", c.UnknownAccountURN(lz, "300000000009"), "urn:tnt:testtnt:lzc:main:org:unknown:acc:300000000009"},
	}
	for _, tc := range cases {
		if tc.got.String() != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, tc.got, tc.want)
		}
		if !urn.ValidURN(tc.got.String()) {
			t.Errorf("%s: %s breaks the URN rule", tc.name, tc.got)
		}
	}

	t.Run("the root, unlisted, is root whatever its name", func(t *testing.T) {
		got := Taxonomy{}.Cloud("main").OUURN(lz, DiscoveredOU{ID: "r-1", Path: "/", Name: "Org Root"})
		if got.String() != "urn:tnt:testtnt:lzc:main:org:root" {
			t.Fatal(got)
		}
	})
	t.Run("an LZ the taxonomy does not list uses names only", func(t *testing.T) {
		unknown := tax.Cloud("other")
		got := unknown.AccountURN(urn.Tenant("testtnt").Child("lzc", "unalloc"), prod, DiscoveredAccount{ID: "200000000001", Name: "stray"})
		if got.String() != "urn:tnt:testtnt:lzc:unalloc:org:prod:acc:stray" {
			t.Fatal(got)
		}
	})
}

func TestPathName(t *testing.T) {
	for in, want := range map[string]string{"/": "", "": "", "/Prod": "Prod", "/Workloads/Checkout/Prod": "Prod", "/A/": "A"} {
		if got := PathName(in); got != want {
			t.Errorf("PathName(%q) = %q, want %q", in, got, want)
		}
	}
}
