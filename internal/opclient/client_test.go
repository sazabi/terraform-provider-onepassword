package opclient

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// newMock returns a Client whose Runner records the args of each call and
// returns the queued outputs in order.
func newMock(account string, outputs ...[]byte) (*Client, *[][]string) {
	var calls [][]string
	i := 0
	c := &Client{
		Account: account,
		Run: func(_ context.Context, args ...string) ([]byte, error) {
			calls = append(calls, args)
			if i < len(outputs) {
				out := outputs[i]
				i++
				return out, nil
			}
			return []byte("{}"), nil
		},
	}
	return c, &calls
}

func TestCreateVault_Args(t *testing.T) {
	c, calls := newMock("my.1password.com", []byte(`{"id":"abc123","name":"Engineering","description":"d"}`))
	v, err := c.CreateVault(context.Background(), "Engineering", "d", "gears")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "abc123" || v.Name != "Engineering" {
		t.Fatalf("unexpected vault: %+v", v)
	}
	want := []string{"--account", "my.1password.com", "vault", "create", "Engineering", "--format", "json", "--description", "d", "--icon", "gears"}
	if got := (*calls)[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("argv mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestCreateVault_NoOptionalFlags(t *testing.T) {
	c, calls := newMock("", []byte(`{"id":"x","name":"n"}`))
	if _, err := c.CreateVault(context.Background(), "n", "", ""); err != nil {
		t.Fatal(err)
	}
	want := []string{"vault", "create", "n", "--format", "json"}
	if got := (*calls)[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("argv mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestGetVault_Parse(t *testing.T) {
	c, _ := newMock("", []byte(`{"id":"v1","name":"Ops","description":"ops vault"}`))
	v, err := c.GetVault(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Ops" || v.Description != "ops vault" {
		t.Fatalf("unexpected: %+v", v)
	}
}

func TestDeleteVault_Args(t *testing.T) {
	c, calls := newMock("acct")
	if err := c.DeleteVault(context.Background(), "v9"); err != nil {
		t.Fatal(err)
	}
	want := []string{"--account", "acct", "vault", "delete", "v9"}
	if got := (*calls)[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("argv mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestGrantVaultGroup_Args(t *testing.T) {
	c, calls := newMock("")
	err := c.GrantVaultGroup(context.Background(), "v1", "engineering@example.com", []string{"allow_viewing", "allow_editing"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"vault", "group", "grant", "--vault", "v1", "--group", "engineering@example.com", "--permissions", "allow_viewing,allow_editing", "--no-input"}
	if got := (*calls)[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("argv mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestRevokeVaultGroup_FullAndScoped(t *testing.T) {
	c, calls := newMock("")
	if err := c.RevokeVaultGroup(context.Background(), "v1", "g", nil); err != nil {
		t.Fatal(err)
	}
	wantFull := []string{"vault", "group", "revoke", "--vault", "v1", "--group", "g", "--no-input"}
	if got := (*calls)[0]; !reflect.DeepEqual(got, wantFull) {
		t.Fatalf("full revoke argv mismatch:\n got  %v\n want %v", got, wantFull)
	}
	if err := c.RevokeVaultGroup(context.Background(), "v1", "g", []string{"allow_managing"}); err != nil {
		t.Fatal(err)
	}
	wantScoped := []string{"vault", "group", "revoke", "--vault", "v1", "--group", "g", "--no-input", "--permissions", "allow_managing"}
	if got := (*calls)[1]; !reflect.DeepEqual(got, wantScoped) {
		t.Fatalf("scoped revoke argv mismatch:\n got  %v\n want %v", got, wantScoped)
	}
}

func TestFindVaultGroup_ByNameAndID(t *testing.T) {
	out := []byte(`[{"id":"grp1","name":"Team Members","permissions":["allow_viewing"]},{"id":"grp2","name":"Owners","permissions":["allow_managing"]}]`)
	// by name (case-insensitive)
	c, _ := newMock("", out)
	g, err := c.FindVaultGroup(context.Background(), "v1", "team members")
	if err != nil {
		t.Fatal(err)
	}
	if g == nil || g.ID != "grp1" {
		t.Fatalf("expected grp1 by name, got %+v", g)
	}
	// by id
	c2, _ := newMock("", out)
	g2, err := c2.FindVaultGroup(context.Background(), "v1", "grp2")
	if err != nil {
		t.Fatal(err)
	}
	if g2 == nil || g2.Name != "Owners" {
		t.Fatalf("expected Owners by id, got %+v", g2)
	}
	// absent
	c3, _ := newMock("", out)
	g3, err := c3.FindVaultGroup(context.Background(), "v1", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if g3 != nil {
		t.Fatalf("expected nil for absent group, got %+v", g3)
	}
}

func TestRunnerErrorPropagates(t *testing.T) {
	c := &Client{Run: func(_ context.Context, _ ...string) ([]byte, error) {
		return nil, errors.New("boom")
	}}
	if _, err := c.GetVault(context.Background(), "v1"); err == nil {
		t.Fatal("expected error to propagate")
	}
}
