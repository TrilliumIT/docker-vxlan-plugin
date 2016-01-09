package vxlan

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	dknet "github.com/docker/go-plugins-helpers/network"
)

const (
	version = "0.1"
)

func main() {

	var flagDebug = cli.BoolFlag{
		Name:  "debug, d",
		Usage: "enable debugging",
	}
	app := cli.NewApp()
	app.Name = "don"
	app.Usage = "Docker vxLan Networking"
	app.Version = version
	app.Flags = []cli.Flag{
		flagDebug,
	}
	app.Action = Run
	app.Run(os.Args)
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	d, err := NewDriver()
	if err != nil {
		panic(err)
	}
	h := dknet.NewHandler(d)
	h.ServeUnix("root", "vxlan")
}
