terraform {
  required_providers {
    onepassword = {
      source  = "sazabi/onepassword"
      version = "~> 0.1"
    }
  }
}

provider "onepassword" {
  # Optional. Omit to use op's default/only account.
  account = "my.1password.com"
}
