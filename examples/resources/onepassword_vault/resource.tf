resource "onepassword_vault" "engineering" {
  name        = "Engineering Shared"
  description = "Engineering team shared vault"

  delete_protection = true

  lifecycle {
    prevent_destroy = true
  }
}
