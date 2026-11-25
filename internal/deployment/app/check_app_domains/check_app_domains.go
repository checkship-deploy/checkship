package check_app_domains

import (
	"context"
	"net"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// DNS statuses
const (
	DnsPointsHere       = "points_here"       // Resolves to the target (or a CNAME leads to its default address)
	DnsBehindCloudflare = "behind_cloudflare" // Only Cloudflare addresses: proxied, the check can't see past it
	DnsPointsElsewhere  = "points_elsewhere"  // Resolves to other addresses
	DnsNoRecord         = "no_record"         // The name doesn't resolve
	DnsUnknown          = "unknown"           // The lookup failed (resolver unreachable, timeout)
)

// CAA statuses
const (
	CaaOk      = "caa_ok"     // No CAA record, or one that allows Let's Encrypt
	CaaBlocks  = "caa_blocks" // A CAA record allows other CAs only
	CaaUnknown = "caa_unknown"
)

// Certificate statuses (https targets, domains pointing here only)
const (
	CertificateOk          = "certificate_ok"      // A valid certificate for the domain
	CertificateDefault     = "default_certificate" // The proxy's own certificate: not issued yet
	CertificateUnreachable = "unreachable"         // Nothing answered on port 443
)

type (
	// Check the DNS records, CAA records and certificates of an application environment's
	// custom domains (seelf#5).
	Query struct {
		bus.Query[Result]

		AppID       string `json:"-"`
		Environment string `json:"-"`
	}

	Result struct {
		Environment string         `json:"environment"`
		Https       bool           `json:"https"`
		Expected    Expected       `json:"expected"`
		Domains     []DomainStatus `json:"domains"`
	}

	// Where the domains should point: the environment's default address on its target and
	// the addresses it resolves to.
	Expected struct {
		Host      string   `json:"host"`
		Addresses []string `json:"addresses"`
	}

	DomainStatus struct {
		Service       string     `json:"service"`
		Hostname      string     `json:"hostname"`
		Unicode       string     `json:"unicode"`
		Dns           string     `json:"dns"`
		CloudflareDns bool       `json:"cloudflare_dns"` // Its zone's nameservers are Cloudflare's: Connect Cloudflare can set it up
		Addresses     []string   `json:"addresses"`
		Caa           string     `json:"caa"`
		CaaIssuers    []string   `json:"caa_issuers,omitempty"`
		Certificate   string     `json:"certificate,omitempty"`
		Issuer        string     `json:"issuer,omitempty"`
		ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	}

	// Looks domains up. Implementations must only ever connect to the allowlisted addresses
	// (the target's own), so a check can't be turned against other hosts.
	Checker interface {
		// Addresses the given host resolves to (the allowlist for certificate checks).
		Resolve(ctx context.Context, host string) []net.IP
		// Checks one hostname against the expected host and addresses.
		Check(ctx context.Context, hostname domain.Hostname, expectedHost string, expected []net.IP, https bool) DomainStatus
	}
)

func (Query) Name_() string { return "deployment.query.check_app_domains" }

func Handler(
	apps domain.AppsReader,
	targets domain.TargetsReader,
	checker Checker,
) bus.RequestHandler[Result, Query] {
	return func(ctx context.Context, q Query) (Result, error) {
		env, err := domain.EnvironmentFrom(q.Environment)

		if err != nil {
			return Result{}, err
		}

		app, err := apps.GetByID(ctx, domain.AppID(q.AppID))

		if err != nil {
			return Result{}, err
		}

		config := app.Config(env)
		result := Result{Environment: string(env), Domains: []DomainStatus{}}

		target, err := targets.GetByID(ctx, config.Target())

		if err != nil {
			return Result{}, err
		}

		url, isAutomatic := target.Url().TryGet()

		if !isAutomatic {
			return Result{}, domain.ErrCustomDomainsNeedAutomaticProxy
		}

		result.Https = url.UseSSL()
		result.Expected.Host = app.DefaultSubdomain(env) + "." + url.Hostname()
		expected := checker.Resolve(ctx, result.Expected.Host)
		result.Expected.Addresses = addresses(expected)

		domains, hasDomains := config.Domains().TryGet()

		if !hasDomains {
			return result, nil
		}

		for service, hostnames := range domains {
			for _, hostname := range hostnames {
				status := checker.Check(ctx, hostname, result.Expected.Host, expected, result.Https)
				status.Service = service
				result.Domains = append(result.Domains, status)
			}
		}

		sortStatuses(result.Domains)

		return result, nil
	}
}

func addresses(ips []net.IP) []string {
	out := make([]string, len(ips))

	for i, ip := range ips {
		out[i] = ip.String()
	}

	return out
}
