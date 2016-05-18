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
		Name:  "allow-empty",
		Usage: "Create interfaces before containers are creted, don't destroy interfaces after containers leave",
	}
	var flagGlobalGateway = cli.BoolFlag{
		Name:  "global-gateway",
		Usage: "Allow Assigning the gateway address to the bridge interface, even on global networks. The globalGateway option must also be specified in the network options for any networks you want global-gateway to be active on.",
	}
	var flagBlockGatewayArp = cli.BoolFlag{
		Name:  "block-gateway-arp",
		Usage: "Use arptables to block arp requests for the gateway from traversing the vxlan overlay. Necessary for enabling distributed routing The blockGatewayArp option must also be specified in the network options for any networks you want to block arps on.",
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
		flagGlobalGateway,
		flagBlockGatewayArp,
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
	d, err := vxlan.NewDriver(ctx.String("scope"), ctx.String("vtepdev"), ctx.Bool("allow-empty"), ctx.Bool("global-gateway"), ctx.Bool("block-gateway-arp"))
	if err != nil {
		panic(err)
	}
	h := network.NewHandler(d)
	h.ServeUnix("root", "vxlan")
}
