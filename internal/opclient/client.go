// Package opclient is a thin wrapper over the 1Password `op` CLI for the
// org-admin operations the official 1Password Terraform provider does not
// expose: vault create/read/update/delete and vault group grant/revoke.
//
// The provider shells out to `op` because there is no Go SDK or documented
// management REST API for these operations; `op` is the only management
// surface. The Runner field is injectable so unit tests can assert argv and
// simulate output without a live 1Password account.
package opclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes the `op` CLI with the given args and returns stdout.
// On failure it returns an error that includes stderr.
type Runner func(ctx context.Context, args ...string) ([]byte, error)

// Client talks to a single 1Password account via the `op` CLI.
type Client struct {
	// Account is the optional `--account` value (e.g. "my.1password.com").
	// When empty, `op` uses whatever session/token is present in the env.
	Account string
	// Run executes op. Defaults to execRunner; overridden in tests.
	Run Runner
}

// New returns a Client that shells out to the real `op` binary.
func New(account string) *Client {
	return &Client{Account: account, Run: execRunner}
}

// execRunner is the default Runner: it invokes the `op` binary on PATH.
func execRunner(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "op", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("op %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// args prepends the account flag when set.
func (c *Client) args(a ...string) []string {
	if c.Account != "" {
		return append([]string{"--account", c.Account}, a...)
	}
	return a
}

// Vault is the subset of `op vault get` output this provider models.
type Vault struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// VaultGroup is one group's access to a vault, from `op vault group list`.
type VaultGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// CreateVault runs `op vault create` and returns the created vault.
func (c *Client) CreateVault(ctx context.Context, name, description, icon string) (*Vault, error) {
	a := []string{"vault", "create", name, "--format", "json"}
	if description != "" {
		a = append(a, "--description", description)
	}
	if icon != "" {
		a = append(a, "--icon", icon)
	}
	out, err := c.Run(ctx, c.args(a...)...)
	if err != nil {
		return nil, err
	}
	var v Vault
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("parse vault create output: %w", err)
	}
	return &v, nil
}

// GetVault runs `op vault get`.
func (c *Client) GetVault(ctx context.Context, id string) (*Vault, error) {
	out, err := c.Run(ctx, c.args("vault", "get", id, "--format", "json")...)
	if err != nil {
		return nil, err
	}
	var v Vault
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("parse vault get output: %w", err)
	}
	return &v, nil
}

// EditVault runs `op vault edit` for the mutable fields.
func (c *Client) EditVault(ctx context.Context, id, name, description, icon string) error {
	a := []string{"vault", "edit", id}
	if name != "" {
		a = append(a, "--name", name)
	}
	if description != "" {
		a = append(a, "--description", description)
	}
	if icon != "" {
		a = append(a, "--icon", icon)
	}
	_, err := c.Run(ctx, c.args(a...)...)
	return err
}

// DeleteVault runs `op vault delete`.
func (c *Client) DeleteVault(ctx context.Context, id string) error {
	_, err := c.Run(ctx, c.args("vault", "delete", id)...)
	return err
}

// ListVaultGroups runs `op vault group list` and returns all group grants.
func (c *Client) ListVaultGroups(ctx context.Context, vaultID string) ([]VaultGroup, error) {
	out, err := c.Run(ctx, c.args("vault", "group", "list", vaultID, "--format", "json")...)
	if err != nil {
		return nil, err
	}
	var groups []VaultGroup
	if err := json.Unmarshal(out, &groups); err != nil {
		return nil, fmt.Errorf("parse vault group list output: %w", err)
	}
	return groups, nil
}

// FindVaultGroup returns the grant for a group identified by UUID or name,
// or (nil, nil) when the group has no access to the vault.
func (c *Client) FindVaultGroup(ctx context.Context, vaultID, group string) (*VaultGroup, error) {
	groups, err := c.ListVaultGroups(ctx, vaultID)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].ID == group || strings.EqualFold(groups[i].Name, group) {
			return &groups[i], nil
		}
	}
	return nil, nil
}

// GrantVaultGroup runs `op vault group grant` with an explicit permission set.
func (c *Client) GrantVaultGroup(ctx context.Context, vaultID, group string, permissions []string) error {
	_, err := c.Run(ctx, c.args(
		"vault", "group", "grant",
		"--vault", vaultID,
		"--group", group,
		"--permissions", strings.Join(permissions, ","),
		"--no-input",
	)...)
	return err
}

// RevokeVaultGroup runs `op vault group revoke`, removing the group's access.
// When permissions is empty, all of the group's access to the vault is revoked.
func (c *Client) RevokeVaultGroup(ctx context.Context, vaultID, group string, permissions []string) error {
	a := []string{"vault", "group", "revoke", "--vault", vaultID, "--group", group, "--no-input"}
	if len(permissions) > 0 {
		a = append(a, "--permissions", strings.Join(permissions, ","))
	}
	_, err := c.Run(ctx, c.args(a...)...)
	return err
}
