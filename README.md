# Terraform Provider for 1Password Vaults & RBAC

A Terraform provider that manages **1Password vault structure** and **vault → group access grants** by wrapping the [`op` CLI](https://developer.1password.com/docs/cli/).

It fills the gap the official [`1Password/onepassword`](https://registry.terraform.io/providers/1Password/onepassword) provider leaves: that provider manages *items* and exposes vaults read-only. It has no resource to create a vault or to grant a group access to one. Those are org-admin operations exposed only through the `op` CLI and the Admin Console — so this provider shells out to `op`.

## Why a provider instead of a shell script

A `null_resource` + `local-exec` wrapper can make a vault "exist," but it is state-blind: it never notices an out-of-band change and cannot show a plan diff. A real provider reconciles declared state against the live vault/grant on every `plan`, so drift is caught and RBAC changes are reviewable before apply.

## Resources

| Resource | Manages |
|---|---|
| `onepassword_vault` | A vault (create/read/update/delete), with two layered delete guards |
| `onepassword_vault_group_access` | One group's access to one vault, at a set of permissions |

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform) >= 1.0
- The [`op` CLI](https://developer.1password.com/docs/cli/get-started/) installed and authenticated wherever the provider runs
- A 1Password **Business/Teams** account for `onepassword_vault_group_access` (group→vault RBAC is not available on individual/family tiers)

## Authentication

The provider is authentication-agnostic: it invokes `op`, which uses whichever credential is present in the environment.

- **`OP_SERVICE_ACCOUNT_TOKEN`** — a [Service Account](https://developer.1password.com/docs/service-accounts/) token, for non-interactive use.
- **`op signin`** — an interactive admin session.

> **Important:** a Service Account can only manage vaults **it created**. Granting a group access to a pre-existing (imported) vault requires an **admin session** with the necessary rights. Choose the credential per operation.

## Usage

```hcl
terraform {
  required_providers {
    onepassword = {
      source  = "sazabi/onepassword"
      version = "~> 0.1"
    }
  }
}

provider "onepassword" {
  # Optional; omit to use op's default account.
  account = "my.1password.com"
}

resource "onepassword_vault" "engineering" {
  name        = "Engineering Shared"
  description = "Engineering team shared vault"

  # Guard 1: provider-enforced, on by default.
  delete_protection = true

  # Guard 2: native Terraform plan-time hard stop.
  lifecycle {
    prevent_destroy = true
  }
}

resource "onepassword_vault_group_access" "engineering" {
  vault_id    = onepassword_vault.engineering.id
  group       = "engineering@example.com" # group name or UUID
  permissions = ["allow_viewing", "allow_editing"]
}
```

## Deleting a vault

Vault deletion is guarded in two layers. To delete a vault: set `delete_protection = false` and apply, remove any `prevent_destroy` lifecycle block, then `terraform destroy`. The two-step is deliberate — vault deletion is destructive.

## Development

```bash
make build   # go build
make test    # unit tests (mock the op CLI; no live account)
make vet
```

### Local testing with dev_overrides

Point Terraform at your locally built binary instead of the registry:

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "sazabi/onepassword" = "/path/to/terraform-provider-onepassword"
  }
  direct {}
}
```

Then run `terraform plan`/`apply` (no `terraform init` needed under dev_overrides).

### Acceptance tests

`make testacc` runs acceptance tests with `TF_ACC=1`. Run them **locally against a dedicated 1Password Business test org** — never in CI and never against a production org.

## License

[Apache 2.0](./LICENSE)
