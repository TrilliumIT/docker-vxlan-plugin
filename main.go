package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/clinta/docker-vxlan-plugin/vxlan"
	"github.com/codegangsta/cli"
	"github.com/clinta/go-plugins-helpers/network"
)

const (
	version = "0.1"
)

func main() {

	var flagDebug = cli.BoolFlag{
		Name:  "debug, d",
		Usage: "enable debugging",
	}
        var flagScope = cli.StringFlag{
                Name: "scope",
                Value: "local",
                Usage: "Scope of the network. local or global.",
        }
	app := cli.NewApp()
	app.Name = "don"
	app.Usage = "Docker vxLan Networking"
	app.Version = version
	app.Flags = []cli.Flag{
		flagDebug,
                flagScope,
	}
	app.Action = Run
	app.Run(os.Args)
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	d, err := vxlan.NewDriver(ctx.String("scope"))
	if err != nil {
		panic(err)
	}
	h := network.NewHandler(d)
	h.ServeUnix("root", "vxlan")
}
