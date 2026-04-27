package main

import "github.com/Mgkusumaputra/pg-factory/cmd"

// Version is set at build time via -ldflags "-X main.Version=<tag>".
var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
