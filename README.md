# Tailfire

Tailfire brings the magic of Tailscale to Prometheus users. It offers both a
file-based and HTTP-based service discovery mechanism for your Tailnet to be used
by Prometheus.

## Usage

Currently, Tailfire is in a prototypical/PoC stage. It's built with
[kong](https://github.com/alecthomas/kong), so to now it's options, simply check
the `--help` output:

```bash
$ tailfire --help
Usage: tailfire <command> [flags]

A Prometheus service discovery tool for Tailscale

Flags:
  -h, --help                         Show context-sensitive help.
      --config.file="config.yaml"    location of the config path

Commands:
  serve [flags]
    serve a Prometheus HTTP SD endpoint

  generate [flags]
    generate a Prometheus file SD file

Run "tailfire <command> --help" for more information on a command.
```

Tailfire support 2 kinds of running modes: `serve` which will sping up an
endpoint that can be consumed by prometheu HTTP SD (on `/prometheus/targets`) or `generate` which is a
oneshot operation that outputs a file compliant with Prometheus file SD.

```bash
$ tailfire serve --help
Usage: tailfire serve [flags]

serve a Prometheus HTTP SD endpoint

Flags:
  -h, --help                         Show context-sensitive help.
      --config.file="config.yaml"    location of the config path

      --port=8080                    http port for tailfire sd endpoint (default: 8080)
      --host="localhost"             http host for tailfire sd endpoint (default: localhost)
      --[no-]refresh-update          update the refresh interval based on the X-Prometheus-Refresh-Interval header (default: true)
      --refresh.interval=60          initials refresh duration in seconds (default: 60)

$ tailfire generate --help
Usage: tailfire generate [flags]

generate a Prometheus file SD file

Flags:
  -h, --help                            Show context-sensitive help.
      --config.file="config.yaml"       location of the config path

      --output.file="tailscale.json"    file to write the tailscale targets too (default: tailscale.json)
      --output.perm=0644                filepermissions on the outputted file (default: 0644)
```

In the `serve` mode, a simple Prometheus configuration can scrape the service
discovery:

```yaml
scrape_configs:
  - job_name: "tailfire"
    http_sd_configs:
      - url: "http://localhost:8080/prometheus/targets"
```

## Configuration

A simple example configuration file is included and uses Tailscale as the
foundation:

```yaml
---
api_token: "tskey-api-foo-bar"
```

The access token can be obtained from the [Tailscale Admin
Panel](https://login.tailscale.com/admin/settings/keys).

Tailfire will update every 60 seconds. In case a Prometheus is configured with a
service refresh interval faster than this, Tailfire will adjust to the lowest
duration from all requests based on the `X-Prometheus-Refresh-Interval-Seconds`
HTTP header. Currently it's not possible to increase the duration at runtime.

The basic configuration is straightforward (see the example above):

- **`tag_separator`**: The separator to insert between multiple tags in the
  `__meta` label set. Default: `,`
- **`tailnet`**: Specifies the Tailnet to use. Default: `"-"` (uses the default
  Tailnet of the account).
- **`api_url`**: The URL for the management server. Default:
  `https://api.tailscale.com`
- **`api_token`**: The Tailscale/Headscale/... API token used to access the
  Tailnet.

However, some edge cases should be kept in mind.

### Multiple Tailnets

By default, Tailfire logs into the account's own Tailnet. It's possible to
specify other Tailnets that the account has access to, like this:

```yaml
---
api_token: "tskey-api-foo-bar"
tailnet: "foo.bar@gmail.com"
```

### Headscale

Tailfire also supports Headscale. This requires a slightly more detailed
configuration, as shown below:

```yaml
---
api_url: "https://where.headscale.foo.bar"
api_token: "tskey-api-foo-bar"
```
