package urn

import (
	"strings"
	"testing"
)

func TestValidCode(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"PRJ_Shop", true},
		{"a.b", true},
		{"x@y=z+1", true},
		{"testtnt", true},
		{"acme-retail", true},
		{"-ab", true}, // leading and trailing dashes are allowed
		{"abc", true},
		{strings.Repeat("a", 50), true},
		{"", false},
		{"ab", false},
		{"a", false},
		{strings.Repeat("a", 51), false},
		{"a/b", false},
		{"a b", false},
		{"abc:def", false},
		{"caf\u00e9", false},
	}
	for _, tc := range cases {
		if got := ValidCode(tc.code); got != tc.want {
			t.Errorf("ValidCode(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestValidURN(t *testing.T) {
	cases := []struct {
		urn  string
		want bool
	}{
		{"urn:tnt:cvd", true},
		{"urn:tnt:testtnt:lzc:main:org:non-prod:acc:chk-nonprod:svc:amazon-elastic-compute-cloud-compute", true},
		{"urn:tnt:testtnt:prj:prj_shop", true},
		{"urn:tnt:a.b:x@y=z+1", true},
		{"urn:" + strings.Repeat("a", 246), true}, // 250 characters
		{"", false},
		{"urn:tnt:cv", false}, // 10 characters
		{strings.Repeat("a", 10), false},
		{"urn:" + strings.Repeat("a", 247), false}, // 251 characters
		{strings.Repeat("a", 251), false},
		{"urn:tnt:Testtnt", false}, // an uppercase letter
		{"urn:tnt:test tnt", false},
		{"urn:tnt:testtnt:prj:caf\u00e9", false},
	}
	for _, tc := range cases {
		if got := ValidURN(tc.urn); got != tc.want {
			t.Errorf("ValidURN(%q) = %v, want %v", tc.urn, got, tc.want)
		}
	}
	if n := len("urn:" + strings.Repeat("a", 246)); n != 250 {
		t.Fatalf("test URN is %d characters, want 250", n)
	}
}

func TestSlug(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// The table of D-053.
		{"SharedServices", "shared-services"},
		{"NonProd", "non-prod"},
		{"AWSConfig", "aws-config"},
		{"AmazonEC2", "amazon-ec2"},
		{"Ec2Prod", "ec2-prod"},
		{"Hub2", "hub2"},
		{"Spoke-SIT", "spoke-sit"},
		{"Amazon Elastic Compute Cloud - Compute", "amazon-elastic-compute-cloud-compute"},
		{"---", "unknown"},
		// More.
		{"Checkout", "checkout"},
		{"  --Prod__EU--  ", "prod-eu"},
		{"AWS Key Management Service", "aws-key-management-service"},
		{"AmazonCloudWatch", "amazon-cloud-watch"},
		{"AmazonS3", "amazon-s3"},
		{"S3Bucket", "s3-bucket"},
		{"already-a-slug", "already-a-slug"},
		{"ABC", "abc"},
		{"abc", "abc"},
		{"A", "a"},
		{"HTTPServer", "http-server"},
		{"Café", "caf"},
		{"", "unknown"},
		{"日本", "unknown"},
	}
	for _, tc := range cases {
		if got := Slug(tc.in); got != tc.want {
			t.Errorf("Slug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestURNIdsAreLowercased(t *testing.T) {
	got := Tenant("TestTnt").Child("prj", "PRJ_Shop").Child("app", "Chk.v2")
	if want := "urn:tnt:testtnt:prj:prj_shop:app:chk.v2"; got.String() != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !ValidURN(got.String()) {
		t.Fatalf("%q breaks the URN rule", got)
	}
}

func TestURN(t *testing.T) {
	acc := Tenant("testtnt").Child("lzc", "main").Child("org", "checkout-prod").Child("acc", "chk-prod")
	cases := []struct {
		name string
		got  URN
		want string
	}{
		{"tenant", Tenant("testtnt"), "urn:tnt:testtnt"},
		{"business unit", Tenant("testtnt").Child("bzu", "retail"), "urn:tnt:testtnt:bzu:retail"},
		{"cost center", Tenant("testtnt").Child("bzu", "retail").Child("cct", "ecom"), "urn:tnt:testtnt:bzu:retail:cct:ecom"},
		{"environment", Tenant("testtnt").Child("prj", "unalloc").Child("app", "unalloc").Child("env", "unalloc"),
			"urn:tnt:testtnt:prj:unalloc:app:unalloc:env:unalloc"},
		{"root ou", Tenant("testtnt").Child("lzc", "main").Child("org", "root"), "urn:tnt:testtnt:lzc:main:org:root"},
		{"account", acc, "urn:tnt:testtnt:lzc:main:org:checkout-prod:acc:chk-prod"},
		{"service", acc.Child("svc", Slug("Amazon Simple Storage Service")),
			"urn:tnt:testtnt:lzc:main:org:checkout-prod:acc:chk-prod:svc:amazon-simple-storage-service"},
	}
	for _, tc := range cases {
		if tc.got.String() != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
