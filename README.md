# Tailfire

Tailfire brings the magic of Tailscale to Prometheus users. It offers both a file-based (TODO) and HTTP-based (alpha) service discovery mechanism for your Tailnet to be used by Prometheus.

## Usage

Currently, Tailfire is in a prototypical/PoC stage. To run it, simply execute the following command in the root of the project:

```bash
go run ./cmd/tailfire
```

A simple example configuration file is included and uses Tailscale as the foundation:

```yaml
---
api_token: "tskey-api-foo-bar"
```

The access token can be obtained from the [Tailscale Admin Panel](https://login.tailscale.com/admin/settings/keys).

## Configuration

The basic configuration is straightforward (see the example above):

- **`tag_separator`**: The separator to insert between multiple tags in the `__meta` label set. Default: `,`
- **`tailnet`**: Specifies the Tailnet to use. Default: `"-"` (uses the default Tailnet of the account).
- **`api_url`**: The URL for the management server. Default: `https://api.tailscale.com`
- **`api_token`**: The Tailscale/Headscale/... API token used to access the Tailnet.

However, some edge cases should be kept in mind.

### Multiple Tailnets

By default, Tailfire logs into the account's own Tailnet. It's possible to specify other Tailnets that the account has access to, like this:

```yaml
---
api_token: "tskey-api-foo-bar"
tailnet: "foo.bar@gmail.com"
```

### Headscale

Tailfire also supports Headscale. This requires a slightly more detailed configuration, as shown below:

```yaml
---
api_url: "https://where.headscale.foo.bar"
api_token: "tskey-api-foo-bar"
```
