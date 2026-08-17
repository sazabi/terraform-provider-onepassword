# onepassword Provider

Manage 1Password **vault structure** and **vault → group access grants** with Terraform, by wrapping the [`op` CLI](https://developer.1password.com/docs/cli/).

The official `1Password/onepassword` provider manages *items* and exposes vaults read-only. This provider adds the two resources it lacks: creating/managing a vault, and granting a group access to a vault.

## Authentication

The provider invokes `op`, which resolves whichever credential is present in the environment:

- `OP_SERVICE_ACCOUNT_TOKEN` — a Service Account token (non-interactive).
- An interactive `op signin` session.

A Service Account can only manage vaults it created; granting a group access to a pre-existing vault requires an admin session. `onepassword_vault_group_access` requires a Business/Teams account.

## Example Usage

```terraform
terraform {
  required_providers {
    onepassword = {
      source  = "sazabi/onepassword"
      version = "~> 0.1"
    }
  }
}

provider "onepassword" {
  account = "my.1password.com"
}
```

## Schema

### Optional

- `account` (String) The 1Password account to target (e.g. `my.1password.com`), passed as `op --account`. Omit to let `op` use its default/only account.
