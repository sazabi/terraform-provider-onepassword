package provider

import (
	"reflect"
	"testing"
)

func TestGrantID(t *testing.T) {
	if got := grantID("v1", "engineering@example.com"); got != "v1:engineering@example.com" {
		t.Fatalf("grantID mismatch: %q", got)
	}
}

func TestNormalizePerms_Sorts(t *testing.T) {
	got := normalizePerms([]string{"allow_managing", "allow_editing", "allow_viewing"})
	want := []string{"allow_editing", "allow_managing", "allow_viewing"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePerms mismatch:\n got  %v\n want %v", got, want)
	}
}
