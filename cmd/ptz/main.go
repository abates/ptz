package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abates/cli"
	"github.com/abates/ptz"
)

var app *cli.Command
var host string
var panSpeed int
var tiltSpeed int
var zoomSpeed int

func init() {
	app = cli.New(
		filepath.Base(os.Args[0]),
		cli.UsageOption("[global options] <command>"),
	)

	app.SetOutput(os.Stderr)

	queryCmd := app.SubCommand("query",
		cli.UsageOption("<remote host>"),
		cli.DescOption("Query camera for current P/T/Z positions"),
		cli.CallbackOption(queryCb),
	)
	queryCmd.Arguments.String(&host, "remote host")

	presetCmd := app.SubCommand("preset",
		cli.UsageOption("<remote host>"),
		cli.DescOption("Generate preset URL for current P/T/Z positions"),
		cli.CallbackOption(presetCb),
	)
	presetCmd.Arguments.String(&host, "remote host")
}

func queryCb(string) error {
	camera, err := ptz.Connect(host)
	if err != nil {
		return err
	}

	info, err := camera.Query()
	if err == nil {
		fmt.Printf("%v\n", info)
	}
	return err
}

func presetCb(string) error {
	camera, err := ptz.Connect(host)
	if err != nil {
		return err
	}

	info, err := camera.Query()
	if err == nil {
		fmt.Printf("Pan/Tilt: http://%s/cgi-bin/ptzctrl.cgi?ptzcmd&abs&%d&%d&%04x&%04x\n", host, panSpeed, tiltSpeed, info.PanPos, info.TiltPos)
		fmt.Printf("    Zoom: http://%s/cgi-bin/ptzctrl.cgi?ptzcmd&zoomto&%d&%04x\n", host, zoomSpeed, info.ZoomPos)
	}
	return err
}

func main() {
	app.Parse(os.Args[1:])
	err := app.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
