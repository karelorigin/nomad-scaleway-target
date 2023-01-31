package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad-autoscaler/plugins"
	"github.com/karelorigin/nomad-scaleway-target/plugin"
)

// factory returns a new target plugin instance
func factory(logger hclog.Logger) interface{} {
	return plugin.New(logger)
}

func main() {
	plugins.Serve(factory)
}
