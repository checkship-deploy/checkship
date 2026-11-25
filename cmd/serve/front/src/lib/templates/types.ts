import type { AppTranslationsString } from '#lib/localization/index.ts';

/** Where a template sits in the gallery's Category filter. */
export type TemplateCategory =
	| 'monitoring'
	| 'notes'
	| 'developer'
	| 'security'
	| 'notifications'
	| 'database'
	| 'email'
	| 'search'
	| 'storage'
	| 'messaging'
	| 'auth'
	| 'backend';

/**
 * One setting of a template, saved as an app variable of the template's service. A raw compose
 * file gets no interpolation from checkship (`provider/docker/deployment.go`, loadProject), so
 * every value the person or checkship chooses travels as an app variable, which overrides the
 * compose file's `environment`.
 *
 * - `required`: the person types it; Deploy stays off until it is set.
 * - `generated`: a random hex secret, made in the browser, shown with copy and regenerate.
 * - `derived`: computed from the app's URL. Without a target URL it becomes a required input.
 * - `optional`: a switch with a default.
 */
export type TemplateField = {
	key: string;
	kind: 'required' | 'generated' | 'derived' | 'optional';
	label: AppTranslationsString;
	help?: AppTranslationsString;
	/** required: a secret, with a Generate button. */
	secret?: boolean;
	/** required: the shortest accepted value. */
	minLength?: number;
	/** required: an email address, which some images check before they start (pgAdmin). */
	email?: boolean;
	/** generated (and required secrets' Generate): how many hex characters. */
	length?: number;
	/**
	 * generated: text the variable carries before the secret, for images that read a secret
	 * inside a larger value: Mailpit's `admin:<password>`, Valkey's `--requirepass <password>`.
	 * Copy and the connection URL use the secret alone.
	 */
	prefix?: string;
	/**
	 * generated: mix lower and upper case letters, digits and URL-safe symbols (-_.~) rather than
	 * hex, for images that refuse a weaker password (OpenObserve).
	 */
	strong?: boolean;
	/** derived: the app URL as is, or with a trailing slash. */
	derive?: 'url' | 'url-slash';
	/** optional: the switch's starting position. */
	default?: boolean;
};

/**
 * One thing a person signs in or connects with, read from the app's variables: a user name, a
 * password, a token. `value` takes `{FIELD_KEY}` (that field's secret, without its prefix),
 * `{url}` (the app's URL) and `{ws}` (the same URL as ws:// or wss://) placeholders; anything else
 * is shown as written.
 */
export type TemplateCredential = {
	label: AppTranslationsString;
	value: string;
	/** Masked until revealed. */
	secret?: boolean;
};

export type Template = {
	/** Also the default app name and the logo's file name. */
	id: string;
	name: string;
	category: TemplateCategory;
	description: AppTranslationsString;
	/** What to do once it runs (create the admin, open the installer). */
	after: AppTranslationsString;
	/** The web UI needs a secure context: warn on an http target that is not *.localhost. */
	needsSecureContext?: boolean;
	/** Image pinned in the compose file, shown next to the name. */
	image: string;
	/** The compose service the fields are saved for, and that gets the app's URL. */
	service: string;
	/** A raw compose file: `image:` only (a raw source has no build context), one service. */
	compose: string;
	fields: TemplateField[];
	/** The project's own SVG, served from static/templates/. */
	logo: string;
	/** A variant for dark mode when the mark disappears on a dark ground. */
	logoDark?: string;
	/**
	 * Dark mode only: draw the logo on a white plate. For a dark mark the project publishes no
	 * light variant of (Valkey's navy): the plate keeps it legible without recolouring it.
	 */
	logoPlate?: boolean;
	/**
	 * A service the person's own applications use (a database, a cache, SMTP, an API): how they
	 * reach it on the target's network, at `<service>.<app>` (the alias checkship gives every
	 * exposed service, `config_snapshot.go` InternalHost). `url` is a connection string with
	 * `{host}` and `{FIELD_KEY}` placeholders, filled with the secret alone.
	 */
	connect?: { port: number; url?: string };
	/** What its web page asks for when you open it (Mailpit's sign-in, Keycloak's admin). */
	credentials?: TemplateCredential[];
	/**
	 * The image reads its sign-in from the variables only on its first start: once created, the
	 * account is changed in the app, and a new value in the variables does nothing.
	 */
	firstStartOnly?: boolean;
	/**
	 * Its web port carries no page, only a protocol (Mosquitto's MQTT over WebSockets): lists say
	 * "No web page" and the drawer shows addresses rather than a Visit link that opens nothing.
	 */
	noWebPage?: boolean;
	/** Keeps nothing worth a volume (Centrifugo's memory engine, phpMyAdmin): none is declared. */
	stateless?: boolean;
	/** Who owns the logo and under which license, shown under the drawer. */
	logoCredit: string;
	website: string;
	docs: string;
};
