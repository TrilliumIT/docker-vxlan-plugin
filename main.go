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
		Usage: "Create interfaces before containers are creted, don't destroy interfaces after containers leave. The driver will poll docker every 5 seconds for new networks that may have been created on other hosts.",
	}
	var flagLocalGateway = cli.BoolFlag{
		Name:  "local-gateway",
		Usage: "Allow this host to act as a gatway for containers on the same host. The driver will create a network namespace which will hold gateway macvlan interfaces and route between them. ARP responses will be disabled on the gateway interface, allowing this feature to be enabled on multiple hosts for distributed routing. The localGateway option must be enabled for any networks where you wish to use this feature.",
	}
	var flagGlobalGateway = cli.BoolFlag{
		Name:  "global-gateway",
		Usage: "Allow this host to act as a gatway for containers on any host. The driver will create a network namespace which will hold gateway macvlan interfaces and route between them. ARP responses will be allowed on the gateway interface, so this should only be enabled on one host in the cluster. This can be used in conjunciton with localgateway to provide distributed routing for containers while allowing other devices on the vxlan network to route via the host configured as the global-gateway.",
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
		flagLocalGateway,
		flagGlobalGateway,
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
	d, err := vxlan.NewDriver(ctx.String("scope"), ctx.String("vtepdev"), ctx.Bool("allow-empty"), ctx.Bool("local-gateway"), ctx.Bool("global-gateway"))
	if err != nil {
		panic(err)
	}
	h := network.NewHandler(d)
	h.ServeUnix("root", "vxlan")
}
