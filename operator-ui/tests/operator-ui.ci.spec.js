import { execFileSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";

if (process.env.OPERATOR_UI_TEST_SCENARIO === "remote" && !process.env.OPERATOR_UI_BACKEND_BASE_URL) {
  throw new Error("OPERATOR_UI_BACKEND_BASE_URL is required when OPERATOR_UI_TEST_SCENARIO=remote");
}

const backendBaseURL =
  process.env.OPERATOR_UI_BACKEND_BASE_URL ?? `http://127.0.0.1:${process.env.OPERATOR_UI_BACKEND_PORT ?? "10000"}`;
const expectedVersionSHA = process.env.OPERATOR_UI_EXPECT_VERSION_SHA;
const delegatedDownloadExpectation = process.env.OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD ?? "0";
const captureArtifacts = process.env.OPERATOR_UI_CAPTURE_ARTIFACTS === "1";
const artifactDir = process.env.OPERATOR_UI_ARTIFACT_DIR ?? "./test-results";
const authEnabled = process.env.OPERATOR_UI_TEST_AUTH === "1";
const authMockUserID = process.env.OPERATOR_UI_AUTH_MOCK_USER_ID ?? "operator-user01";
const authMockLogin = process.env.OPERATOR_UI_AUTH_MOCK_LOGIN ?? authMockUserID;
const authSignupUserID = process.env.OPERATOR_UI_AUTH_SIGNUP_USER_ID ?? "operator-signup-user01";
const authSignupLogin = process.env.OPERATOR_UI_AUTH_SIGNUP_LOGIN ?? authSignupUserID;
const frontendHost = process.env.OPERATOR_UI_FRONTEND_HOST ?? "127.0.0.1";
const frontendPort = process.env.OPERATOR_UI_FRONTEND_PORT ?? "4173";
const testDir = path.dirname(fileURLToPath(import.meta.url));
const bundleFixtureDir = process.env.OPERATOR_UI_BUNDLE_FIXTURE_DIR ?? path.resolve(testDir, "../../.local/operator-ui-game-bundles");
const bundlePath = (name) => path.resolve(bundleFixtureDir, `${name}.arena-bundle.zip`);
const bundleFamilies = [
  {
    gameID: "echo-count",
    gameVersion: "2.0.0",
    rulesetVersion: "phase2-simultaneous-3turn",
    gameBundle: bundlePath("echo-count"),
    alphaBundle: bundlePath("echo-ai-alpha"),
    revisionBundle: bundlePath("echo-ai-revision"),
    betaBundle: bundlePath("echo-ai-beta"),
  },
  {
    gameID: "janken",
    gameVersion: "2.1.0",
    rulesetVersion: "regular",
    gameBundle: bundlePath("janken"),
    alphaBundle: bundlePath("janken-ai-alpha"),
    revisionBundle: bundlePath("janken-ai-revision"),
    betaBundle: bundlePath("janken-ai-beta"),
  },
];

test.setTimeout(240_000);

test("remote read-only smoke verifies version, anonymous session, and operator login redirect", async ({ page, request }) => {
  test.skip(process.env.OPERATOR_UI_TEST_SCENARIO !== "remote", "remote-only scenario");
  if (!expectedVersionSHA) {
    throw new Error("OPERATOR_UI_EXPECT_VERSION_SHA is required when OPERATOR_UI_TEST_SCENARIO=remote");
  }

  const version = await request.get(`${backendBaseURL}/version`);
  expect(version.status()).toBe(200);
  expect(await version.json()).toEqual({ version_sha: expectedVersionSHA });

  const health = await request.get(`${backendBaseURL}/healthz`);
  expect(health.status()).toBe(200);
  expect(await health.json()).toEqual({ api: "OK", worker: "OK" });

  const session = await request.get(`${backendBaseURL}/auth/session`);
  expect(session.status()).toBe(200);
  expect(await session.json()).toEqual({ auth_mode: "enabled", authenticated: false });

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
  await page.goto("/operator");
  await expect(page).toHaveURL(/\/login\?return_to=/);
  await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
  await expect(page.getByText("Session check failed")).toHaveCount(0);
  await expect(page.getByText("Auth Error")).toHaveCount(0);
});

test("auth-enabled signup lane bootstraps a signup-only GitHub user via invite", async ({ page }) => {
  test.skip(!authEnabled, "auth-only scenario");

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
  await expect
    .poll(async () =>
      page.evaluate(async () => {
        const response = await fetch("/auth/session", { credentials: "include" });
        return response.json();
      }),
    )
    .toMatchObject({
      auth_mode: "enabled",
      authenticated: false,
    });

  const invite = createSignupInvite();
  await page.goto(invite.invite_url);
  await expect(page.getByText("Invite token detected.")).toBeVisible();
  await page.getByRole("link", { name: "Continue with GitHub" }).click();
  await expect(page.getByRole("heading", { name: "GitHub OAuth Test Double" })).toBeVisible();
  await page.getByLabel("User ID").fill(authSignupUserID);
  await page.getByRole("button", { name: "Login" }).click();
  await expect
    .poll(async () =>
      page.evaluate(async () => {
        const response = await fetch("/auth/session", { credentials: "include" });
        return response.json();
      }),
    )
    .toMatchObject({
      auth_mode: "enabled",
      authenticated: true,
      principal: { provider_login: authSignupLogin, roles: ["operator"] },
    });
  await expect(page).toHaveURL(/\/($|operator$)/);
  await expect(page.getByText(`Signed in as @${authSignupLogin}`)).toBeVisible();
  await page.getByRole("button", { name: "Logout" }).click();
  await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
});

test("service-backed operator UI browser lane covers registration, request execution, ranking correction, and artifact access", async ({
  context,
  page,
  request,
}) => {
  test.skip(process.env.OPERATOR_UI_TEST_SCENARIO === "remote", "remote lane is limited to read-only smoke");
  test.skip(!authEnabled, "general bundle admission requires the authenticated operator lane");
  if (captureArtifacts) {
    await context.tracing.start({ screenshots: true, snapshots: true, sources: true });
  }

  const api = authEnabled ? createBrowserAPI(page) : createRequestAPI(request);
  const health = await request.get(`${backendBaseURL}/healthz`);
  expect(health.status()).toBe(200);
  const healthBody = await health.json();
  expect(healthBody).toMatchObject({ api: "OK" });
  expect(["OK", "NOT_READY"]).toContain(healthBody.worker);

  await page.goto("/");

  if (authEnabled) {
    await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
    await page.getByRole("link", { name: "Continue with GitHub" }).click();
    await expect(page.getByRole("heading", { name: "GitHub OAuth Test Double" })).toBeVisible();
    await page.getByLabel("User ID").fill(authMockUserID);
    await page.getByRole("button", { name: "Login" }).click();
    await expect
      .poll(async () =>
        page.evaluate(async () => {
          const response = await fetch("/auth/session", { credentials: "include" });
          return response.json();
        }),
      )
      .toMatchObject({
        auth_mode: "enabled",
        authenticated: true,
        principal: { provider_login: authMockLogin },
      });
    await expect(page).toHaveURL(/\/($|operator$)/);
    await expect(page.getByText(`Signed in as @${authMockLogin}`)).toBeVisible();
  }

  await expect(page.getByRole("heading", { name: "AI Arena Operator Console" })).toBeVisible();
  await expect(page.getByTestId("operator-nav-invites")).toBeVisible();
  await expect(page.getByTestId("operator-nav-overview")).toBeVisible();
  await expect(page.getByTestId("operator-nav-games")).toBeVisible();
  await expect(page.getByTestId("operator-nav-submissions")).toBeVisible();
  await expect(page.getByTestId("operator-nav-requests")).toBeVisible();
  await expect(page.getByTestId("operator-nav-rankings")).toBeVisible();

  await page.getByTestId("operator-nav-invites").click();
  await expect(page.getByTestId("operator-form-invites")).toBeVisible();
  if (authEnabled) {
    await page.getByLabel("Role").selectOption("developer");
    await page.getByLabel("TTL").fill("12h");
    await page.getByRole("button", { name: "Create invite" }).click();
    await expect(page.getByTestId("signup-invite-result")).toBeVisible();
    await expect(page.getByTestId("signup-invite-role")).toHaveText("developer");
    await expect(page.getByTestId("signup-invite-token")).toHaveText(/.+/);
    await expect(page.getByTestId("signup-invite-url")).toHaveAttribute("href", /\/login\?invite_token=/);
  }

  for (const family of bundleFamilies) {
    await runBundleAdmissionFlow(page, api, family);
  }

  if (captureArtifacts) {
    await page.screenshot({
      fullPage: true,
      path: path.join(artifactDir, "completed-detail.png"),
    });
    await context.tracing.stop({ path: path.join(artifactDir, "operator-ui-flow.zip") });
  }

  if (authEnabled) {
    await page.getByRole("button", { name: "Logout" }).click();
    await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
    await page.goto("/operator");
    await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
  }
});

async function runBundleAdmissionFlow(page, api, family) {
  const registrationID = `${family.gameID}-v${family.gameVersion.split(".")[0]}-${family.rulesetVersion}`;

  await page.getByTestId("operator-nav-games").click();
  await page.getByLabel("Game bundle ZIP").setInputFiles(family.gameBundle);
  await page.getByRole("button", { name: "Upload game bundle" }).click();
  await expect(page.getByTestId("game-bundle-admission")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByTestId("admitted-game-id")).toHaveText(family.gameID);
  await expect(page.getByTestId("admitted-game-version")).toHaveText(family.gameVersion);
  const gameArtifactID = await page.getByTestId("admitted-artifact-id").textContent();
  expect(gameArtifactID).toMatch(/^sha256:[0-9a-f]{64}$/);
  await page.getByLabel("Ruleset Version").selectOption(family.rulesetVersion);
  await page.getByRole("button", { name: "Activate game" }).click();
  await expect(page.getByTestId(`game-row-${registrationID}`)).toContainText(gameArtifactID);

  await page.getByTestId("operator-nav-submissions").click();
  await page.getByLabel("Competition scope").fill(registrationID);
  await page.getByLabel("Bot name").fill(`${family.gameID} Alpha`);
  await page.getByLabel("AI bundle ZIP").setInputFiles(family.alphaBundle);
  await page.getByRole("button", { name: "Upload AI bundle" }).click();
  await expect(page.getByTestId("ai-bundle-admission")).toBeVisible({ timeout: 30_000 });
  const alphaArtifactID = await page.getByTestId("admitted-ai-artifact-id").textContent();
  expect(alphaArtifactID).toMatch(/^sha256:[0-9a-f]{64}$/);
  await page.getByRole("button", { name: "Save bot revision" }).click();
  const bots = await waitForRecord(api, async () => {
    const items = await listItems(api, `${backendBaseURL}/api/v1/bots?scope_id=${encodeURIComponent(registrationID)}`);
    return items.length === 1 ? items : null;
  }, "first admitted bot");
  const alphaBot = bots[0];

  await page.getByLabel("Existing bot ID").fill(alphaBot.bot_id);
  await page.getByLabel("AI bundle ZIP").setInputFiles(family.revisionBundle);
  await page.getByRole("button", { name: "Upload AI bundle" }).click();
  await expect(page.getByTestId("ai-bundle-admission")).toBeVisible({ timeout: 30_000 });
  const revisionArtifactID = await page.getByTestId("admitted-ai-artifact-id").textContent();
  expect(revisionArtifactID).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(revisionArtifactID).not.toBe(alphaArtifactID);
  await page.getByRole("button", { name: "Save bot revision" }).click();
  await expect(page.getByTestId("ai-bundle-admission")).toHaveCount(0);

  await page.getByLabel("Existing bot ID").fill("");
  await page.getByLabel("Bot name").fill(`${family.gameID} Beta`);
  await page.getByLabel("AI bundle ZIP").setInputFiles(family.betaBundle);
  await page.getByRole("button", { name: "Upload AI bundle" }).click();
  await expect(page.getByTestId("ai-bundle-admission")).toBeVisible({ timeout: 30_000 });
  const betaArtifactID = await page.getByTestId("admitted-ai-artifact-id").textContent();
  expect(betaArtifactID).toMatch(/^sha256:[0-9a-f]{64}$/);
  await page.getByRole("button", { name: "Save bot revision" }).click();
  const activeBots = await waitForRecord(api, async () => {
    const items = await listItems(api, `${backendBaseURL}/api/v1/bots?scope_id=${encodeURIComponent(registrationID)}`);
    return items.length === 2 ? items : null;
  }, "two admitted bots");

  await page.getByTestId("operator-nav-requests").click();
  await page.getByLabel("Competition scope").selectOption(registrationID);
  await expect(page.getByText("Selected seats (2/2)")).toBeVisible();
  await page.getByRole("button", { name: "Create match request" }).click();
  const createdRequest = await waitForRequest(api, registrationID);
  const initialRun = await waitForRunState(api, createdRequest.latest_run_id, "completed");
  expect(initialRun.game_id).toBe(family.gameID);
  expect(initialRun.game_version).toBe(family.gameVersion);
  expect(initialRun.players.map((player) => player.artifact_id).sort()).toEqual([revisionArtifactID, betaArtifactID].sort());
  expect(initialRun.players.map((player) => player.bot_id).sort()).toEqual(activeBots.map((bot) => bot.bot_id).sort());

  await page.goto(`/operator/runs/${initialRun.run_id}`);
  await page.getByTestId("run-action-rerun").click();
  const rerunRequest = await waitForLatestRunChange(api, createdRequest.request_id, initialRun.run_id);
  const rerunRun = await waitForRunState(api, rerunRequest.latest_run_id, "completed");
  await page.goto(`/operator/runs/${rerunRun.run_id}`);
  await page.getByTestId("run-action-promote").click();
  await expect.poll(async () => getRunDetail(api, rerunRun.run_id)).toMatchObject({ run_id: rerunRun.run_id, official: true });
  await waitForRecord(api, async () => {
    const response = await api.getJSON(
      `${backendBaseURL}/api/v1/rankings?game_id=${encodeURIComponent(family.gameID)}&game_version=${encodeURIComponent(family.gameVersion)}&ruleset_version=${encodeURIComponent(family.rulesetVersion)}`,
    );
    return response.ok ? response.json : null;
  }, `ranking snapshot for ${family.gameID}`);
  const resultSummaryArtifact = page.getByTestId("artifact-entry-result-summary");
  await expect(resultSummaryArtifact).toBeVisible();
  const downloadLink = resultSummaryArtifact.getByRole("link", { name: "open delegated download" });
  const expectsDelegatedDownload =
    delegatedDownloadExpectation === "auto"
      ? (rerunRun.result_summary_path ?? "").startsWith("s3://")
      : delegatedDownloadExpectation === "1";
  if (expectsDelegatedDownload) {
    await expect(downloadLink).toBeVisible();
    await expect(downloadLink).toHaveAttribute("href", /http:\/\//);
  } else {
    await expect(downloadLink).toHaveCount(0);
  }

  await page.getByTestId("operator-nav-rankings").click();
  await expect(page.getByTestId("operator-form-rankings")).toContainText("ready");
  await page.getByLabel("Game ID").fill(family.gameID);
  await page.getByLabel("Game Version").fill(family.gameVersion);
  await page.getByLabel("Ruleset Version").fill(family.rulesetVersion);
  await page.getByRole("button", { name: "Load ranking snapshot" }).click();
  for (const bot of activeBots) {
    await expect(page.getByTestId(`ranking-entry-${encodeURIComponent(bot.bot_id)}`)).toBeVisible();
  }
}

async function waitForRequest(api, registrationID) {
  return waitForRecord(api, async () => {
    const items = await listItems(api, `${backendBaseURL}/api/v1/match-requests`);
    return items.find((item) => item.game_registration_id === registrationID);
  }, "created match request");
}

async function waitForLatestRunChange(api, requestID, previousRunID) {
  return waitForRecord(api, async () => {
    const items = await listItems(api, `${backendBaseURL}/api/v1/match-requests`);
    const item = items.find((candidate) => candidate.request_id === requestID);
    if (!item || item.latest_run_id === previousRunID) {
      return null;
    }
    return item;
  }, "rerun latest run id update");
}

async function waitForRunState(api, runID, lifecycleState) {
  return waitForRecord(api, async () => {
    const detail = await getRunDetail(api, runID);
    return detail.lifecycle_state === lifecycleState ? detail : null;
  }, `run ${runID} reaching ${lifecycleState}`);
}

async function getRunDetail(api, runID) {
  const response = await api.getJSON(`${backendBaseURL}/api/v1/runs/${runID}`);
  expect(response.ok).toBeTruthy();
  return response.json;
}

async function listItems(api, url) {
  const response = await api.getJSON(url);
  expect(response.ok).toBeTruthy();
  const payload = response.json;
  return payload.items ?? [];
}

async function waitForRecord(api, probe, description) {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    const record = await probe(api);
    if (record) {
      return record;
    }
    await pageWait(500);
  }
  throw new Error(`timed out waiting for ${description}`);
}

function pageWait(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function createRequestAPI(request) {
  return {
    async getJSON(url) {
      const response = await request.get(url);
      return {
        ok: response.ok(),
        status: response.status(),
        json: await response.json(),
      };
    },
    async postJSON(url, data) {
      const response = await request.post(url, { data });
      return {
        ok: response.ok(),
        status: response.status(),
        json: await response.json(),
      };
    },
  };
}

function createBrowserAPI(page) {
  const usesRemoteServers = process.env.OPERATOR_UI_TEST_SCENARIO === "remote";
  return {
    async getJSON(url) {
      const target = new URL(url);
      const fetchTarget = usesRemoteServers ? target.toString() : `${target.pathname}${target.search}`;
      return page.evaluate(async (requestURL) => {
        const response = await fetch(requestURL, { credentials: "include" });
        return {
          ok: response.ok,
          status: response.status,
          json: await response.json(),
        };
      }, fetchTarget);
    },
    async postJSON(url, data) {
      const target = new URL(url);
      const fetchTarget = usesRemoteServers ? target.toString() : `${target.pathname}${target.search}`;
      return page.evaluate(async ({ requestURL, body }) => {
        const response = await fetch(requestURL, {
          method: "POST",
          credentials: "include",
          headers: { "content-type": "application/json" },
          body: JSON.stringify(body),
        });
        return {
          ok: response.ok,
          status: response.status,
          json: await response.json(),
        };
      }, { requestURL: fetchTarget, body: data });
    },
  };
}

function createSignupInvite() {
  const scriptPath = path.resolve(testDir, "../../tools/dev/local-invite-url.sh");
  const stdout = execFileSync(scriptPath, {
    cwd: path.resolve(testDir, "../.."),
    env: {
      ...process.env,
      LOCAL_AUTH_FRONTEND_ORIGIN: `http://${frontendHost}:${frontendPort}`,
    },
    encoding: "utf8",
  });
  const payloadStart = stdout.indexOf("{");
  if (payloadStart === -1) {
    throw new Error("signup invite helper returned unexpected output");
  }
  return JSON.parse(stdout.slice(payloadStart));
}
