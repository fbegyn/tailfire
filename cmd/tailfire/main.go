package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	"github.com/prometheus/prometheus/discovery/targetgroup"

	"github.com/alecthomas/kong"
	"tailscale.com/client/tailscale"
)

var (
	DefaultSDConfig = SDConfig{
		RefreshInterval: model.Duration(60 * time.Second),
		URL:             "https://api.tailscale.com",
		Tailnet:         "-",
		TagSeparator:    ",",
	}
	failuresCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tailfire_sd_failures_total",
			Help: "Number of tailscale service discovery refresh failures.",
		})
	serviceTargetGroup []*targetgroup.Group
)

func main() {
	tailscale.I_Acknowledge_This_API_Is_Unstable = true

	appCtx := kong.Parse(&cli,
		kong.Name("tailfire"),
		kong.Description("A Prometheus service discovery tool for Tailscale"),
		kong.UsageOnError(),
		kong.Vars{
			"version": "0.2.0",
		},
	)

	err := appCtx.Run(&cli.Globals)
	if err != nil {
		slog.Error("failed to run kong app", "error", err)
		os.Exit(5)
	}
}
