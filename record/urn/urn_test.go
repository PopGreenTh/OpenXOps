package urn

import "testing"

func TestValidID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"testtnt", true},
		{"acme-retail", true},
		{"a1-b2-c3", true},
		{"0", true},
		{"", false},
		{"Acme", false},
		{"-acme", false},
		{"acme-", false},
		{"acme--retail", false},
		{"acme_retail", false},
		{"acme retail", false},
		{"LZ-MAIN", false},
	}
	for _, tc := range cases {
		if got := ValidID(tc.id); got != tc.want {
			t.Errorf("ValidID(%q) = %v, want %v", tc.id, got, tc.want)
		}
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

func TestOUPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/Workloads/Checkout/Prod", "workloads/checkout/prod"},
		{"/", "root"},
		{"", "root"},
		{"/Shared Services", "shared-services"},
		{"/Workloads/Checkout/NonProd", "workloads/checkout/non-prod"},
		{"/Workloads/!!!", "workloads/unknown"},
	}
	for _, tc := range cases {
		if got := OUPath(tc.in); got != tc.want {
			t.Errorf("OUPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestURN(t *testing.T) {
	acc := Tenant("testtnt").Child("lzc", "main").Child("org", OUPath("/Workloads/Checkout/Prod")).Child("acc", "200000000001")
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
		{"root ou", Tenant("testtnt").Child("lzc", "main").Child("org", OUPath("/")), "urn:tnt:testtnt:lzc:main:org:root"},
		{"account", acc, "urn:tnt:testtnt:lzc:main:org:workloads/checkout/prod:acc:200000000001"},
		{"service", acc.Child("svc", Slug("Amazon Simple Storage Service")),
			"urn:tnt:testtnt:lzc:main:org:workloads/checkout/prod:acc:200000000001:svc:amazon-simple-storage-service"},
	}
	for _, tc := range cases {
		if tc.got.String() != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
