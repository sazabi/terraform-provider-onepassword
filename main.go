// terraform-provider-onepassword is a Terraform provider that manages
// 1Password vault structure and vault -> group RBAC by wrapping the `op`
// CLI. It fills the gap the official 1Password/onepassword provider leaves:
// that provider manages items only and exposes vaults read-only.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/sazabi/terraform-provider-onepassword/internal/provider"
)

// version is set by goreleaser at build time via -ldflags.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/sazabi/onepassword",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
