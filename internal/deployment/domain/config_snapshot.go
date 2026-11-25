package domain

import (
	"strings"

	"github.com/checkship-deploy/checkship/pkg/monad"
)

// Holds data related to the configuration of the final application. It should
// have everything needed to resolve service and image names and is the primarily used
// structure during the deployment by a provider.
type ConfigSnapshot struct {
	appid       AppID
	appname     AppName
	environment Environment
	target      TargetID
	vars        monad.Maybe[ServicesEnv]
	domains     monad.Maybe[ServicesDomains] // Custom domains, as they were when the deployment was queued
}

// Builds a new config snapshot for the given environment.
func (a *App) configSnapshotFor(env Environment) (ConfigSnapshot, error) {
	var (
		conf     EnvironmentConfig
		snapshot ConfigSnapshot
	)

	switch env {
	case Production:
		conf = a.production
	case Staging:
		conf = a.staging
	default:
		return snapshot, ErrInvalidEnvironmentName
	}

	snapshot.appid = a.id
	snapshot.appname = a.name
	snapshot.environment = env
	snapshot.target = conf.Target()
	snapshot.vars = conf.Vars()
	snapshot.domains = conf.Domains()

	return snapshot, nil
}

func (c ConfigSnapshot) AppID() AppID                          { return c.appid }
func (c ConfigSnapshot) AppName() AppName                      { return c.appname }
func (c ConfigSnapshot) Environment() Environment              { return c.environment }
func (c ConfigSnapshot) Target() TargetID                      { return c.target }
func (c ConfigSnapshot) Vars() monad.Maybe[ServicesEnv]        { return c.vars } // FIXME: If I want to follow my mantra, it should returns a readonly map
func (c ConfigSnapshot) Domains() monad.Maybe[ServicesDomains] { return c.domains }

// Custom domains of the given service, if any.
func (c ConfigSnapshot) DomainsFor(service string) []Hostname {
	if domains, isSet := c.domains.TryGet(); isSet {
		return domains.For(service)
	}

	return nil
}

// Retrieve environment variables associated with the given service name.
// FIXME: If I want to follow my mantra, it should returns a readonly map
func (c ConfigSnapshot) EnvironmentVariablesFor(service string) (m monad.Maybe[EnvVars]) {
	env, isSet := c.vars.TryGet()

	if !isSet {
		return m
	}

	vars, exists := env[service]

	if !exists {
		return m
	}

	m.Set(vars)

	return m
}

// Retrieve the name of the project which is the combination of the appname, environment and appid
// targeted by this configuration.
func (c ConfigSnapshot) ProjectName() string {
	return ProjectName(c.appname, c.environment, c.appid)
}

// Compose project name of an application environment. Its containers, networks
// and volumes are named after it.
func ProjectName(name AppName, env Environment, id AppID) string {
	return string(name) + "-" + string(env) + "-" + strings.ToLower(string(id))
}

// Returns the subdomain that will be used to expose a specific service.
func (c ConfigSnapshot) subDomain(service string, isDefault bool) string {
	subdomain := string(c.appname)

	if !c.environment.IsProduction() {
		subdomain += "-" + string(c.environment)
	}

	// If the default domain has already been taken by another service, build a
	// unique subdomain with the service name being exposed.
	if !isDefault {
		subdomain = service + "." + subdomain
	}

	return subdomain
}

// The name other applications on the target reach a service at, on the target's network:
// <service>.<app>, with -staging in staging, so "postgres.db" or "postgres.db-staging". App
// names are unique on a target, so these are too, where a bare service name such as "db"
// would clash between applications.
func (c ConfigSnapshot) InternalHost(service string) string {
	return service + "." + c.subDomain(service, true)
}

// Builds a unique image name for the given service.
func (c ConfigSnapshot) imageName(service string) string {
	return string(c.appname) + "-" + strings.ToLower(string(c.appid)) + "/" + service + ":" + string(c.environment)
}

// Builds a qualified name, truly unique, for the given service.
func (c ConfigSnapshot) qualifiedName(service string) string {
	return c.ProjectName() + "-" + service
}
