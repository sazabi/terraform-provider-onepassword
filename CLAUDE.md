# Terraform Provider for 1Password

Manages 1Password vault structure and vault → group access grants as code, wrapping the [`op` CLI](https://developer.1password.com/docs/cli/). Fills the gap left by the [official provider](https://registry.terraform.io/providers/1Password/onepassword), which exposes vaults read-only and has no resource to create a vault or grant group access.

## Architecture

| Path | Role |
|------|------|
| `main.go` | Provider entry point |
| `internal/provider/` | Provider registration and schema |
| `internal/resources/` | `onepassword_vault` and `onepassword_vault_group_access` resources |
| `docs/` | Generated Terraform Registry documentation |

## Development

```bash
make build   # go build
make test    # unit tests (mocks the op CLI; no live account needed)
make vet     # go vet
```

Local testing with `dev_overrides` — point Terraform at your locally built binary instead of the registry:

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "sazabi/onepassword" = "/path/to/terraform-provider-onepassword"
  }
  direct {}
}
```

Acceptance tests: `make testacc` with `TF_ACC=1`. Run against a dedicated 1Password Business test org — never in CI, never against a production org.

## Conventions

- Shells out to the `op` CLI; never imports a 1Password SDK directly.
- A Service Account can only manage vaults it created. Granting group access to a pre-existing vault requires an admin session with the necessary rights.
- Vault deletion is guarded in two steps: set `delete_protection = false` and apply, then destroy. Never bypass this guard.
