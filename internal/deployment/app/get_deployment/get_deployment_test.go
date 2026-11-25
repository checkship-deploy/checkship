package get_deployment_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

func Test_Deployment(t *testing.T) {
	t.Run("should skip the url resolve if the target has been deleted", func(t *testing.T) {
		d := get_deployment.Deployment{
			Target: get_deployment.TargetSummary{
				ID: "target-id",
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:  "app",
						Image: "app-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:      "http1",
								Router:    "http",
								IsCustom:  false,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
							{
								Name:      "http2",
								Router:    "http",
								IsCustom:  true,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
						},
					},
					{
						Name:  "db",
						Image: "db-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:     "tcp1",
								Router:   "tcp",
								IsCustom: true,
								Port:     5432,
							},
						},
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:  "app",
				Image: "app-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:      "http1",
						Router:    "http",
						IsCustom:  false,
						Subdomain: monad.Value("app"),
						Port:      8081,
					},
					{
						Name:      "http2",
						Router:    "http",
						IsCustom:  true,
						Subdomain: monad.Value("app"),
						Port:      8081,
					},
				},
			},
			{
				Name:  "db",
				Image: "db-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:     "tcp1",
						Router:   "tcp",
						IsCustom: true,
						Port:     5432,
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should handle deployment data before the v2.0.0 release", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:          "target-id",
				Url:         monad.Value("https://docker.localhost"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:  "app",
						Image: "app-image",
						Url:   monad.Value("https://app.docker.localhost"),
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:  "app",
				Image: "app-image",
				Url:   monad.Value("https://app.docker.localhost"),
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:     "default",
						Router:   "http",
						IsCustom: false,
						Port:     80,
						Url:      monad.Value("https://app.docker.localhost"),
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should handle deployment data after the v2.0.0 and before the v2.2.0 release", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:          "target-id",
				Url:         monad.Value("https://docker.localhost"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:      "app",
						Image:     "app-image",
						Subdomain: monad.Value("app"),
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:      "app",
				Image:     "app-image",
				Subdomain: monad.Value("app"),
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:      "default",
						Router:    "http",
						IsCustom:  false,
						Port:      80,
						Subdomain: monad.Value("app"),
						Url:       monad.Value("https://app.docker.localhost"),
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should correctly handle new deployments", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:  "target-id",
				Url: monad.Value("https://docker.localhost"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{
					"app-id": {
						"production": {
							"http2": monad.Value[uint](80810),
							"tcp1":  monad.Value[uint](54320),
						},
					},
				}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:  "app",
						Image: "app-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:      "http1",
								Router:    "http",
								IsCustom:  false,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
							{
								Name:      "http2",
								Router:    "http",
								IsCustom:  true,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
						},
					},
					{
						Name:  "db",
						Image: "db-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:     "tcp1",
								Router:   "tcp",
								IsCustom: true,
								Port:     5432,
							},
							{
								Name:     "tcp2",
								Router:   "tcp",
								IsCustom: true,
								Port:     5433,
							},
						},
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:  "app",
				Image: "app-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:      "http1",
						Router:    "http",
						IsCustom:  false,
						Subdomain: monad.Value("app"),
						Port:      8081,
						Url:       monad.Value("https://app.docker.localhost"),
					},
					{
						Name:          "http2",
						Router:        "http",
						IsCustom:      true,
						Subdomain:     monad.Value("app"),
						Port:          8081,
						Url:           monad.Value("http://app.docker.localhost:80810"),
						PublishedPort: monad.Value[uint](80810),
					},
				},
				Urls: []string{"https://app.docker.localhost"},
			},
			{
				Name:  "db",
				Image: "db-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:          "tcp1",
						Router:        "tcp",
						IsCustom:      true,
						Port:          5432,
						Url:           monad.Value("tcp://docker.localhost:54320"),
						PublishedPort: monad.Value[uint](54320),
					},
					{
						Name:     "tcp2",
						Router:   "tcp",
						IsCustom: true,
						Port:     5433,
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should keep the target port on http urls only", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:  "target-id",
				Url: monad.Value("http://docker.localhost:8090"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{
					"app-id": {"production": {"tcp1": monad.Value[uint](32768)}},
				}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{Name: "app", Entrypoints: []get_deployment.Entrypoint{
						{Name: "http1", Router: "http", Subdomain: monad.Value("app"), Port: 80},
					}},
					{Name: "db", Entrypoints: []get_deployment.Entrypoint{
						{Name: "tcp1", Router: "tcp", IsCustom: true, Port: 5432},
					}},
				}),
			},
		}

		d.ResolveServicesUrls()

		services := d.State.Services.Get(get_deployment.Services{})
		assert.DeepEqual(t, []string{"http://app.docker.localhost:8090"}, services[0].Urls)
		assert.Equal(t, "tcp://docker.localhost:32768", services[1].Entrypoints[0].Url.Get(""))
	})

	t.Run("should list custom domains first in the service urls", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:          "target-id",
				Url:         monad.Value("https://apps.example.com"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:    "web",
						Image:   "web-image",
						Domains: []string{"shop.example.com", "www.shop.example.com"},
						Entrypoints: []get_deployment.Entrypoint{
							{Name: "http1", Router: "http", Subdomain: monad.Value("shop"), Port: 80},
						},
					},
					{
						Name:        "db",
						Image:       "db-image",
						Entrypoints: []get_deployment.Entrypoint{},
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		services := d.State.Services.MustGet()
		assert.DeepEqual(t, []string{
			"https://shop.example.com",
			"https://www.shop.example.com",
			"https://shop.apps.example.com",
		}, services[0].Urls)
		assert.HasLength(t, 0, services[1].Urls)
	})

	t.Run("should link custom domains served through a Cloudflare Tunnel in https", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:            "target-id",
				Url:           monad.Value("http://docker.localhost"),
				Entrypoints:   monad.Value(get_deployment.Entrypoints{}),
				TunnelDomains: true,
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:    "web",
						Image:   "web-image",
						Domains: []string{"app.gitdesk.app"},
						Entrypoints: []get_deployment.Entrypoint{
							{Name: "http1", Router: "http", Subdomain: monad.Value("notes-web"), Port: 80},
						},
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		services := d.State.Services.MustGet()
		// The default address keeps the target's own scheme: it isn't behind the tunnel
		assert.DeepEqual(t, []string{"https://app.gitdesk.app", "http://notes-web.docker.localhost"}, services[0].Urls)
	})
}
