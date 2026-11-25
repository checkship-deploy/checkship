# Targets

Targets represents an **host** where your deployments will be exposed. When configuring an application, you choose on which target a specific environment should be deployed.

::: info
For now, only one target per host is allowed.
:::

## Proxy configuration {#proxy}

When declaring a target, you must choose how the proxy (needed to make your services available from the outside world) should be managed:

- **Automatic**: **checkship** will deploy and configure a [traefik](https://traefik.io/traefik/) proxy on the target. Services urls will be automatically generated based on the [target's url](#url) and [service file](/reference/providers/docker#exposing-services) when deploying. Exposed services will also join the proxy network.
- **Manual**: you're in charge of **everything** related to services exposure. **checkship** will deploy services on this target without attempting to expose them in any way.

### Url

If the target manages the proxy itself, this url **determines where your applications will be made available**. It should be a **root url** as applications will use subdomains on it.

It may carry a port, such as `http://docker.localhost:8090`, when something in front of the target forwards that port to the proxy, which still listens on 80 (and 443). Application links keep the port; the proxy routes on the hostname alone, and DNS records point to it.

The scheme associated with this url (`http` or `https`) will determine if certificates should be generated or not. With `https`, the proxy gets them from Let's Encrypt with the **HTTP-01** challenge (port 80 must be reachable from the internet), which also works for [custom domains](/reference/applications#custom-domains) proxied by Cloudflare. Targets set up by a version that used the TLS-ALPN-01 challenge are reconfigured once, on their own, at startup.

Only targets with an automatic proxy can serve [custom domains](/reference/applications#custom-domains).

### Cloudflare {#cloudflare}

A target with an automatic proxy can be connected to a Cloudflare account with an API token ([Guide › Cloudflare](/guide/cloudflare#connect)). checkship then keeps the DNS records of the target's custom domains in step on Cloudflare, and for an `http://` target the internet can't reach, runs a Cloudflare Tunnel connector (`checkship-tunnel-<target>`) next to the proxy. The target's page shows the connection under **Cloudflare**, with **Disconnect**. Deleting the target disconnects it.

## Providers {#providers}

You must choose one provider kind when creating a target. Some providers have specific parameters for you to configure how things work. See the [providers reference](/reference/providers) for more informations.

::: warning
Whatever provider you choose, you should make sure your **DNS is correctly configured with a wildcard redirecting to the target host**, the [DigitalOcean procedure](https://docs.digitalocean.com/glossary/wildcard-record/) can be applied to your specific provider.
:::

## Remote targets

When configuring a remote target, you'll **have to add** the public key associated with the private one you'll be using to connect to the host to the `~/.ssh/authorized_keys` file. You can check the [Digital Ocean documentation](https://docs.digitalocean.com/products/droplets/how-to/add-ssh-keys/to-existing-droplet/#with-ssh) for more information.

## Configuration {#configuration}

When creating a target, updating its url / provider configuration or when new custom entrypoints should be created to handle custom ports, a **configuration process** will occur to make sure the target is ready to handle deployments. This [task](/reference/jobs) will deploy the needed infrastructure on the target.

::: info
If you messed your server up, you can **reconfigure** a target by clicking the corresponding button on the interface. It will relaunch the configuration process.
:::

## Cleanup

Deleting a target will (if it has been configured at least once correctly) remove **everything created by checkship** on it:

- Containers
- Images
- Networks
- Volumes

A target that runs the [built-in backup storage](/reference/backups#builtin) cannot be deleted: move the storage to another target on **Backups › Storage** first.

::: info
If you want to delete a target and the cleanup could not be done correctly because of a particular situation, you can [cancel the cleanup job](/reference/jobs#cancellation) from the **jobs** page.
:::
