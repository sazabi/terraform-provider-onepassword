resource "onepassword_vault_group_access" "engineering" {
  vault_id    = onepassword_vault.engineering.id
  group       = "engineering@example.com" # group name or UUID
  permissions = ["allow_viewing", "allow_editing"]
}
