/*
 * Candidate-Hub browser acceptance for P015. This is intentionally an
 * environment-driven smoke test: it uses disposable synthetic accounts and a
 * Hub that has already been seeded by the P015 fixture setup.
 *
 * Example:
 *   NODE_PATH=/tmp/p015-browser/node_modules \
 *   P015_CHROMIUM_PATH=/usr/bin/chromium \
 *   P015_HUB_URL=http://127.0.0.1:8097 \
 *   P015_ADMIN_EMAIL=p015-admin@example.invalid \
 *   P015_OUTSIDER_EMAIL=p015-outsider@example.invalid \
 *   P015_SYNTHETIC_PASSWORD='...' \
 *   P015_SYSTEM_ID=... P015_OUTSIDER_SYSTEM_ID=... \
 *   node tests/browser/p015-frontend-acceptance.cjs
 */

const assert = require("node:assert/strict")
const { chromium } = require(process.env.PLAYWRIGHT_CORE_MODULE || "playwright-core")

const hubUrl = (process.env.P015_HUB_URL || "http://127.0.0.1:8097").replace(/\/$/, "")
const adminEmail = process.env.P015_ADMIN_EMAIL
const outsiderEmail = process.env.P015_OUTSIDER_EMAIL
const password = process.env.P015_SYNTHETIC_PASSWORD
const systemId = process.env.P015_SYSTEM_ID
const outsiderSystemId = process.env.P015_OUTSIDER_SYSTEM_ID
const systemName = process.env.P015_SYSTEM_NAME || "p015-synthetic-system"
const outsiderSystemName = process.env.P015_OUTSIDER_SYSTEM_NAME || "p015-outsider-system"

for (const [name, value] of Object.entries({ adminEmail, outsiderEmail, password, systemId, outsiderSystemId })) {
	assert.ok(value, `missing required environment variable for ${name}`)
}

const errors = { page: [], console: [], request: [] }
const jsonUrl = (path) => `${hubUrl}${path}`

function observe(page) {
	page.on("pageerror", (error) => errors.page.push(error.message))
	page.on("console", (message) => {
		if (message.type() === "error") errors.console.push(message.text())
	})
	page.on("requestfailed", (request) => errors.request.push(`${request.method()} ${request.url()}`))
}

async function signIn(page, email) {
	await page.goto(jsonUrl("/"), { waitUntil: "domcontentloaded" })
	await page.locator("#email").waitFor()
	await page.locator("#email").fill(email)
	await page.locator("#pass").fill(password)
	await page.getByRole("button", { name: "Sign in" }).click()
	await page.getByRole("button", { name: "User Actions" }).waitFor()
}

async function selectLocale(page, locale, label) {
	await page.locator("#lang").click()
	if (locale === "en") {
		await page.getByRole("option", { name: /English/ }).last().click()
	} else {
		await page.getByRole("option", { name: label, exact: false }).click()
	}
	await page.waitForFunction((expected) => document.documentElement.lang === expected, locale)
}

async function main() {
	const browser = await chromium.launch({
		headless: true,
		executablePath: process.env.P015_CHROMIUM_PATH || undefined,
		args: ["--no-sandbox"],
	})
	const result = { hub_url: hubUrl, routes: {}, localization: {}, security: {}, browser_errors: errors }
	try {
		const detectedLocaleContext = await browser.newContext({ locale: "vi-VN" })
		const detectedLocalePage = await detectedLocaleContext.newPage()
		detectedLocalePage.setDefaultTimeout(8000)
		observe(detectedLocalePage)
		await detectedLocalePage.goto(jsonUrl("/"), { waitUntil: "domcontentloaded" })
		await detectedLocalePage.waitForFunction(() => document.documentElement.lang === "vi")
		await detectedLocalePage.waitForFunction(() => document.body.innerText.includes("Đăng nhập"))
		result.localization.browser_language_detection = await detectedLocalePage.evaluate(() => ({
			language: document.documentElement.lang,
			unicode: document.body.innerText.includes("Đăng nhập"),
		}))
		assert.equal(result.localization.browser_language_detection.unicode, true)
		await detectedLocaleContext.close()

		const invalidLocaleContext = await browser.newContext({ locale: "en-US" })
		await invalidLocaleContext.addInitScript(() => localStorage.setItem("lang", "unsupported-p015"))
		const invalidLocalePage = await invalidLocaleContext.newPage()
		invalidLocalePage.setDefaultTimeout(8000)
		observe(invalidLocalePage)
		await invalidLocalePage.goto(jsonUrl("/"), { waitUntil: "domcontentloaded" })
		await invalidLocalePage.waitForFunction(
			() => document.documentElement.lang === "en" && localStorage.getItem("lang") === "en",
		)
		result.localization.invalid_stored_locale_falls_back_to_english = true
		await invalidLocaleContext.close()

		const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: "en-US" })
		const page = await context.newPage()
		page.setDefaultTimeout(8000)
		observe(page)

		for (const route of ["/forgot-password", "/request-otp"]) {
			const response = await page.goto(jsonUrl(route), { waitUntil: "domcontentloaded" })
			assert.equal(response.status(), 200, `unauthenticated route ${route}`)
			await page.locator("form").waitFor({ state: "visible" })
			assert.ok(await page.locator("form").count(), `form missing at ${route}`)
			result.routes[route] = response.status()
		}

		await page.goto(jsonUrl("/"), { waitUntil: "domcontentloaded" })
		await page.locator("#email").fill(adminEmail)
		await page.locator("#pass").fill(`${password}-wrong`)
		await page.getByRole("button", { name: "Sign in" }).click()
		await page.getByText("Login attempt failed", { exact: true }).waitFor()
		assert.equal(errors.page.length, 0, "rejected login produced an uncaught browser exception")
		result.auth = { failed_login_returns_to_form: "PASS", rejected_login_page_exceptions: 0 }

		await signIn(page, adminEmail)
		assert.match(await page.locator("body").innerText(), new RegExp(systemName))

		const routes = [
			["/", new RegExp(systemName)],
			["/containers", /p015-synthetic-container/],
			["/smart", /p015-synthetic-disk/],
			["/monitors", /monitor\.synthetic\.invalid/],
			[`/system/${systemId}`, new RegExp(systemName)],
			["/settings/general", /Preferred Language|Ngôn ngữ/],
			["/settings/alert-history", new RegExp(systemName)],
		]
		for (const [route, expected] of routes) {
			const response = await page.goto(jsonUrl(route), { waitUntil: "domcontentloaded" })
			assert.equal(response.status(), 200, `authenticated route ${route}`)
			await page.waitForFunction(
				({ source, flags }) => new RegExp(source, flags).test(document.body.innerText),
				{ source: expected.source, flags: expected.flags },
			)
			const content = await page.locator("body").innerText()
			assert.match(content, expected, `expected route content at ${route}`)
			result.routes[route] = response.status()
		}

		const nestedRefresh = await page.reload({ waitUntil: "domcontentloaded" })
		assert.equal(nestedRefresh.status(), 200)
		assert.match(page.url(), /\/settings\/alert-history$/)
		result.routes.nested_refresh = "PASS"

		await page.goto(jsonUrl("/"), { waitUntil: "domcontentloaded" })
		await page.getByRole("link", { name: "Network Monitors" }).click()
		await page.waitForURL(/\/monitors$/)
		await page.goBack()
		await page.waitForURL(/\/$/)
		result.routes.spa_navigation_and_history = "PASS"

		await page.goto(jsonUrl("/settings/general"), { waitUntil: "domcontentloaded" })
		await page.locator("#lang").waitFor()
		await selectLocale(page, "vi", "Tiếng Việt")
		assert.match(await page.locator("body").innerText(), /Đăng nhập|Ngôn ngữ|Ngôn ngữ ưu tiên/)
		await page.locator('form button[type="submit"]').click()
		await page.waitForTimeout(500)
		await page.reload({ waitUntil: "domcontentloaded" })
		await page.waitForFunction(() => document.documentElement.lang === "vi")
		result.localization.vietnamese = {
			language: await page.evaluate(() => document.documentElement.lang),
			unicode: (await page.locator("body").innerText()).includes("Tiếng Việt"),
			persisted_after_reload: true,
		}

		for (const [locale, label] of [["ar", "العربية"], ["fa", "فارسی"], ["he", "עברית"]]) {
			await selectLocale(page, locale, label)
			const state = await page.evaluate(() => ({
				lang: document.documentElement.lang,
				dir: document.documentElement.dir,
				navigation: getComputedStyle(document.querySelector('[aria-label="Containers"]')).direction,
			}))
			assert.equal(state.lang, locale)
			assert.equal(state.dir, "rtl")
			assert.equal(state.navigation, "rtl")
			result.localization[locale] = state
		}

		await page.getByRole("link", { name: "Network Monitors" }).click()
		await page.waitForURL(/\/monitors$/)
		await page.waitForTimeout(350)
		const rtlTable = await page.evaluate(() => ({
			dir: document.documentElement.dir,
			table: document.querySelector("table") ? getComputedStyle(document.querySelector("table")).direction : null,
			alert: document.querySelector('[role="alert"]') ? getComputedStyle(document.querySelector('[role="alert"]')).direction : null,
		}))
		assert.equal(rtlTable.dir, "rtl")
		assert.equal(rtlTable.table, "rtl")
		assert.equal(rtlTable.alert, "rtl")
		const addButton = page.locator("button:has(svg.lucide-plus)").last()
		await addButton.click()
		const monitorDialog = page.locator('[role="dialog"]').first()
		await monitorDialog.waitFor({ state: "visible" })
		const dialogDirection = await monitorDialog.evaluate((node) => getComputedStyle(node).direction)
		assert.equal(dialogDirection, "rtl")
		await page.keyboard.press("Escape")
		const monitorRow = page.getByRole("row").filter({ hasText: "monitor.synthetic.invalid" })
		await monitorRow.click()
		const monitorDetails = page.locator('[role="dialog"]').first()
		await monitorDetails.waitFor({ state: "visible" })
		assert.match(await monitorDetails.innerText(), new RegExp(systemName))
		result.routes.monitor_details = "PASS"
		await monitorDetails.getByRole("link", { name: systemName }).click()
		await page.waitForURL(new RegExp(`/system/${systemId}$`))
		await page.waitForTimeout(400)
		const charts = await page.locator("svg.recharts-surface").count()
		assert.ok(charts > 0, "system charts did not render in RTL")
		result.localization.rtl_components = { ...rtlTable, dialog: dialogDirection, chart_count: charts }

		await page.getByRole("link", { name: "Home" }).click()
		await page.waitForURL(/\/$/)
		await page.getByRole("link", { name: "Settings" }).click()
		await page.waitForURL(/\/settings\/general$/)
		await selectLocale(page, "en", "English")
		await page.locator('form button[type="submit"]').click()
		await page.waitForTimeout(400)
		result.localization.restore_english = await page.evaluate(() => document.documentElement.lang === "en")

		await page.goto(jsonUrl(`/system/${outsiderSystemId}`), { waitUntil: "domcontentloaded" })
		await page.waitForTimeout(400)
		assert.ok(!(await page.locator("body").innerText()).includes(outsiderSystemName))
		result.security.owner_cannot_view_nonmember_system = "PASS"

		const failedEndpoint = /\/api\/collections\/system_stats\/records/
		await page.route(failedEndpoint, (route) => route.abort("failed"))
		await page.goto(jsonUrl(`/system/${systemId}`), { waitUntil: "domcontentloaded" })
		await page.waitForTimeout(500)
		assert.ok(errors.request.some((entry) => failedEndpoint.test(entry)), "controlled failed API request was not exercised")
		assert.match(await page.locator("body").innerText(), new RegExp(systemName))
		await page.unroute(failedEndpoint)
		result.security.failed_api_request_fallback = "PASS"

		await page.getByRole("button", { name: "User Actions" }).click()
		await page.getByText("Log Out", { exact: true }).click()
		await page.locator("#email").waitFor()
		await signIn(page, outsiderEmail)
		const outsiderDashboard = await page.locator("body").innerText()
		assert.match(outsiderDashboard, new RegExp(outsiderSystemName))
		assert.ok(!outsiderDashboard.includes(systemName), "account state leaked between synthetic users")
		result.security.account_state_isolation = "PASS"
		await context.close()

		const expiryContext = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
		const expiryPage = await expiryContext.newPage()
		expiryPage.setDefaultTimeout(10000)
		observe(expiryPage)
		await signIn(expiryPage, adminEmail)
		const storageKey = await expiryPage.evaluate(() => {
			for (const key of Object.keys(localStorage)) {
				try {
					const value = JSON.parse(localStorage.getItem(key))
					if (value?.token && value?.record?.email) return key
				} catch {}
			}
			return null
		})
		assert.ok(storageKey, "PocketBase session was not stored")
		await expiryPage.evaluate((key) => {
			const session = JSON.parse(localStorage.getItem(key))
			const payload = btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) - 3600 }))
				.replace(/=/g, "")
				.replace(/\+/g, "-")
				.replace(/\//g, "_")
			session.token = `eyJhbGciOiJub25lIn0.${payload}.invalid-signature`
			localStorage.setItem(key, JSON.stringify(session))
		}, storageKey)
		await expiryPage.reload({ waitUntil: "domcontentloaded" })
		await expiryPage.locator("#email").waitFor({ timeout: 10000 })
		assert.ok(!(await expiryPage.locator("body").innerText()).includes(systemName))
		result.security.rejected_expired_session_returns_to_login = "PASS"
		await expiryContext.close()

		assert.equal(errors.page.length, 0, `unexpected browser exceptions: ${errors.page.join(" | ")}`)
		assert.ok(![...errors.console, ...errors.request].some((entry) => entry.includes(password)), "synthetic credential appeared in browser diagnostics")
		console.log(JSON.stringify({ status: "PASS", ...result }, null, 2))
	} finally {
		await browser.close()
	}
}

main().catch((error) => {
	console.error(error)
	process.exitCode = 1
})
