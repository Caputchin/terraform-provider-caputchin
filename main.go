// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider"
)

// version and commit are overridden at build time by GoReleaser via
// -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = "unknown"
)

// Commit returns the build-time commit hash for callers that want to surface
// it in diagnostics (used in user-agent + provider metadata via main.version).
func Commit() string { return commit }

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/caputchin/caputchin",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
