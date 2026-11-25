// Package cloudflare_setup is Connect Cloudflare: a target keeps the Cloudflare API token the
// person created from a pre-filled link, and checkship sets up the DNS records of the target's
// custom domains in their account. For a target the internet can't reach, it routes them through
// a Cloudflare Tunnel whose connector runs next to the target's proxy.
//
// checkship only ever updates or deletes DNS records it created (they carry
// domain.CloudflareRecordComment), and only replaces someone else's record when asked to.
package cloudflare_setup

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

type (
	Deps struct {
		Connections domain.CloudflareConnections
		Cloudflare  domain.Cloudflare
		Targets     domain.TargetsReader
		Apps        domain.AppsReader
		Connector   domain.TunnelConnector
	}

	// Connect a target to Cloudflare with an API token (Zone Read, DNS Edit, and Cloudflare
	// Tunnel Edit for tunnels).
	ConnectCommand struct {
		bus.Command[Status]

		TargetID string `json:"-"`
		Token    string `json:"token"`
	}

	// What a target has on Cloudflare.
	StatusQuery struct {
		bus.Query[Status]

		TargetID string `json:"-"`
	}

	// Set up (or bring up to date) the target's custom domains on Cloudflare, now.
	SyncCommand struct {
		bus.Command[SyncResult]

		TargetID string              `json:"-"`
		Mode     monad.Maybe[string] `json:"mode"`    // tunnel or dns; the target's current one when omitted
		Replace  []string            `json:"replace"` // Hostnames whose existing record (not checkship's) may be replaced
	}

	// The same, as a background job, after domains or applications change on a connected target.
	SyncJob struct {
		bus.Command[bus.UnitType]

		TargetID string `json:"target_id"`
	}

	// Remove everything checkship created on Cloudflare for a target, and forget its token.
	DisconnectCommand struct {
		bus.Command[bus.UnitType]

		TargetID string `json:"target_id"`
	}

	Status struct {
		Connected     bool           `json:"connected"`
		AccountName   string         `json:"account_name,omitempty"`
		AccountID     string         `json:"account_id,omitempty"` // Not a secret: in every dashboard URL
		TunnelAllowed bool           `json:"tunnel_allowed"`
		Mode          string         `json:"mode,omitempty"`
		Zones         []string       `json:"zones"`
		Tunnel        *TunnelStatus  `json:"tunnel,omitempty"`
		Records       []RecordStatus `json:"records"`
		ConnectedAt   *time.Time     `json:"connected_at,omitempty"`
	}

	TunnelStatus struct {
		Name        string `json:"name"`
		Status      string `json:"status"`
		Connections int    `json:"connections"`
	}

	RecordStatus struct {
		Hostname string `json:"hostname"`
		Type     string `json:"type"`
		Content  string `json:"content"`
	}

	SyncResult struct {
		Tasks     []Task     `json:"tasks"`
		Conflicts []Conflict `json:"conflicts"`
		Outside   []string   `json:"outside"` // Domains in no zone of the token's account: set up by hand
		Status    Status     `json:"status"`
	}

	Task struct {
		Key    string `json:"key"` // tunnel, routes, records, connector, cleanup
		Detail string `json:"detail"`
	}

	// A record checkship didn't create, in the way of a domain.
	Conflict struct {
		Hostname string `json:"hostname"`
		Type     string `json:"type"`
		Content  string `json:"content"`
		Zone     string `json:"zone"`
		Want     Wanted `json:"want"`   // The record to put instead, by hand or with replace
		Reason   string `json:"reason"` // record (someone else's is there) or dns_edit (the token can't write DNS)
	}

	// A DNS record as Cloudflare's "Add record" form asks for it.
	Wanted struct {
		Type    string `json:"type"`
		Name    string `json:"name"` // "@" for the zone itself, else the part before it
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
	}
)

func (ConnectCommand) Name_() string    { return "deployment.command.connect_cloudflare" }
func (StatusQuery) Name_() string       { return "deployment.query.get_cloudflare" }
func (SyncCommand) Name_() string       { return "deployment.command.setup_cloudflare" }
func (SyncJob) Name_() string           { return "deployment.command.sync_cloudflare" }
func (j SyncJob) ResourceID() string    { return j.TargetID }
func (DisconnectCommand) Name_() string { return "deployment.command.disconnect_cloudflare" }
func (c DisconnectCommand) ResourceID() string {
	return c.TargetID
}

// Why a domain's record isn't set up.
const (
	ConflictRecord  = "record"   // A record checkship didn't create is there
	ConflictDnsEdit = "dns_edit" // The token can't write DNS records: added by hand
)

// Group of Cloudflare jobs of a target, so they never run together.
func Group(target domain.TargetID) string { return "deployment.cloudflare." + string(target) }

func ConnectHandler(d Deps) bus.RequestHandler[Status, ConnectCommand] {
	return func(ctx context.Context, cmd ConnectCommand) (Status, error) {
		target, err := d.automaticTarget(ctx, domain.TargetID(cmd.TargetID))

		if err != nil {
			return Status{}, err
		}

		token := strings.TrimSpace(cmd.Token)

		if !domain.ValidCloudflareToken(token) {
			return Status{}, validate.NewError(validate.FieldErrors{"token": domain.ErrCloudflareTokenInvalid})
		}

		if err = d.Cloudflare.VerifyToken(ctx, token); err != nil {
			return Status{}, tokenField(err)
		}

		zones, err := d.Cloudflare.Zones(ctx, token)

		if err != nil {
			return Status{}, tokenField(err)
		}

		if len(zones) == 0 {
			return Status{}, validate.NewError(validate.FieldErrors{"token": missing(domain.CloudflarePermissionZoneRead)})
		}

		// The account of the target's domains when one is there, else the first one
		account := zones[0]

		if domains, err := d.Apps.GetDomainsOnTarget(ctx, target.ID()); err == nil {
			for _, item := range domains {
				if zone, found := domain.ZoneOf(item.Hostname, zones); found {
					account = zone
					break
				}
			}
		}

		conn := domain.CloudflareConnection{
			Target:      target.ID(),
			Token:       token,
			AccountID:   account.AccountID,
			AccountName: account.AccountName,
			Mode:        domain.CloudflareModeDns,
			ConnectedAt: time.Now().UTC(),
			ConnectedBy: auth.CurrentUser(ctx).MustGet(),
		}

		// Tunnel Edit is optional: without it, only DNS records can be set up
		_, err = d.Cloudflare.FindTunnel(ctx, token, account.AccountID, domain.CloudflareTunnelName(target.ID()))

		switch {
		case err == nil:
			conn.TunnelAllowed = true
		case errors.Is(err, domain.ErrCloudflarePermissionMissing):
		default:
			return Status{}, err
		}

		// A target the internet can't reach (http://) goes through a tunnel by default
		if conn.TunnelAllowed && !target.Url().MustGet().UseSSL() {
			conn.Mode = domain.CloudflareModeTunnel
		}

		// Reconnecting keeps the tunnel already created in the same account
		if existing, err := d.Connections.GetCloudflareConnection(ctx, target.ID()); err == nil && existing.AccountID == conn.AccountID {
			conn.TunnelID, conn.TunnelName, conn.Mode = existing.TunnelID, existing.TunnelName, existing.Mode
		}

		if err = d.Connections.SaveCloudflareConnection(ctx, conn); err != nil {
			return Status{}, err
		}

		return d.status(ctx, conn, zones)
	}
}

func StatusHandler(d Deps) bus.RequestHandler[Status, StatusQuery] {
	return func(ctx context.Context, q StatusQuery) (Status, error) {
		conn, err := d.Connections.GetCloudflareConnection(ctx, domain.TargetID(q.TargetID))

		if errors.Is(err, apperr.ErrNotFound) {
			return Status{Zones: []string{}, Records: []RecordStatus{}}, nil
		}

		if err != nil {
			return Status{}, err
		}

		zones, err := d.Cloudflare.Zones(ctx, conn.Token)

		if err != nil {
			return Status{}, err
		}

		return d.status(ctx, conn, zones)
	}
}

func SyncHandler(d Deps) bus.RequestHandler[SyncResult, SyncCommand] {
	return func(ctx context.Context, cmd SyncCommand) (SyncResult, error) {
		return d.sync(ctx, domain.TargetID(cmd.TargetID), cmd.Mode, cmd.Replace)
	}
}

func SyncJobHandler(d Deps) bus.RequestHandler[bus.UnitType, SyncJob] {
	return func(ctx context.Context, job SyncJob) (bus.UnitType, error) {
		_, err := d.sync(ctx, domain.TargetID(job.TargetID), monad.None[string](), nil)

		// Only Cloudflare being unreachable is worth retrying: a refused token, a missing
		// permission or a target that isn't connected won't fix themselves, and the drawer
		// shows them the next time it's opened.
		if err != nil && !errors.Is(err, domain.ErrCloudflareUnavailable) {
			return bus.Unit, nil
		}

		return bus.Unit, err
	}
}

func DisconnectHandler(d Deps) bus.RequestHandler[bus.UnitType, DisconnectCommand] {
	return func(ctx context.Context, cmd DisconnectCommand) (bus.UnitType, error) {
		id := domain.TargetID(cmd.TargetID)
		conn, err := d.Connections.GetCloudflareConnection(ctx, id)

		if errors.Is(err, apperr.ErrNotFound) {
			return bus.Unit, nil
		}

		if err != nil {
			return bus.Unit, err
		}

		target, targetErr := d.Targets.GetByID(ctx, id)

		// The connector first, so the tunnel has no live connection when it is deleted
		if targetErr == nil && conn.TunnelID.HasValue() {
			if err = d.Connector.RemoveTunnelConnector(ctx, target); err != nil {
				return bus.Unit, err
			}
		}

		if err = d.removeAll(ctx, conn); err != nil && !refused(err) {
			return bus.Unit, err
		}

		return bus.Unit, d.Connections.DeleteCloudflareConnection(ctx, id)
	}
}

// Queue a sync of the targets whose domains just changed.
func OnAppEnvChangedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		targets := []domain.TargetID{evt.Config.Target()}

		if evt.TargetHasChanged() {
			targets = append(targets, evt.OldConfig.Target())
		}

		return queueSync(ctx, scheduler, targets...)
	}
}

func OnAppCreatedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppCreated] {
	return func(ctx context.Context, evt domain.AppCreated) error {
		if !evt.Production.Domains().HasValue() && !evt.Staging.Domains().HasValue() {
			return nil
		}

		return queueSync(ctx, scheduler, evt.Production.Target(), evt.Staging.Target())
	}
}

func OnAppCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		return queueSync(ctx, scheduler, evt.ProductionConfig.Target(), evt.StagingConfig.Target())
	}
}

// A target being deleted takes its tunnel and records with it (the delete job waits for this one).
func OnTargetCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.TargetCleanupRequested] {
	return func(ctx context.Context, evt domain.TargetCleanupRequested) error {
		return scheduler.Queue(ctx, DisconnectCommand{TargetID: string(evt.ID)},
			bus.WithGroup(Group(evt.ID)))
	}
}

func queueSync(ctx context.Context, scheduler bus.Scheduler, targets ...domain.TargetID) error {
	seen := make(map[domain.TargetID]bool)

	for _, target := range targets {
		if seen[target] {
			continue
		}

		seen[target] = true

		if err := scheduler.Queue(ctx, SyncJob{TargetID: string(target)},
			bus.WithGroup(Group(target)), bus.WithPolicy(bus.JobPolicyMerge)); err != nil {
			return err
		}
	}

	return nil
}

func (d Deps) automaticTarget(ctx context.Context, id domain.TargetID) (domain.Target, error) {
	target, err := d.Targets.GetByID(ctx, id)

	if err != nil {
		return target, err
	}

	if !target.Url().HasValue() {
		return target, domain.ErrCloudflareAutomaticProxyOnly
	}

	return target, nil
}

func (d Deps) sync(ctx context.Context, id domain.TargetID, mode monad.Maybe[string], replace []string) (SyncResult, error) {
	result := SyncResult{Tasks: []Task{}, Conflicts: []Conflict{}, Outside: []string{}}

	conn, err := d.Connections.GetCloudflareConnection(ctx, id)

	if errors.Is(err, apperr.ErrNotFound) {
		return result, domain.ErrCloudflareNotConnected
	}

	if err != nil {
		return result, err
	}

	target, err := d.automaticTarget(ctx, id)

	if err != nil {
		return result, err
	}

	url := target.Url().MustGet()

	if value, isSet := mode.TryGet(); isSet {
		chosen, err := domain.CloudflareModeFrom(value)

		if err != nil {
			return result, err
		}

		conn.Mode = chosen
	}

	domains, err := d.Apps.GetDomainsOnTarget(ctx, id)

	if err != nil {
		return result, err
	}

	zones, err := d.Cloudflare.Zones(ctx, conn.Token)

	if err != nil {
		return result, err
	}

	type entry struct {
		domain.TargetDomain
		zone domain.CloudflareZone
	}

	var wanted []entry

	for _, item := range domains {
		if zone, found := domain.ZoneOf(item.Hostname, zones); found && zone.AccountID == conn.AccountID {
			wanted = append(wanted, entry{item, zone})
		} else {
			result.Outside = append(result.Outside, string(item.Hostname))
		}
	}

	// Where each domain's record points: the tunnel, or the environment's default address
	content := func(e entry) string {
		return string(e.AppName) + environmentSuffix(e.Environment) + "." + url.Hostname()
	}

	if conn.Mode == domain.CloudflareModeTunnel {
		tunnel, err := d.ensureTunnel(ctx, &conn)

		// Listing tunnels (all Connect can check) only needs Read: remember the token can't
		// create one, so the drawer offers what it can do.
		if errors.Is(err, domain.ErrCloudflarePermissionMissing) {
			saved, getErr := d.Connections.GetCloudflareConnection(ctx, id)
			if getErr == nil {
				saved.TunnelAllowed = false
				saved.Mode = domain.CloudflareModeDns
				_ = d.Connections.SaveCloudflareConnection(ctx, saved)
			}
		}

		if err != nil {
			return result, err
		}

		// Cloudflare decides: a token refused before may have been given Tunnel Edit since
		conn.TunnelAllowed = true
		result.Tasks = append(result.Tasks, Task{Key: "tunnel", Detail: tunnel.Name})

		ingress := make([]domain.CloudflareIngress, len(wanted))

		for i, e := range wanted {
			ingress[i] = domain.CloudflareIngress{
				Hostname:    string(e.Hostname),
				Service:     d.Connector.ProxyService(target),
				NoTLSVerify: url.UseSSL(),
			}
		}

		if err = d.Cloudflare.ConfigureTunnel(ctx, conn.Token, conn.AccountID, tunnel.ID, ingress); err != nil {
			return result, err
		}

		result.Tasks = append(result.Tasks, Task{Key: "routes", Detail: joinHostnames(wanted, func(e entry) string { return string(e.Hostname) })})

		content = func(entry) string { return tunnel.ID + ".cfargotunnel.com" }
	} else if conn.TunnelID.HasValue() {
		// Switched to DNS records: the tunnel goes
		if err = d.Connector.RemoveTunnelConnector(ctx, target); err != nil {
			return result, err
		}

		if err = d.Cloudflare.DeleteTunnel(ctx, conn.Token, conn.AccountID, conn.TunnelID.MustGet()); err != nil {
			return result, err
		}

		conn.TunnelID, conn.TunnelName = monad.None[string](), monad.None[string]()
	}

	if err = d.Connections.SaveCloudflareConnection(ctx, conn); err != nil {
		return result, err
	}

	// DNS records
	comment := domain.CloudflareRecordComment(id)
	kept := make(map[string]bool)

	for _, e := range wanted {
		hostname := string(e.Hostname)
		kept[hostname] = true

		existing, err := d.Cloudflare.Records(ctx, conn.Token, e.zone.ID, hostname)

		if err != nil {
			return result, err
		}

		var mine, others []domain.CloudflareRecord

		for _, record := range existing {
			switch {
			case record.Comment == comment:
				mine = append(mine, record)
			case record.Type == "A" || record.Type == "AAAA" || record.Type == "CNAME":
				others = append(others, record)
			}
		}

		want := domain.CloudflareRecord{Type: "CNAME", Name: hostname, Content: content(e), Proxied: true, Comment: comment}

		wantedRecord := Wanted{Type: want.Type, Name: relativeName(hostname, e.zone.Name), Content: want.Content, Proxied: true}
		conflict := func(reason string, existing []domain.CloudflareRecord) {
			c := Conflict{Hostname: hostname, Zone: e.zone.Name, Want: wantedRecord, Reason: reason}
			if len(existing) > 0 {
				c.Type, c.Content = existing[0].Type, existing[0].Content
			}
			result.Conflicts = append(result.Conflicts, c)
		}

		// A record the person made by hand as the drawer showed: checkship takes it over
		adopted := len(mine) == 0 && len(others) == 1 && sameTarget(others[0], want)

		if adopted {
			mine, others = others, nil
		}

		if len(others) > 0 {
			if !slices.Contains(replace, hostname) {
				conflict(ConflictRecord, others)
				continue
			}

			if err = d.deleteRecords(ctx, conn, e.zone.ID, others); errors.Is(err, domain.ErrCloudflarePermissionMissing) {
				conflict(ConflictDnsEdit, others)
				continue
			} else if err != nil {
				return result, err
			}
		}

		if len(mine) == 0 {
			err = d.Cloudflare.CreateRecord(ctx, conn.Token, e.zone.ID, want)
		} else if mine[0].Type != want.Type || mine[0].Content != want.Content || !mine[0].Proxied || mine[0].Comment != comment {
			want.ID = mine[0].ID
			err = d.Cloudflare.UpdateRecord(ctx, conn.Token, e.zone.ID, want)
		}

		// A token without DNS Edit: the person adds the record by hand, as the drawer shows. One
		// they made already works as long as it is proxied; checkship just can't mark it as its own.
		if errors.Is(err, domain.ErrCloudflarePermissionMissing) {
			if adopted && mine[0].Proxied {
				continue
			}
			conflict(ConflictDnsEdit, mine)
			continue
		}

		if err != nil {
			return result, err
		}

		for _, extra := range mine[min(1, len(mine)):] {
			if err = d.Cloudflare.DeleteRecord(ctx, conn.Token, e.zone.ID, extra.ID); err != nil {
				return result, err
			}
		}

	}

	// Only when a record is in place: a domain left to a conflict has none yet
	if set := joinHostnames(wanted, func(e entry) string {
		if kept[string(e.Hostname)] && !conflicting(result.Conflicts, string(e.Hostname)) {
			return string(e.Hostname)
		}
		return ""
	}); set != "" {
		result.Tasks = append(result.Tasks, Task{Key: "records", Detail: set})
	}

	// Records checkship created for domains that are gone
	removed, err := d.removeRecords(ctx, conn, zones, kept)

	if err != nil {
		return result, err
	}

	if removed > 0 {
		result.Tasks = append(result.Tasks, Task{Key: "cleanup", Detail: pluralRecords(removed)})
	}

	// The connector, last: everything it serves is configured
	if conn.Mode == domain.CloudflareModeTunnel {
		tunnelToken, err := d.Cloudflare.TunnelToken(ctx, conn.Token, conn.AccountID, conn.TunnelID.MustGet())

		if err != nil {
			return result, err
		}

		if err = d.Connector.RunTunnelConnector(ctx, target, tunnelToken); err != nil {
			return result, err
		}

		result.Tasks = append(result.Tasks, Task{Key: "connector", Detail: "cloudflared"})
	}

	result.Status, err = d.status(ctx, conn, zones)

	return result, err
}

func (d Deps) deleteRecords(ctx context.Context, conn domain.CloudflareConnection, zone string, records []domain.CloudflareRecord) error {
	for _, record := range records {
		if err := d.Cloudflare.DeleteRecord(ctx, conn.Token, zone, record.ID); err != nil {
			return err
		}
	}

	return nil
}

func (d Deps) ensureTunnel(ctx context.Context, conn *domain.CloudflareConnection) (domain.CloudflareTunnel, error) {
	if id, isSet := conn.TunnelID.TryGet(); isSet {
		tunnel, err := d.Cloudflare.Tunnel(ctx, conn.Token, conn.AccountID, id)

		if err == nil {
			return tunnel, nil
		}

		if !errors.Is(err, apperr.ErrNotFound) {
			return tunnel, err
		}
	}

	name := domain.CloudflareTunnelName(conn.Target)
	found, err := d.Cloudflare.FindTunnel(ctx, conn.Token, conn.AccountID, name)

	if err != nil {
		return domain.CloudflareTunnel{}, err
	}

	tunnel, isSet := found.TryGet()

	if !isSet {
		if tunnel, err = d.Cloudflare.CreateTunnel(ctx, conn.Token, conn.AccountID, name); err != nil {
			return tunnel, err
		}
	}

	conn.TunnelID.Set(tunnel.ID)
	conn.TunnelName.Set(tunnel.Name)

	return tunnel, nil
}

// Deletes the records checkship created for the target, except the kept hostnames.
func (d Deps) removeRecords(ctx context.Context, conn domain.CloudflareConnection, zones []domain.CloudflareZone, kept map[string]bool) (int, error) {
	comment := domain.CloudflareRecordComment(conn.Target)
	removed := 0

	for _, zone := range zones {
		if zone.AccountID != conn.AccountID {
			continue
		}

		records, err := d.Cloudflare.RecordsWithComment(ctx, conn.Token, zone.ID, comment)

		if err != nil {
			return removed, err
		}

		for _, record := range records {
			if kept[record.Name] || record.Comment != comment {
				continue
			}

			if err = d.Cloudflare.DeleteRecord(ctx, conn.Token, zone.ID, record.ID); err != nil {
				return removed, err
			}

			removed++
		}
	}

	return removed, nil
}

func (d Deps) removeAll(ctx context.Context, conn domain.CloudflareConnection) error {
	zones, err := d.Cloudflare.Zones(ctx, conn.Token)

	if err != nil {
		return err
	}

	if _, err = d.removeRecords(ctx, conn, zones, nil); err != nil {
		return err
	}

	if id, isSet := conn.TunnelID.TryGet(); isSet {
		return d.Cloudflare.DeleteTunnel(ctx, conn.Token, conn.AccountID, id)
	}

	return nil
}

func (d Deps) status(ctx context.Context, conn domain.CloudflareConnection, zones []domain.CloudflareZone) (Status, error) {
	connectedAt := conn.ConnectedAt
	status := Status{
		Connected:     true,
		AccountName:   conn.AccountName,
		AccountID:     conn.AccountID,
		TunnelAllowed: conn.TunnelAllowed,
		Mode:          string(conn.Mode),
		Zones:         []string{},
		Records:       []RecordStatus{},
		ConnectedAt:   &connectedAt,
	}

	comment := domain.CloudflareRecordComment(conn.Target)

	for _, zone := range zones {
		if zone.AccountID != conn.AccountID {
			continue
		}

		status.Zones = append(status.Zones, zone.Name)

		records, err := d.Cloudflare.RecordsWithComment(ctx, conn.Token, zone.ID, comment)

		if err != nil {
			return status, err
		}

		for _, record := range records {
			status.Records = append(status.Records, RecordStatus{Hostname: record.Name, Type: record.Type, Content: record.Content})
		}
	}

	if id, isSet := conn.TunnelID.TryGet(); isSet {
		if tunnel, err := d.Cloudflare.Tunnel(ctx, conn.Token, conn.AccountID, id); err == nil {
			status.Tunnel = &TunnelStatus{Name: tunnel.Name, Status: tunnel.Status, Connections: tunnel.Connections}
		}
	}

	return status, nil
}

func environmentSuffix(env domain.Environment) string {
	if env.IsProduction() {
		return ""
	}

	return "-" + string(env)
}

// A refused token or a missing permission: shown under the token field.
func tokenField(err error) error {
	if errors.Is(err, domain.ErrCloudflareTokenRefused) || errors.Is(err, domain.ErrCloudflarePermissionMissing) {
		return validate.NewError(validate.FieldErrors{"token": err})
	}

	return err
}

func refused(err error) bool {
	return errors.Is(err, domain.ErrCloudflareTokenRefused) || errors.Is(err, domain.ErrCloudflarePermissionMissing)
}

type missingPermission struct {
	Permission string `json:"permission"`
}

func (m missingPermission) Error() string { return m.Permission }

func missing(permission string) error {
	return apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), missingPermission{Permission: permission})
}

func conflicting(conflicts []Conflict, hostname string) bool {
	for _, c := range conflicts {
		if c.Hostname == hostname {
			return true
		}
	}

	return false
}

// Whether a record already sends the name where checkship would (a CNAME to the same target).
func sameTarget(record, want domain.CloudflareRecord) bool {
	return record.Type == want.Type &&
		strings.EqualFold(strings.TrimSuffix(record.Content, "."), strings.TrimSuffix(want.Content, "."))
}

// The name as Cloudflare's form wants it: "@" for the zone, else the labels before it.
func relativeName(hostname, zone string) string {
	if hostname == zone {
		return "@"
	}

	return strings.TrimSuffix(hostname, "."+zone)
}

func joinHostnames[T any](items []T, name func(T) string) string {
	var names []string

	for _, item := range items {
		if n := name(item); n != "" {
			names = append(names, n)
		}
	}

	return strings.Join(names, ", ")
}

func pluralRecords(n int) string {
	if n == 1 {
		return "1 record"
	}

	return strconv.Itoa(n) + " records"
}
