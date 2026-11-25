package create_app

import (
	"context"
	"errors"
	"slices"
	"strconv"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

type (
	// What custom domains (seelf#5) are checked against.
	DomainsRules struct {
		Targets  domain.TargetsReader
		Apps     domain.AppsReader
		Reserved monad.Maybe[domain.Url] // The address checkship itself is exposed on, if any
	}

	// Custom domains given for one environment, as typed: hostnames per service.
	DomainsInput struct {
		Environment domain.Environment
		Target      domain.TargetID
		Raw         map[string][]string
	}

	// Detail of a domain_already_used error: which application has it.
	DomainOwner struct {
		App string `json:"app"`
	}

	domainEntry struct {
		field    string
		hostname domain.Hostname
		env      domain.Environment
	}
)

func (o DomainOwner) Error() string { return o.App }

// Validates the custom domains given for some environments of an application and returns
// them parsed, per environment. Field errors are keyed
// "<environment>.domains.<service>.<index>" (or "<environment>.domains[.<service>]" for limits
// and target problems), so the dashboard shows each one under its domain.
//
// Rules: valid hostnames; at most MaxDomainsPerService per service and
// MaxDomainsPerEnvironment per environment; no domain twice in the application; a target with
// checkship's proxy; nothing inside the target's own domain or equal to checkship's own
// address; nothing another application already serves.
func BuildDomains(
	ctx context.Context,
	rules DomainsRules,
	app domain.AppID,
	inputs ...DomainsInput,
) (map[domain.Environment]domain.ServicesDomains, error) {
	var (
		errs    = make(validate.FieldErrors)
		result  = make(map[domain.Environment]domain.ServicesDomains, len(inputs))
		entries []domainEntry
		seen    = make(map[domain.Hostname]bool)
	)

	for _, input := range inputs {
		prefix := string(input.Environment) + ".domains"
		parsed := make(domain.ServicesDomains, len(input.Raw))
		total := 0

		services := make([]string, 0, len(input.Raw))
		for service := range input.Raw {
			services = append(services, service)
		}
		slices.Sort(services)

		for _, service := range services {
			values := input.Raw[service]
			field := prefix + "." + service

			if service == "" {
				errs[prefix] = domain.ErrInvalidHostname
				continue
			}

			if len(values) > domain.MaxDomainsPerService {
				errs[field] = domain.ErrTooManyDomains
				continue
			}

			total += len(values)

			for i, value := range values {
				key := field + "." + strconv.Itoa(i)
				hostname, err := domain.HostnameFrom(value)

				if err != nil {
					errs[key] = err
					continue
				}

				if seen[hostname] {
					errs[key] = domain.ErrDomainDuplicated
					continue
				}

				seen[hostname] = true
				parsed[service] = append(parsed[service], hostname)
				entries = append(entries, domainEntry{field: key, hostname: hostname, env: input.Environment})
			}
		}

		if total > domain.MaxDomainsPerEnvironment {
			errs[prefix] = domain.ErrTooManyDomains
		}

		result[input.Environment] = parsed
	}

	if len(errs) > 0 {
		return nil, validate.NewError(errs)
	}

	if len(entries) == 0 {
		return result, nil
	}

	// The target serves its own subdomains, and a manual proxy can't route anything
	for _, input := range inputs {
		if len(result[input.Environment]) == 0 {
			continue
		}

		target, err := rules.Targets.GetByID(ctx, input.Target)

		if errors.Is(err, apperr.ErrNotFound) {
			continue // Reported by the naming check on the target field
		}

		if err != nil {
			return nil, err
		}

		url, isAutomatic := target.Url().TryGet()

		if !isAutomatic {
			errs[string(input.Environment)+".domains"] = domain.ErrCustomDomainsNeedAutomaticProxy
			continue
		}

		for _, entry := range entries {
			if entry.env == input.Environment && entry.hostname.IsUnder(url.Hostname()) {
				errs[entry.field] = domain.ErrDomainReservedByTarget
			}
		}
	}

	// checkship's own address: an application must not capture the dashboard
	if reserved, isSet := rules.Reserved.TryGet(); isSet {
		for _, entry := range entries {
			if string(entry.hostname) == reserved.Hostname() {
				errs[entry.field] = domain.ErrDomainReserved
			}
		}
	}

	// Reported together with the reservation errors above, so a save lists every problem once
	hostnames := make([]domain.Hostname, len(entries))
	for i, entry := range entries {
		hostnames[i] = entry.hostname
	}

	owners, err := rules.Apps.GetDomainsOwners(ctx, app, hostnames)

	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if _, hasError := errs[entry.field]; hasError {
			continue
		}

		if owner, taken := owners[entry.hostname]; taken {
			errs[entry.field] = apperr.NewWithDetail(domain.ErrDomainAlreadyUsed.Error(), DomainOwner{App: string(owner)})
		}
	}

	if len(errs) > 0 {
		return nil, validate.NewError(errs)
	}

	return result, nil
}
