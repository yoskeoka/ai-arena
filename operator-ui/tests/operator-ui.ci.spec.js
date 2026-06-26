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
const artifactRef = process.env.OPERATOR_UI_TEST_ARTIFACT_REF ?? path.resolve(testDir, "../../testdata/ai/echo/echo-ai-2turn");

test.setTimeout(120_000);

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
  if (captureArtifacts) {
    await context.tracing.start({ screenshots: true, snapshots: true, sources: true });
  }

  const api = authEnabled ? createBrowserAPI(page) : createRequestAPI(request);
  const health = await request.get(`${backendBaseURL}/healthz`);
  expect(health.ok()).toBeTruthy();

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
  await expect(page.getByTestId("operator-nav-overview")).toBeVisible();
  await expect(page.getByTestId("operator-nav-games")).toBeVisible();
  await expect(page.getByTestId("operator-nav-submissions")).toBeVisible();
  await expect(page.getByTestId("operator-nav-requests")).toBeVisible();
  await expect(page.getByTestId("operator-nav-rankings")).toBeVisible();

  const suffix = Date.now().toString();
  const registrationID = `echo-count-ui-${suffix}`;
  const aiSubmissionID1 = `ai-ui-${suffix}-01`;
  const aiSubmissionID2 = `ai-ui-${suffix}-02`;
  const requestOutputDir = path.join(os.tmpdir(), `operator-ui-request-${suffix}`);

  await page.getByTestId("operator-nav-games").click();
  await expect(page.getByTestId("operator-form-games")).toBeVisible();
  await page.getByLabel("Registration ID").fill(registrationID);
  await page.getByLabel("Game ID").fill("echo-count");
  await page.getByLabel("Game Version").fill("2.0.0");
  await page.getByLabel("Ruleset Version").fill("phase2-simultaneous-2turn");
  await page.getByRole("button", { name: "Create game registration" }).click();
  await expect(page.getByTestId(`game-row-${registrationID}`)).toBeVisible();

  await page.getByTestId("operator-nav-submissions").click();
  await expect(page.getByTestId("operator-form-submissions")).toBeVisible();
  await createAISubmission(page, {
    submissionID: aiSubmissionID1,
    registrationID,
    artifactRef,
    displayName: "Echo UI Alpha",
  });
  await expect(page.getByTestId(`submission-row-${aiSubmissionID1}`)).toBeVisible();
  await createAISubmission(page, {
    submissionID: aiSubmissionID2,
    registrationID,
    artifactRef,
    displayName: "Echo UI Beta",
  });
  await expect(page.getByTestId(`submission-row-${aiSubmissionID2}`)).toBeVisible();

  await page.getByTestId("operator-nav-requests").click();
  await expect(page.getByTestId("operator-form-requests")).toBeVisible();
  await page.getByLabel("Game Registration ID").fill(registrationID);
  await page.getByLabel("Output Dir").fill(requestOutputDir);
  await page.getByLabel("Player 1 ID").fill("alpha");
  await page.getByLabel("Player 1 AI Submission ID").fill(aiSubmissionID1);
  await page.getByLabel("Player 2 ID").fill("beta");
  await page.getByLabel("Player 2 AI Submission ID").fill(aiSubmissionID2);
  await page.getByRole("button", { name: "Create match request" }).click();

  const createdRequest = await waitForRequest(api, registrationID, requestOutputDir);
  await expect(page.getByTestId(`request-row-${createdRequest.request_id}`)).toBeVisible();

  const initialRun = await waitForRunState(api, createdRequest.latest_run_id, "completed");

  await page.getByRole("link", { name: "Open latest run detail" }).click();
  await expect(page).toHaveURL(new RegExp(`/operator/runs/${initialRun.run_id}$`));
  await expect(page.getByTestId(`match-detail-${initialRun.run_id}`)).toBeVisible();
  await expect(page.getByTestId("run-action-rerun")).toBeVisible();
  await page.getByTestId("run-action-rerun").click();

  const rerunRequest = await waitForLatestRunChange(api, createdRequest.request_id, initialRun.run_id);
  const rerunRun = await waitForRunState(api, rerunRequest.latest_run_id, "completed");

  await page.goto(`/operator/runs/${rerunRun.run_id}`);
  await expect(page.getByTestId(`match-detail-${rerunRun.run_id}`)).toBeVisible();
  await expect(page.getByTestId("run-action-promote")).toBeVisible();
  await page.getByTestId("run-action-promote").click();
  await expect.poll(async () => getRunDetail(api, rerunRun.run_id)).toMatchObject({ run_id: rerunRun.run_id, official: true });

  await page.getByTestId("operator-nav-rankings").click();
  const quickScopeId = scopeTestId(rerunRun.game_id, rerunRun.game_version, rerunRun.ruleset_version);
  await expect(page.getByTestId(`ranking-scope-${quickScopeId}`)).toBeVisible();
  await page.getByTestId(`ranking-scope-${quickScopeId}`).click();
  await expect(page.getByText(`last applied run: ${rerunRun.run_id}`)).toBeVisible();
  await expect(page.getByTestId(`ranking-entry-${encodeURIComponent(artifactRef)}`)).toBeVisible();

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

async function createAISubmission(page, { submissionID, registrationID, artifactRef, displayName }) {
  await page.getByLabel("AI Submission ID").fill(submissionID);
  await page.getByLabel("Game Registration ID").fill(registrationID);
  await page.getByLabel("Artifact Ref").fill(artifactRef);
  await page.getByLabel("Display Name").fill(displayName);
  await page.getByRole("button", { name: "Create AI submission" }).click();
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
