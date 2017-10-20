package main

import (
	"github.com/DavidWittman/packer-post-processor-tarball/tarball"
	"github.com/hashicorp/packer/packer/plugin"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterPostProcessor(new(tarball.PostProcessor))
	server.Serve()
}
