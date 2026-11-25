# Credits

**checkship** is made by the [checkship-deploy](https://github.com/checkship-deploy) team. Its source code, issues and releases live on [GitHub](https://github.com/checkship-deploy/checkship).

## Built with {#built-with}

**checkship** stands on open-source work. Thank you to everyone behind it.

**Server**

- [Go](https://go.dev), with [Gin](https://gin-gonic.com) for the API, [Cobra](https://cobra.dev) for the command line and [zap](https://github.com/uber-go/zap) for logs
- [SQLite](https://sqlite.org) through [go-sqlite3](https://github.com/mattn/go-sqlite3), and [golang-migrate](https://github.com/golang-migrate/migrate) for the schema
- [Docker](https://www.docker.com) and [Docker Compose](https://docs.docker.com/compose/), whose libraries deploy the applications, and [compose-go](https://github.com/compose-spec/compose-go) to read `compose.yml` files
- [go-git](https://github.com/go-git/go-git) to fetch repositories, and [minio-go](https://github.com/minio/minio-go) to store backups on S3-compatible storage

**Dashboard**

- [Svelte](https://svelte.dev) and [SvelteKit](https://svelte.dev/docs/kit)
- [shadcn-svelte](https://www.shadcn-svelte.com) and [Bits UI](https://bits-ui.com) for the components, [Tailwind CSS](https://tailwindcss.com) for the styles
- [Hugeicons](https://hugeicons.com) for the icons, [Inter](https://rsms.me/inter/) for the type, and [Shiki](https://shiki.style) to highlight code

**On your targets**

- [Traefik](https://traefik.io) as the proxy that exposes services and issues certificates
- [Versity S3 Gateway](https://github.com/versity/versitygw) as the built-in backup storage, and [BusyBox](https://busybox.net) to copy volumes

**This site** is built with [VitePress](https://vitepress.dev).
