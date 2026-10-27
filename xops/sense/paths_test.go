package sense

import (
	"testing"

	"github.com/PopGreenTh/OpenXOps/aws"
)

func TestPathsAmbiguousSiblingNames(t *testing.T) {
	ous := []aws.OU{
		{ID: "r-1", Name: "Root", ParentID: ""},
		{ID: "ou-1", Name: "Workloads", ParentID: "r-1"},
		{ID: "ou-a", Name: "Prod", ParentID: "ou-1"},
		{ID: "ou-b", Name: "Prod", ParentID: "ou-1"},
	}
	views, findings := Paths(ous)
	if views["ou-a"].path != "/Workloads/Prod" {
		t.Fatalf("ou-a path = %q", views["ou-a"].path)
	}
	if len(findings) != 1 || findings[0]["code"] != "ou-path-ambiguous" {
		t.Fatalf("findings = %v", findings)
	}
}

func TestPathsIDPath(t *testing.T) {
	ous := []aws.OU{
		{ID: "r-1", Name: "Root", ParentID: ""},
		{ID: "ou-1", Name: "Workloads", ParentID: "r-1"},
		{ID: "ou-2", Name: "Checkout", ParentID: "ou-1"},
	}
	views, _ := Paths(ous)
	if views["ou-2"].idPath != "r-1/ou-1/ou-2" {
		t.Fatalf("id_path = %q", views["ou-2"].idPath)
	}
}
