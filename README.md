# Nomad Scaleway Target
Nomad Scaleway Target is a Nomad target plugin for the Scaleway cloud platform. It enables horizontal cluster scaling by creating and destroying server instances.

## Requirements

* Nomad autoscaler (tested on v0.3.7)
* Scaleway account with credentials

## Documentation

### Plugin Configuration

The plugin requires a couple of fields to be set in order to function. Below is an example of a minimal working configuration.

```hcl
target "scaleway" {
    driver = "scaleway"

    config = {
        access_token    = "<access-token>"
        secret_token    = "<secret-token>"
        organization_id = "<org-id>"
        project_id      = "<project-id>"
        region          = "nl-ams"
        zone            = "nl-ams-1"
    }
}
```

- `access_token` `(string: "")` - A Scaleway API access token.
- `secret_token` `(string: "")` - A Scaleway API secret token.
- `organization_id` `(string: "")` - The Scaleway organization identifier.
- `project_id` `(string: "")` - The Scaleway project identifier region.
- `zone` `(string: "")` - THe Scaleway zone.

Alternatively, these fields can be specified via environment variables. See the [Scaleway CLI](https://github.com/scaleway/scaleway-cli/blob/master/docs/commands/config.md#documentation-for-scw-config) documentation for more.

### Policy Configuration


``` hcl
check "allocated-cpu" {
    # ...
    target "scaleway" {
        image               = "0d1cf4a3-aae9-4294-9fd9-fefffb297615"
        commercial_type     = "DEV1-S"
        zone                = "nl-ams-1"
    }
}
```

- `name` `(string: "")` - The server instance name.
- `tags` `(string: "")` - A list of comma-separated tags. The tags configured here are appended to a base list of `["nomad", "client", "autoscaler"]`. Only servers with the `autoscaler` tag will be managed by the autoscaler.
- `zone` `(string: "")` - The Scaleway datacenter zone.
- `dynamic_ip` `(string: "false)` - A boolean in string format. If set to `"true"`, sets a dynamic IP after instance creation.
- `commercial_type` `(string: "")` - A Scaleway server instance commercial type. Refer to the [Scaleway Pricing](https://www.scaleway.com/en/pricing/?tags=compute) page for a list of available types.
- `image` `(string: "")` - The Scaleway image ID.
- `enable_ipv6` `(string: "false")` - A boolean in string format. If set to `"true"`, sets an IPv6 IP address after instance creation.
- `security_group` `(string: "")` - The Scaleawy server instance security group ID.
- `placement_group` `(string: "")` - The Scaleway server instance placement group ID.

- `node_class` `(string: "")` - The Nomad [client node class](https://www.nomadproject.io/docs/configuration/client#node_class)
  identifier used to group nodes into a pool of resource. Conflicts with
  `datacenter`.

- `node_drain_deadline` `(duration: "15m")` The Nomad [drain deadline](https://www.nomadproject.io/api-docs/nodes#deadline) to use when performing node draining
  actions. **Note that the default value for this setting differs from Nomad's
  default of 1h.**

- `node_drain_ignore_system_jobs` `(string: "false")` A boolean flag used to
  control if system jobs should be stopped when performing node draining
  actions.

- `node_purge` `(string: "false")` A boolean flag to determine whether Nomad
  clients should be [purged](https://www.nomadproject.io/api-docs/nodes#purge-node) when performing scale in
  actions.

- `node_selector_strategy` `(string: "least_busy")` The strategy to use when
  selecting nodes for termination. Refer to the [node selector
  strategy](https://www.nomadproject.io/docs/autoscaling/internals/node-selector-strategy) documentation for more information.
