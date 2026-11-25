import { defineConfig } from "vitepress";

// https://vitepress.dev/reference/site-config
export default defineConfig({
  base: "/checkship/",
  ignoreDeadLinks: "localhostLinks",
  title: "checkship documentation",
  head: [["link", { rel: "icon", href: "/checkship/favicon.svg" }]],
  description: "Lightweight self-hosted deployment platform written in Go.",
  themeConfig: {
    siteTitle: false,
    search: {
      provider: "local",
    },
    logo: {
      light: "/logo-light.svg",
      dark: "/logo-dark.svg",
      alt: "checkship",
    },
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: "Home", link: "/" },
      { text: "Docs", link: "/guide/quickstart" },
    ],

    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Quickstart", link: "/guide/quickstart" },
          { text: "Templates", link: "/guide/templates" },
          { text: "Installation", link: "/guide/installation" },
          { text: "Updating", link: "/guide/updating" },
          { text: "Configuration", link: "/guide/configuration" },
          { text: "Migrating major versions", link: "/guide/migration" },
          {
            text: "Continuous Integration / Deployment",
            link: "/guide/continuous-integration-deployment",
          },
          { text: "Custom domains", link: "/guide/custom-domains" },
          { text: "Cloudflare", link: "/guide/cloudflare" },
        ],
      },
      {
        text: "Reference",
        items: [
          {
            text: "Targets",
            link: "/reference/targets",
          },
          {
            text: "Providers",
            link: "/reference/providers",
            items: [
              {
                text: "Docker",
                link: "/reference/providers/docker",
              },
            ],
          },
          {
            text: "Registries",
            link: "/reference/registries",
          },
          {
            text: "Applications",
            link: "/reference/applications",
          },
          {
            text: "Deployments",
            link: "/reference/deployments",
          },
          {
            text: "Sources",
            link: "/reference/deployments#sources",
          },
          {
            text: "Backups",
            link: "/reference/backups",
          },
          {
            text: "Jobs",
            link: "/reference/jobs",
          },
          {
            text: "API",
            link: "/reference/api",
          },
          {
            text: "FAQ",
            link: "/reference/faq",
          },
        ],
      },
      {
        text: "Contributing",
        items: [
          { text: "Docs", link: "/contributing/docs" },
          { text: "Backend", link: "/contributing/backend" },
          { text: "Frontend", link: "/contributing/frontend" },
          { text: "Credits", link: "/contributing/credits" },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/checkship-deploy/checkship" },
    ],
  },
});
