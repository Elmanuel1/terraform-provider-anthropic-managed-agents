package main

import (
	"context"
	"flag"
	"log"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// providerAddr is set at build time via -ldflags "-X main.providerAddr=..."
var providerAddr = "registry.terraform.io/Elmanuel1/anthropic"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: providerAddr,
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
