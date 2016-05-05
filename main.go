package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/clinta/docker-vxlan-plugin/vxlan"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/codegangsta/cli"
)

const (
	version = "0.6"
)

func main() {

	var flagDebug = cli.BoolFlag{
		Name:  "debug, d",
		Usage: "Enable debugging.",
	}
	var flagScope = cli.StringFlag{
		Name:  "scope",
		Value: "local",
		Usage: "Scope of the network. local or global.",
	}
	var flagVtepDev = cli.StringFlag{
		Name:  "vtepdev",
		Value: "",
		Usage: "VTEP device.",
	}
	var flagAllowEmpty = cli.BoolFlag{
		Name:  "allow_empty",
		Usage: "Create interfaces before containers are creted, don't destroy interfaces after containers leave",
	}
	app := cli.NewApp()
	app.Name = "don"
	app.Usage = "Docker vxLan Networking"
	app.Version = version
	app.Flags = []cli.Flag{
		flagDebug,
		flagScope,
		flagVtepDev,
		flagAllowEmpty,
	}
	app.Action = Run
	app.Run(os.Args)
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	log.SetFormatter(&log.TextFormatter{
		ForceColors: false,
		DisableColors: true,
		DisableTimestamp: false,
		FullTimestamp: true,
	})
	d, err := vxlan.NewDriver(ctx.String("scope"), ctx.String("vtepdev"), ctx.Bool("allow_empty"))
	if err != nil {
		panic(err)
	}
	h := network.NewHandler(d)
	h.ServeUnix("root", "vxlan")
}
