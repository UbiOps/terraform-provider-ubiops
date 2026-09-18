// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"terraform-provider-ubiops/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// version and related build info are set by goreleaser for the compiled binary.
	version string = "dev"

	// See https://goreleaser.com/cookbooks/using-main.version/ for what else it can inject.
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/ubiops/ubiops",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
