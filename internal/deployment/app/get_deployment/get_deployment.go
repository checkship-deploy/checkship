package get_deployment

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

var SourceDataTypes = storage.NewDiscriminatedMapper(func(sd SourceData) string { return sd.Kind() })

type (
	// Retrieve a deployment detail.
	Query struct {
		bus.Query[Deployment]

		AppID            string `json:"-"`
		DeploymentNumber int    `json:"-"`
	}

	Deployment struct {
		AppID            string          `json:"app_id"`
		DeploymentNumber int             `json:"deployment_number"`
		Environment      string          `json:"environment"`
		Target           TargetSummary   `json:"target"`
		Source           Source          `json:"source"`
		State            State           `json:"state"`
		RequestedAt      time.Time       `json:"requested_at"`
		RequestedBy      app.UserSummary `json:"requested_by"`
	}

	// This summary is specific in the sense that it represents a target which may
	// have been deleted since hence the optional fields.
	TargetSummary struct {
		ID          string                   `json:"id"`
		Name        monad.Maybe[string]      `json:"name"`
		Url         monad.Maybe[string]      `json:"url"`
		Status      monad.Maybe[uint8]       `json:"status"`
		Entrypoints monad.Maybe[Entrypoints] `json:"-"`
		// Its custom domains reach it through a Cloudflare Tunnel: Cloudflare serves them over
		// https whatever the target's own scheme.
		TunnelDomains bool `json:"-"`
	}

	Source struct {
		Discriminator string     `json:"discriminator"`
		Data          SourceData `json:"data"`
	}

	SourceData interface {
		Kind() string
	}

	State struct {
		Status     uint8                  `json:"status"`
		Services   monad.Maybe[Services]  `json:"services"`
		ErrCode    monad.Maybe[string]    `json:"error_code"`
		StartedAt  monad.Maybe[time.Time] `json:"started_at"`
		FinishedAt monad.Maybe[time.Time] `json:"finished_at"`
	}

	Services []Service

	Entrypoints map[string]map[string]map[string]monad.Maybe[uint]

	Entrypoint struct {
		Name          string              `json:"name"`
		Router        string              `json:"router"`
		Subdomain     monad.Maybe[string] `json:"subdomain"`
		IsCustom      bool                `json:"is_custom"`
		Port          uint                `json:"port"`
		Url           monad.Maybe[string] `json:"url"`
		PublishedPort monad.Maybe[uint]   `json:"published_port"`
	}

	Service struct {
		Name        string              `json:"name"`
		Image       string              `json:"image"`
		Entrypoints []Entrypoint        `json:"entrypoints"`
		Domains     []string            `json:"domains,omitempty"` // Custom domains the deployment was made with (seelf#5)
		Urls        []string            `json:"urls,omitempty"`    // Where it is served over http: custom domains first, then the default address
		Url         monad.Maybe[string] `json:"url"`               // For compatibility with prior versions
		Subdomain   monad.Maybe[string] `json:"subdomain"`         // For compatibility with prior versions
	}
)

func (Query) Name_() string { return "deployment.query.get_deployment" }

func (s *Services) Scan(value any) error {
	return storage.ScanJSON(value, s)
}

func (e *Entrypoints) Scan(value any) error {
	return storage.ScanJSON(value, e)
}

// Since the target domain is dynamic, compute exposed service urls based on the resolved
// target url and entrypoints if available.
//
// This method should be called after the deployment has been loaded.
func (d *Deployment) ResolveServicesUrls() {
	services, hasServices := d.State.Services.TryGet()
	url, hasUrl := d.Target.Url.TryGet()
	entrypoints, hasEntrypoints := d.Target.Entrypoints.TryGet()

	// Target not found, could not populate services urls
	if !hasUrl || !hasServices || !hasEntrypoints {
		return
	}

	idx := strings.Index(url, "://")
	targetScheme, targetHost := url[:idx+3], url[idx+3:]
	domainScheme := targetScheme

	if d.Target.TunnelDomains {
		domainScheme = "https://"
	}

	for i, service := range services {
		// Compatibility with old deployments
		if service.Url.HasValue() || service.Subdomain.HasValue() {
			compatEntrypoint := Entrypoint{
				Name:      "default",
				Router:    string(domain.RouterHttp),
				Port:      80,
				Subdomain: service.Subdomain, // (> 2.0.0 - < 2.2.0)
				Url:       service.Url,       // (< 2.0.0)
			}

			if subdomain, isSet := compatEntrypoint.Subdomain.TryGet(); !service.Url.HasValue() && isSet {
				compatEntrypoint.Url.Set(targetScheme + subdomain + "." + targetHost)
			}

			services[i].Entrypoints = append(service.Entrypoints, compatEntrypoint)
			continue
		}

		for j, entrypoint := range service.Entrypoints {
			host := targetHost

			if subdomain, isSet := entrypoint.Subdomain.TryGet(); isSet {
				host = subdomain + "." + targetHost
			}

			if !entrypoint.IsCustom {
				entrypoint.Url.Set(targetScheme + host)
				services[i].Entrypoints[j] = entrypoint

				if entrypoint.Router == string(domain.RouterHttp) && len(services[i].Urls) == 0 {
					urls := make([]string, 0, len(service.Domains)+1)
					for _, hostname := range service.Domains {
						urls = append(urls, domainScheme+hostname)
					}
					services[i].Urls = append(urls, targetScheme+host)
				}

				continue
			}

			publishedPort, isAssigned := entrypoints[d.AppID][d.Environment][entrypoint.Name].TryGet()

			if !isAssigned {
				continue
			}

			entrypoint.PublishedPort.Set(publishedPort)
			entrypoint.Url.Set(entrypoint.Router + "://" + withoutPort(host) + ":" + strconv.FormatUint(uint64(publishedPort), 10))

			services[i].Entrypoints[j] = entrypoint
		}
	}
}

// The host of a custom entrypoint gets its own published port: the target URL's port, as in
// http://docker.localhost:8090, belongs to the http entrypoint only.
func withoutPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}

	return host
}
