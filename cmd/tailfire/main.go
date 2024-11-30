package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"

	"github.com/prometheus/prometheus/discovery/targetgroup"

	"net/http"

	"tailscale.com/client/tailscale"
)

const (
	tsLabel                                = model.MetaLabelPrefix + "tailscale_"
	deviceLabel                            = tsLabel + "device_"
	tsLabelDeviceAddresses                 = deviceLabel + "addresses"
	tsLabelDeviceID                        = deviceLabel + "id"
	tsLabelDeviceName                      = deviceLabel + "name"
	tsLabelDeviceHostname                  = deviceLabel + "hostname"
	tsLabelDeviceUser                      = deviceLabel + "user"
	tsLabelDeviceClientVersion             = deviceLabel + "client_version"
	tsLabelDeviceOS                        = deviceLabel + "os"
	tsLabelDeviceUpdateAvailable           = deviceLabel + "update_available"
	tsLabelDeviceAuthorized                = deviceLabel + "authorized"
	tsLabelDeviceIsExternal                = deviceLabel + "is_external"
	tsLabelDeviceKeyExpiryDisabled         = deviceLabel + "key_expiry_disabled"
	tsLabelDeviceBlocksIncomingConnections = deviceLabel + "blocks_incoming_connections"
	tsLabelDeviceEnabledRoutes             = deviceLabel + "enabled_routes"
	tsLabelDeviceAdvertisedRoutes          = deviceLabel + "advertised_routes"
	tsLabelDeviceTags                      = deviceLabel + "tags"
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
)

func main() {
	tailscale.I_Acknowledge_This_API_Is_Unstable = true

	data, _ := os.ReadFile("./config.yaml")
	cfg := SDConfig{}
	yaml.Unmarshal(data, &cfg)

	d, err := NewDiscovery(&cfg, nil)
	if err != nil {
		slog.Error("failed to create discovery", "error", err)
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /prometheus/targets", HandlePromSDRequest(d))

	slog.Info("starting up webserver on http://localhost:8080")
	slog.Info("curl -vL http://localhost:8080/prometheus/targets")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("web server encountered an error", "error", err)
	} else {
		slog.Info("web server shut down cleanly")
	}
}

func HandlePromSDRequest(d *Discovery) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("received request to return Tailnet Prometheus targets")
		targetGroup, err := d.refresh(r.Context())
		if err != nil {
			slog.Error("an error occured during discovery refresh", "error", err)
			w.Header().Add("Content-Type", "application/plain")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("an error occured during discovery refresh"))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(targetGroup)
	})
}

type SDConfig struct {
	RefreshInterval model.Duration `yaml:"refresh_interval,omitempty"`
	Port            int            `yaml:"port"`
	TagSeparator    string         `yaml:"tag_separator,omitempty"`
	Tailnet         string         `yaml:"tailnet"`
	URL             string         `yaml:"api_url"`
	APIToken        string         `yaml:"api_token,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *SDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultSDConfig
	type plain SDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if c.URL == "" {
		return fmt.Errorf("URL is missing")
	}
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be 'http' or 'https'")
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("host is missing in URL")
	}
	if c.APIToken == "" {
		return fmt.Errorf("API token is missing")
	}
	return nil
}

type Discovery struct {
	tagSeparator string
	url          string
	client       *tailscale.Client
}

func NewDiscovery(conf *SDConfig, logger *slog.Logger) (*Discovery, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if conf.URL == "" {
		conf.URL = "https://api.tailscale.com"
	}

	tsClient := tailscale.NewClient(
		conf.Tailnet,
		tailscale.APIKey(conf.APIToken),
	)
	tsClient.BaseURL = conf.URL

	d := &Discovery{
		client:       tsClient,
		url:          conf.URL,
		tagSeparator: conf.TagSeparator,
	}
	return d, nil
}

func (d *Discovery) refresh(ctx context.Context) (list []*targetgroup.Group, err error) {
	tsDevices, err := d.client.Devices(ctx, tailscale.DeviceDefaultFields)
	if err != nil {
		failuresCount.Inc()
		return nil, err
	}

	for _, device := range tsDevices {
		// Each device forms it's own group since each device has unique labels
		tg := &targetgroup.Group{
			// Use a pseudo-URL as source.
			Source: d.url,
		}

		// labelset to hold the actual metadata labels
		labels := model.LabelSet{
			tsLabelDeviceID:                        model.LabelValue(device.DeviceID),
			tsLabelDeviceOS:                        model.LabelValue(device.OS),
			tsLabelDeviceClientVersion:             model.LabelValue(device.ClientVersion),
			tsLabelDeviceName:                      model.LabelValue(device.Name),
			tsLabelDeviceHostname:                  model.LabelValue(device.Hostname),
			tsLabelDeviceUser:                      model.LabelValue(device.User),
			tsLabelDeviceIsExternal:                model.LabelValue(strconv.FormatBool(device.IsExternal)),
			tsLabelDeviceAuthorized:                model.LabelValue(strconv.FormatBool(device.Authorized)),
			tsLabelDeviceUpdateAvailable:           model.LabelValue(strconv.FormatBool(device.UpdateAvailable)),
			tsLabelDeviceKeyExpiryDisabled:         model.LabelValue(strconv.FormatBool(device.KeyExpiryDisabled)),
			tsLabelDeviceBlocksIncomingConnections: model.LabelValue(strconv.FormatBool(device.BlocksIncomingConnections)),
		}

		// the address has a special label that can be used
		addr := device.Addresses[0]
		labels[model.AddressLabel] = model.LabelValue(addr)

		if len(device.Addresses) > 0 {
			addresses := d.tagSeparator + strings.Join(device.Addresses, d.tagSeparator) + d.tagSeparator
			labels[tsLabelDeviceAddresses] = model.LabelValue(addresses)
		}

		if len(device.Tags) > 0 {
			tags := d.tagSeparator + strings.Join(device.Tags, d.tagSeparator) + d.tagSeparator
			labels[tsLabelDeviceTags] = model.LabelValue(tags)
		}

		if len(device.EnabledRoutes) > 0 {
			tags := d.tagSeparator + strings.Join(device.EnabledRoutes, d.tagSeparator) + d.tagSeparator
			labels[tsLabelDeviceEnabledRoutes] = model.LabelValue(tags)
		}
		if len(device.AdvertisedRoutes) > 0 {
			tags := d.tagSeparator + strings.Join(device.AdvertisedRoutes, d.tagSeparator) + d.tagSeparator
			labels[tsLabelDeviceAdvertisedRoutes] = model.LabelValue(tags)
		}

		tg.Targets = append(tg.Targets, labels)
		tg.Labels = labels

		// we add each device their targetgroup to the list to return
		list = append(list, tg)
	}

	return list, nil
}
