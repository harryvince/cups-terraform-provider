package main

import (
	"context"
	"flag"
	"log"

	"github.com/harryvince/terraform-provider-cups/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "Enable debugger support")
	flag.Parse()
	if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/harryvince/cups",
		Debug:   *debug,
	}); err != nil {
		log.Fatal(err)
	}
}
