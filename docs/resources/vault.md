# onepassword_vault (Resource)

A 1Password vault, managed via the `op` CLI.

Deletion is guarded in two layers: the provider-enforced `delete_protection` attribute (on by default) and, when you add it, the native `lifecycle { prevent_destroy = true }` block. To delete a vault, turn both off — a deliberate two-step for a destructive operation.

## Example Usage

```terraform
resource "onepassword_vault" "engineering" {
  name        = "Engineering Shared"
  description = "Engineering team shared vault"

  delete_protection = true

  lifecycle {
    prevent_destroy = true
  }
}
```

## Schema

### Required

- `name` (String) The vault name.

### Optional

- `description` (String) The vault description.
- `icon` (String) The vault icon name. Write-only on create/update; not refreshed from the API, so it never shows spurious drift.
- `delete_protection` (Boolean) When true (the default), `terraform destroy` is refused for this vault. Set to false in a first apply, then destroy in a second.

### Read-Only

- `id` (String) The vault UUID assigned by 1Password.

## Import

```shell
terraform import onepassword_vault.engineering <vault-uuid>
```
