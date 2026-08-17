# onepassword_vault_group_access (Resource)

Grants a 1Password group access to a vault at a set of permissions. Model one resource per (vault, group) pair. Bind it to a directory-synced group by name or UUID.

Requires a 1Password **Business/Teams** account. Granting a group on a pre-existing (imported) vault requires an admin `op signin` session — a Service Account can only manage vaults it created.

## Example Usage

```terraform
resource "onepassword_vault_group_access" "engineering" {
  vault_id    = onepassword_vault.engineering.id
  group       = "engineering@example.com" # group name or UUID
  permissions = ["allow_viewing", "allow_editing"]
}
```

## Schema

### Required

- `vault_id` (String) The vault UUID to grant access to. Changing this forces a new resource.
- `group` (String) The group to grant, by name or UUID. Changing this forces a new resource.
- `permissions` (Set of String) The permission tokens to grant, e.g. `allow_viewing`, `allow_editing`, `allow_managing`.

### Read-Only

- `id` (String) Synthetic id, `<vault_id>:<group>`.

## Import

```shell
terraform import onepassword_vault_group_access.engineering "<vault-uuid>:<group>"
```
