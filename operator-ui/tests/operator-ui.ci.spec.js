import path from "node:path";
import { expect, test } from "@playwright/test";

if (process.env.OPERATOR_UI_TEST_SCENARIO === "remote" && !process.env.OPERATOR_UI_BACKEND_BASE_URL) {
  throw new Error("OPERATOR_UI_BACKEND_BASE_URL is required when OPERATOR_UI_TEST_SCENARIO=remote");
}

const backendBaseURL =
  process.env.OPERATOR_UI_BACKEND_BASE_URL ?? `http://127.0.0.1:${process.env.OPERATOR_UI_BACKEND_PORT ?? "10000"}`;
const presetId = process.env.OPERATOR_UI_PRESET_ID ?? "echo-reference";
const delegatedDownloadExpectation = process.env.OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD ?? "0";
const captureArtifacts = process.env.OPERATOR_UI_CAPTURE_ARTIFACTS === "1";
const artifactDir = process.env.OPERATOR_UI_ARTIFACT_DIR ?? "./test-results";
const authEnabled = process.env.OPERATOR_UI_TEST_AUTH === "1";
const authMockUserID = process.env.OPERATOR_UI_AUTH_MOCK_USER_ID ?? "operator-user01";
const authMockLogin = process.env.OPERATOR_UI_AUTH_MOCK_LOGIN ?? authMockUserID;

test.setTimeout(90_000);

test("service-backed operator UI browser lane covers queue, active, completed detail, and artifact access", async ({
  context,
  page,
  request,
}) => {
  if (captureArtifacts) {
    await context.tracing.start({ screenshots: true, snapshots: true, sources: true });
  }

  const health = await request.get(`${backendBaseURL}/healthz`);
  expect(health.ok()).toBeTruthy();

  await page.goto("/");

  if (authEnabled) {
    await expect(page.getByRole("heading", { name: "Sign in with GitHub" })).toBeVisible();
    await page.getByRole("link", { name: "Continue with GitHub" }).click();
    await expect(page.getByRole("heading", { name: "GitHub OAuth Test Double" })).toBeVisible();
    await page.getByLabel("User ID").fill(authMockUserID);
    await page.getByRole("button", { name: "Login" }).click();
    await expect(page.getByText(`Signed in as @${authMockLogin}`, { exact: true })).toBeVisible();
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
  }

  await expect(page.getByRole("heading", { name: "AI Arena Minimal Operator UI" })).toBeVisible();
  await expect(page.getByTestId("operator-panel-preset-queue")).toBeVisible();
  await expect(page.getByTestId("operator-panel-active-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-detail")).toBeVisible();

  const api = authEnabled ? page.context().request : request;
  const activePanel = page.getByTestId("operator-panel-active-matches");
  const completedPanel = page.getByTestId("operator-panel-completed-matches");
  if (authEnabled) {
    const activeRows = activePanel.locator('[data-testid^="match-row-"]');
    const completedRows = completedPanel.locator('[data-testid^="match-row-"]');
    const initialActiveRows = await activeRows.count();
    const initialCompletedRows = await completedRows.count();

    await page.getByTestId(`preset-queue-action-${presetId}`).click();

    await expect
      .poll(async () => (await activeRows.count()) + (await completedRows.count()), { timeout: 30_000 })
      .toBeGreaterThan(initialActiveRows + initialCompletedRows);

    await expect.poll(async () => completedRows.count(), { timeout: 30_000 }).toBeGreaterThan(initialCompletedRows);
    await page.reload();

    const completedRow = completedPanel.locator('[data-testid^="match-row-"]').first();
    await expect(completedRow).toBeVisible();
    await completedRow.click();

    const detail = page.locator('[data-testid^="match-detail-"]').first();
    await expect(detail).toBeVisible();
    await expect(detail.getByText("Status", { exact: true })).toBeVisible();
    await expect(detail.getByText("completed", { exact: true })).toBeVisible();

    const resultSummaryArtifact = detail.getByTestId("artifact-entry-result-summary");
    await expect(resultSummaryArtifact).toBeVisible();
    await expect(resultSummaryArtifact.getByRole("link", { name: "open delegated download" })).toHaveCount(0);
  } else {
    const knownRunIDs = await currentRunIDs(api);

    await page.getByTestId(`preset-queue-action-${presetId}`).click();

    const created = await waitForRecord(api, async () => {
      const activeItems = await listItems(api, `${backendBaseURL}/api/v1/matches/active`);
      const activeRecord = activeItems.find((item) => !knownRunIDs.has(item.run_id));
      if (activeRecord) {
        return { record: activeRecord, source: "active" };
      }
      const completedItems = await listItems(api, `${backendBaseURL}/api/v1/matches/completed`);
      const completedRecord = completedItems.find((item) => !knownRunIDs.has(item.run_id));
      if (completedRecord) {
        return { record: completedRecord, source: "completed" };
      }
      return null;
    }, "new submission after preset enqueue");

    if (created.source === "active") {
      await expect(activePanel.getByTestId(`match-row-${created.record.run_id}`)).toBeVisible();
    }

    const completedRecord = await waitForRecord(api, async () => {
      const completedItems = await listItems(api, `${backendBaseURL}/api/v1/matches/completed`);
      return completedItems.find((item) => item.run_id === created.record.run_id);
    }, "completed submission in completed list");

    await page.reload();

    const completedRow = completedPanel.getByTestId(`match-row-${completedRecord.run_id}`);
    await expect(completedRow).toBeVisible();
    await completedRow.click();

    const detail = page.getByTestId(`match-detail-${completedRecord.run_id}`);
    await expect(detail).toBeVisible();
    await expect(detail.getByRole("heading", { name: completedRecord.match_id, exact: true })).toBeVisible();
    await expect(detail.getByText(completedRecord.run_id, { exact: true })).toBeVisible();
    await expect(detail.getByText("Status", { exact: true })).toBeVisible();
    await expect(detail.getByText("completed", { exact: true })).toBeVisible();

    const resultSummaryArtifact = detail.getByTestId("artifact-entry-result-summary");
    await expect(resultSummaryArtifact).toBeVisible();
    await expect(resultSummaryArtifact.getByText(completedRecord.result_summary_path ?? "", { exact: true })).toBeVisible();

    const downloadLink = resultSummaryArtifact.getByRole("link", { name: "open delegated download" });
    const expectsDelegatedDownload =
      delegatedDownloadExpectation === "auto"
        ? (completedRecord.result_summary_path ?? "").startsWith("s3://")
        : delegatedDownloadExpectation === "1";
    if (expectsDelegatedDownload) {
      await expect(downloadLink).toBeVisible();
      await expect(downloadLink).toHaveAttribute("href", /http:\/\//);
    } else {
      await expect(downloadLink).toHaveCount(0);
    }
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

async function currentRunIDs(api) {
  const activeItems = await listItems(api, `${backendBaseURL}/api/v1/matches/active`);
  const completedItems = await listItems(api, `${backendBaseURL}/api/v1/matches/completed`);
  return new Set([...activeItems, ...completedItems].map((item) => item.run_id));
}

async function listItems(api, url) {
  const response = await api.get(url);
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  return payload.items ?? [];
}

async function waitForRecord(api, probe, description) {
  const deadline = Date.now() + 30_000;
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
