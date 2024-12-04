package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"tailscale.com/client/tailscale"
)

var tailscaleTestAPIKey = "tskey-foo-bar"

func TestTailscaleRefresh(t *testing.T) {
	tailscale.I_Acknowledge_This_API_Is_Unstable = true
	mock := httptest.NewServer(http.HandlerFunc(MockTailscaleAPI))
	defer mock.Close()

	cfgString := fmt.Sprintf(`
---
api_url: %s
api_token: %s
`, mock.URL, tailscaleTestAPIKey)

	var cfg SDConfig
	require.NoError(t, yaml.UnmarshalStrict([]byte(cfgString), &cfg))

	d, err := NewDiscovery(&cfg, nil)
	if err != nil {
		fmt.Println(err)
	}
	require.NoError(t, err)

	ctx := context.Background()
	tgs, err := d.refresh(ctx)
	if err != nil {
		fmt.Println(err)
	}
	require.NoError(t, err)

	require.Len(t, tgs, 3)

	for i, lbls := range []model.LabelSet{
		{
			"__address__":                                         "100.101.101.102",
			"__meta_tailscale_device_addresses":                   ",100.101.101.102,1234:5678:90ab:cdef:1234:5678:90ab:cdef,",
			"__meta_tailscale_device_id":                          "0000000000000002",
			"__meta_tailscale_device_name":                        "test.foo.bar",
			"__meta_tailscale_device_hostname":                    "test",
			"__meta_tailscale_device_client_version":              "1.56.1",
			"__meta_tailscale_device_os":                          "linux",
			"__meta_tailscale_device_tags":                        ",tag:exit,tag:hass,tag:node,tag:prod,",
			"__meta_tailscale_device_user":                        "foo.bar@prometheus.io",
			"__meta_tailscale_device_is_external":                 "false",
			"__meta_tailscale_device_authorized":                  "true",
			"__meta_tailscale_device_update_available":            "true",
			"__meta_tailscale_device_key_expiry_disabled":         "false",
			"__meta_tailscale_device_blocks_incoming_connections": "false",
		},
		{
			"__address__":                                         "100.101.101.103",
			"__meta_tailscale_device_addresses":                   ",100.101.101.103,1234:5678:90ab:c3ef:1234:5678:90ab:cdef,",
			"__meta_tailscale_device_id":                          "0000000000000003",
			"__meta_tailscale_device_name":                        "test.foo.bar",
			"__meta_tailscale_device_hostname":                    "test",
			"__meta_tailscale_device_client_version":              "1.56.1",
			"__meta_tailscale_device_os":                          "macos",
			"__meta_tailscale_device_tags":                        ",tag:hello,tag:world,tag:bar,",
			"__meta_tailscale_device_user":                        "foo.bar@prometheus.io",
			"__meta_tailscale_device_is_external":                 "false",
			"__meta_tailscale_device_authorized":                  "true",
			"__meta_tailscale_device_update_available":            "true",
			"__meta_tailscale_device_key_expiry_disabled":         "false",
			"__meta_tailscale_device_blocks_incoming_connections": "false",
		},
		{
			"__address__":                                         "100.101.101.104",
			"__meta_tailscale_device_addresses":                   ",100.101.101.104,1234:5678:90ab:c4ef:1234:5678:90ab:cdef,",
			"__meta_tailscale_device_id":                          "0000000000000004",
			"__meta_tailscale_device_name":                        "prometheus.foo.bar",
			"__meta_tailscale_device_hostname":                    "prometheus",
			"__meta_tailscale_device_client_version":              "1.56.1",
			"__meta_tailscale_device_os":                          "android",
			"__meta_tailscale_device_tags":                        ",tag:hello,tag:world,tag:bar,tag:foo,tag:test,",
			"__meta_tailscale_device_user":                        "foo.bar@prometheus.io",
			"__meta_tailscale_device_is_external":                 "false",
			"__meta_tailscale_device_authorized":                  "true",
			"__meta_tailscale_device_update_available":            "false",
			"__meta_tailscale_device_key_expiry_disabled":         "true",
			"__meta_tailscale_device_blocks_incoming_connections": "true",
			"__meta_tailscale_device_enabled_routes":              ",10.0.0.0/8,",
			"__meta_tailscale_device_advertised_routes":           ",192.168.0.0/14,10.0.0.0/8,172.12.0.0/12,",
		},
	} {
		t.Run(fmt.Sprintf("item %d", i), func(t *testing.T) {
			// check sizing on each targetGroup
			tg := tgs[i]
			require.NotNil(t, tg)
			require.NotNil(t, tg.Targets)
			require.Len(t, tg.Targets, 1)

			// validate the targetGroup value
			require.Equal(t, lbls, tgs[i].Targets[0])
			require.Equal(t, lbls, tgs[i].Labels)
		})
	}
}

func MockTailscaleAPI(w http.ResponseWriter, r *http.Request) {
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(tailscaleTestAPIKey+":")))
	if r.Header.Get("Authorization") != authHeader {
		http.Error(w, "bad application key", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/api/v2/tailnet/-/devices" {
		tailnetList, err := os.ReadFile("../../testdata/tailnet.json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(tailnetList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
