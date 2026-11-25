import { expect, test, type Page } from '@playwright/test';

/**
 * New application › From a template, against the real Go API: the menu, the gallery's search, a
 * template that must be configured before it deploys (Gotify), then Deploy creating the
 * application and queueing its first deployment. The target is a remote one whose configuration
 * fails here, which doesn't matter: only the requests and the redirect are checked.
 */

const ADMIN = { email: 'admin@example.com', password: 'admin' };

async function signIn(page: Page, path = '/') {
	await page.goto(`/signin/?redirectTo=${encodeURIComponent(path)}`);
	await page.getByLabel('Email').fill(ADMIN.email);
	await page.getByLabel('Password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page).toHaveURL(new RegExp(`${path.replaceAll('/', '\\/')}$`));
}

async function target(page: Page) {
	const targets = await (await page.request.get('/api/v1/targets')).json();
	const existing = targets.find((t: { name: string }) => t.name === 'templates-target');
	if (existing) return existing.id;
	const created = await page.request.post('/api/v1/targets', {
		data: { name: 'templates-target', docker: { host: 'templates.e2e.invalid', user: 'e2e' } }
	});
	expect(created.ok()).toBe(true);
	return (await created.json()).id;
}

test.describe('templates', () => {
	test.use({ viewport: { width: 1440, height: 900 } });

	test('From scratch still opens the application form', async ({ page }) => {
		await signIn(page, '/apps/');
		await page.getByRole('button', { name: 'New application' }).click();
		await page.getByRole('menuitem', { name: /From scratch/ }).click();
		await expect(page).toHaveURL(/\/apps\/new\/$/);
	});

	test('Gotify asks for its admin before it deploys, then opens deployment #1', async ({
		page
	}) => {
		await signIn(page, '/apps/');
		await target(page);
		await page.reload();

		await page.getByRole('button', { name: 'New application' }).click();
		await page.getByRole('menuitem', { name: /From a template/ }).click();
		await expect(page).toHaveURL(/\/apps\/templates\/$/);
		await expect(page.getByRole('heading', { name: 'Templates' })).toBeVisible();

		await page.getByRole('searchbox', { name: 'Search templates' }).fill('goti');
		const rows = page.getByRole('list', { name: 'Templates' }).getByRole('listitem');
		await expect(rows).toHaveCount(1);
		await expect(rows.first()).toContainText('Needs setup');

		await page.getByRole('button', { name: /^Gotify,/ }).click();
		const drawer = page.getByRole('dialog', { name: 'Deploy Gotify' });
		await expect(drawer).toBeVisible();
		await drawer.getByLabel('Name').fill('e2e-gotify');

		// Deploy is held back: both required fields say why.
		const deploy = page.getByRole('button', { name: 'Deploy', exact: true });
		await deploy.click();
		await expect(page.getByText('Fill the 2 required settings')).toBeVisible();
		await expect(drawer.getByText('Required.')).toHaveCount(2);

		await drawer.getByLabel('Admin username').fill('ops');
		await drawer.getByLabel('Admin password').fill('short');
		await drawer.getByLabel('Admin password').blur();
		await expect(drawer.getByText('Use at least 8 characters.')).toBeVisible();
		await drawer.getByRole('button', { name: 'Generate' }).click();
		await expect(drawer.getByLabel('Admin password')).toHaveValue(/^[0-9a-f]{20}$/);

		const created = page.waitForResponse(
			(r) => r.url().endsWith('/api/v1/apps') && r.request().method() === 'POST'
		);
		const queued = page.waitForResponse(
			(r) => /\/api\/v1\/apps\/[^/]+\/deployments$/.test(r.url()) && r.request().method() === 'POST'
		);
		await deploy.click();

		const app = await (await created).json();
		expect(app.name).toBe('e2e-gotify');
		expect(app.production.vars.gotify.GOTIFY_DEFAULTUSER_NAME).toBe('ops');
		expect(app.staging.vars.gotify.GOTIFY_DEFAULTUSER_PASS).toMatch(/^[0-9a-f]{20}$/);
		const deployment = await (await queued).json();
		expect(deployment.source.discriminator).toBe('raw');
		expect(deployment.source.data).toContain('image: gotify/server:3.1.1');

		await expect(page).toHaveURL(new RegExp(`/apps/${app.id}/deployments/1/$`));
	});

	test('a taken name stops Deploy before the server does', async ({ page }) => {
		await signIn(page, '/apps/templates/');
		const targetId = await target(page);
		await page.request.post('/api/v1/apps', {
			data: { name: 'taken-kuma', production: { target: targetId }, staging: { target: targetId } }
		});
		await page.reload();

		await page.getByRole('button', { name: /^Uptime Kuma,/ }).click();
		const drawer = page.getByRole('dialog', { name: 'Deploy Uptime Kuma' });
		await drawer.getByRole('combobox').click();
		await page.getByRole('option', { name: /templates-target/ }).click();
		await drawer.getByLabel('Name').fill('taken-kuma');
		await expect(drawer.getByText('App name is already taken on this target')).toBeVisible();
		await expect(page.getByText('Fix the name to deploy')).toBeVisible();
	});
});
