import os from "node:os";
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
const localOIDCEnabled = process.env.OPERATOR_UI_TEST_LOCAL_OIDC === "1";
const authMockUserID = process.env.OPERATOR_UI_AUTH_MOCK_USER_ID ?? "operator-user01";
const authMockLogin = process.env.OPERATOR_UI_AUTH_MOCK_LOGIN ?? authMockUserID;
const authSignupUserID = process.env.OPERATOR_UI_AUTH_SIGNUP_USER_ID ?? "operator-signup-user01";
const authSignupLogin = process.env.OPERATOR_UI_AUTH_SIGNUP_LOGIN ?? authSignupUserID;
const frontendHost = process.env.OPERATOR_UI_FRONTEND_HOST ?? "127.0.0.1";
const frontendPort = process.env.OPERATOR_UI_FRONTEND_PORT ?? "4173";
const testDir = path.dirname(fileURLToPath(import.meta.url));
const artifactRef = process.env.OPERATOR_UI_TEST_ARTIFACT_REF ?? path.resolve(testDir, "../../testdata/ai/echo/echo-ai");
const gameBundlePath =
  process.env.OPERATOR_UI_GAME_BUNDLE ??
  (process.env.OPERATOR_UI_TEST_SCENARIO === "remote"
    ? undefined
    : path.resolve(testDir, "../../.local/operator-ui-game-bundles/echo-count.arena-bundle.zip"));
const aiBundlePath =
  process.env.OPERATOR_UI_AI_BUNDLE ??
  (process.env.OPERATOR_UI_TEST_SCENARIO === "remote"
    ? undefined
    : path.resolve(testDir, "../../.local/operator-ui-game-bundles/echo-ai.arena-bundle.zip"));
const aiRevisionBundlePath =
  process.env.OPERATOR_UI_AI_REVISION_BUNDLE ??
  (process.env.OPERATOR_UI_TEST_SCENARIO === "remote"
    ? undefined
    : path.resolve(testDir, "../../.local/operator-ui-game-bundles/echo-ai-revision.arena-bundle.zip"));

test.setTimeout(120_000);

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
  test.skip(!authEnabled || localOIDCEnabled, "GitHub auth-only scenario");

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

test("local OIDC lane signs in a second fixed tester account", async ({ page }) => {
  test.skip(!localOIDCEnabled, "local OIDC-only scenario");
  await page.goto("/");
  await page.getByRole("link", { name: "Continue with local OIDC" }).click();
  await expect(page.getByRole("heading", { name: "Local OIDC sign in" })).toBeVisible();
  await page.getByLabel("Username").fill("tester02");
  await page.getByLabel("Password").fill("local-oidc-password");
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/($|operator$)/);
  await expect(page.getByText("Signed in as @tester02")).toBeVisible();
});

test("service-backed operator UI browser lane covers registration, request execution, ranking correction, and artifact access", async ({
  context,
  page,
  request,
}) => {
  test.skip(process.env.OPERATOR_UI_TEST_SCENARIO === "remote", "remote lane is limited to read-only smoke");
  if (!gameBundlePath) {
    throw new Error("OPERATOR_UI_GAME_BUNDLE is required for game bundle upload verification");
  }
  if (!aiBundlePath || !aiRevisionBundlePath) {
    throw new Error("OPERATOR_UI_AI_BUNDLE and OPERATOR_UI_AI_REVISION_BUNDLE are required for AI bundle upload verification");
  }
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
    if (localOIDCEnabled) {
      await page.getByRole("link", { name: "Continue with local OIDC" }).click();
      await expect(page.getByRole("heading", { name: "Local OIDC sign in" })).toBeVisible();
      await page.getByLabel("Username").fill("tester01");
      await page.getByLabel("Password").fill("local-oidc-password");
    } else {
      await page.getByRole("link", { name: "Continue with GitHub" }).click();
      await expect(page.getByRole("heading", { name: "GitHub OAuth Test Double" })).toBeVisible();
      await page.getByLabel("User ID").fill(authMockUserID);
    }
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
        principal: { provider_login: localOIDCEnabled ? "tester01" : authMockLogin },
      });
    await expect(page).toHaveURL(/\/($|operator$)/);
    await expect(page.getByText(`Signed in as @${localOIDCEnabled ? "tester01" : authMockLogin}`)).toBeVisible();
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

  const suffix = Date.now().toString();
  const registrationID = "echo-count-v2-phase2-simultaneous-3turn";
  const aiSubmissionID1 = `ai-ui-${suffix}-01`;
  const aiSubmissionID2 = `ai-ui-${suffix}-02`;
  const requestOutputDir = path.join(os.tmpdir(), `operator-ui-request-${suffix}`);

  await page.getByTestId("operator-nav-games").click();
  await expect(page.getByTestId("operator-form-games")).toBeVisible();
  await page.getByLabel("Game bundle ZIP").setInputFiles(path.resolve(testDir, "../package.json"));
  await page.getByRole("button", { name: "Upload game bundle" }).click();
  await expect(page.getByTestId("game-bundle-admission")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Activate game" })).toHaveCount(0);
  await expect(page.getByTestId("operator-form-games")).toContainText(/invalid|zip|bundle/i);

  await page.getByLabel("Game bundle ZIP").setInputFiles(gameBundlePath);
  await page.getByRole("button", { name: "Upload game bundle" }).click();
  await expect(page.getByTestId("game-bundle-admission")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByTestId("admitted-game-id")).toHaveText("echo-count");
  await expect(page.getByTestId("admitted-game-version")).toHaveText("2.0.0");
  await expect(page.getByTestId("admitted-artifact-id")).toHaveText(/[0-9a-f]{64}/);
  await page.getByLabel("Ruleset Version").selectOption("phase2-simultaneous-3turn");
  await page.getByRole("button", { name: "Activate game" }).click();
  await expect(page.getByTestId(`game-row-${registrationID}`)).toBeVisible();
  await expect(page.getByTestId(`game-row-${registrationID}`)).toContainText(/[0-9a-f]{64}/);

  await page.getByTestId("operator-nav-submissions").click();
  await expect(page.getByTestId("operator-form-submissions")).toBeVisible();
  await expect(page.getByLabel("Uploaded AI artifact ID")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Create AI submission" })).toHaveCount(0);
  if (authEnabled) {
    await page.getByLabel("Competition scope").fill(registrationID);
    await page.getByLabel("Bot name").fill("Echo UI Alpha");
    await page.getByLabel("AI bundle ZIP").setInputFiles(path.resolve(testDir, "../package.json"));
    await page.getByRole("button", { name: "Upload AI bundle" }).click();
    await expect(page.getByTestId("ai-bundle-admission")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Save bot revision" })).toHaveCount(0);
    await expect(page.getByTestId("operator-form-submissions")).toContainText(/invalid|zip|bundle/i);
    await page.getByLabel("AI bundle ZIP").setInputFiles(aiBundlePath);
    await page.getByRole("button", { name: "Upload AI bundle" }).click();
    await expect(page.getByTestId("ai-bundle-admission")).toBeVisible({ timeout: 30_000 });
    const firstArtifactID = await page.getByTestId("admitted-ai-artifact-id").textContent();
    expect(firstArtifactID).toMatch(/[0-9a-f]{64}/);
    await page.getByRole("button", { name: "Save bot revision" }).click();
    const botRow = page.getByTestId("operator-panel-submissions").locator('[data-testid^="bot-row-"]').first();
    await expect(botRow).toBeVisible();
    const botID = (await botRow.getAttribute("data-testid")).replace("bot-row-", "");
    const firstBotText = await botRow.textContent();
    await page.getByLabel("Existing bot ID").fill(botID);
    await page.getByLabel("AI bundle ZIP").setInputFiles(aiRevisionBundlePath);
    await page.getByRole("button", { name: "Upload AI bundle" }).click();
    await expect(page.getByTestId("ai-bundle-admission")).toBeVisible({ timeout: 30_000 });
    await expect(page.getByTestId("admitted-ai-artifact-id")).not.toHaveText(firstArtifactID);
    await page.getByRole("button", { name: "Save bot revision" }).click();
    await expect(page.getByTestId(`bot-row-${botID}`)).toBeVisible();
    await expect(botRow).not.toHaveText(firstBotText);
    await page.getByLabel("AI bundle ZIP").setInputFiles(path.resolve(testDir, "../package.json"));
    await page.getByRole("button", { name: "Upload AI bundle" }).click();
    await expect(page.getByTestId("ai-bundle-admission")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Save bot revision" })).toHaveCount(0);
    await expect(page.getByTestId(`bot-row-${botID}`)).toBeVisible();
  }

  await createLegacyAISubmission(api, {
    submissionID: aiSubmissionID1,
    registrationID,
    artifactRef,
    displayName: "Echo UI Alpha",
  });
  await createLegacyAISubmission(api, {
    submissionID: aiSubmissionID2,
    registrationID,
    artifactRef,
    displayName: "Echo UI Beta",
  });

  await page.getByTestId("operator-nav-requests").click();
  await expect(page.getByTestId("operator-form-requests")).toBeVisible();
  await expect(page.getByLabel("Competition scope")).toBeVisible();
  await expect(page.getByLabel("Game Registration ID")).toHaveCount(0);
  await expect(page.getByLabel("Output Dir")).toHaveCount(0);
  await createLegacyMatchRequest(api, registrationID, requestOutputDir, aiSubmissionID1, aiSubmissionID2);

  const createdRequest = await waitForRequest(api, registrationID, requestOutputDir);
  await page.reload();
  await expect(page.getByTestId(`request-row-${createdRequest.request_id}`)).toBeVisible();

  const initialRun = await waitForRunState(api, createdRequest.latest_run_id, "completed");

  await page.getByRole("link", { name: "Open latest run detail" }).click();
  await expect(page).toHaveURL(new RegExp(`/operator/runs/${initialRun.run_id}$`));
  await expect(page.getByTestId(`match-detail-${initialRun.run_id}`)).toBeVisible();
  const compactSummary = page.getByTestId(`match-detail-${initialRun.run_id}`).locator(".bg-ink");
  for (const label of ["Attempt", "Game", "Ruleset", "Output Dir", "Result Summary"]) {
    const metadata = compactSummary.getByText(label, { exact: true }).locator("..");
    await expect(metadata.locator("dt")).toHaveClass(/text-paper\/70/);
    await expect(metadata.locator("dd")).toHaveClass(/(?:^|\s)text-paper(?:\s|$)/);
  }
  await expect(page.getByTestId("run-action-rerun")).toBeVisible();
  await page.getByTestId("run-action-rerun").click();

  const rerunRequest = await waitForLatestRunChange(api, createdRequest.request_id, initialRun.run_id);
  const rerunRun = await waitForRunState(api, rerunRequest.latest_run_id, "completed");

  await page.goto(`/operator/runs/${rerunRun.run_id}`);
  await expect(page.getByTestId(`match-detail-${rerunRun.run_id}`)).toBeVisible();
  await expect(page.getByTestId("run-action-promote")).toBeVisible();
  await page.getByTestId("run-action-promote").click();
  await expect.poll(async () => getRunDetail(api, rerunRun.run_id)).toMatchObject({ run_id: rerunRun.run_id, official: true });

  await page.goto(`/operator/runs/${rerunRun.run_id}`);
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

async function createLegacyAISubmission(api, { submissionID, registrationID, artifactRef, displayName }) {
  const response = await api.postJSON(`${backendBaseURL}/api/v1/ai-submissions`, {
    ai_submission_id: submissionID,
    game_registration_id: registrationID,
    artifact_ref: artifactRef,
    display_name: displayName,
  });
  expect(response.ok).toBeTruthy();
}

async function createLegacyMatchRequest(api, registrationID, outputDir, firstSubmissionID, secondSubmissionID) {
  const response = await api.postJSON(`${backendBaseURL}/api/v1/match-requests`, {
    game_registration_id: registrationID,
    output_dir: outputDir,
    participants: [
      { player_id: "alpha", ai_submission_id: firstSubmissionID },
      { player_id: "beta", ai_submission_id: secondSubmissionID },
    ],
  });
  expect(response.ok).toBeTruthy();
}

async function waitForRequest(api, registrationID, outputDir) {
  return waitForRecord(api, async () => {
    const items = await listItems(api, `${backendBaseURL}/api/v1/match-requests`);
    return items.find((item) => item.game_registration_id === registrationID && item.output_dir === outputDir);
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

function scopeTestId(gameID, gameVersion, rulesetVersion) {
  return `${gameID}-${gameVersion}-${rulesetVersion}`.replace(/[^a-zA-Z0-9_-]+/g, "_");
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
