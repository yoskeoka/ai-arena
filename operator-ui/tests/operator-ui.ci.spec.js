import path from "node:path";
import { expect, test } from "@playwright/test";

const backendBaseURL = `http://127.0.0.1:${process.env.OPERATOR_UI_BACKEND_PORT ?? "10000"}`;
const presetId = process.env.OPERATOR_UI_PRESET_ID ?? "echo-reference";
const delegatedDownloadExpectation = process.env.OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD ?? "0";
const captureArtifacts = process.env.OPERATOR_UI_CAPTURE_ARTIFACTS === "1";
const artifactDir = process.env.OPERATOR_UI_ARTIFACT_DIR ?? "./test-results";

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

  await expect(page.getByRole("heading", { name: "AI Arena Minimal Operator UI" })).toBeVisible();
  await expect(page.getByTestId("operator-panel-preset-queue")).toBeVisible();
  await expect(page.getByTestId("operator-panel-active-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-detail")).toBeVisible();

  const activePanel = page.getByTestId("operator-panel-active-matches");
  const completedPanel = page.getByTestId("operator-panel-completed-matches");
  const knownSubmissionIDs = await currentSubmissionIDs(request);

  await page.getByTestId(`preset-queue-action-${presetId}`).click();

  const created = await waitForRecord(request, async () => {
    const activeItems = await listItems(request, `${backendBaseURL}/api/v1/matches/active`);
    const activeRecord = activeItems.find((item) => !knownSubmissionIDs.has(item.submission_id));
    if (activeRecord) {
      return { record: activeRecord, source: "active" };
    }
    const completedItems = await listItems(request, `${backendBaseURL}/api/v1/matches/completed`);
    const completedRecord = completedItems.find((item) => !knownSubmissionIDs.has(item.submission_id));
    if (completedRecord) {
      return { record: completedRecord, source: "completed" };
    }
    return null;
  }, "new submission after preset enqueue");

  if (created.source === "active") {
    await expect(activePanel.getByTestId(`match-row-${created.record.submission_id}`)).toBeVisible();
  }

  const completedRecord = await waitForRecord(request, async () => {
    const completedItems = await listItems(request, `${backendBaseURL}/api/v1/matches/completed`);
    return completedItems.find((item) => item.submission_id === created.record.submission_id);
  }, "completed submission in completed list");

  await page.reload();

  const completedRow = completedPanel.getByTestId(`match-row-${completedRecord.submission_id}`);
  await expect(completedRow).toBeVisible();
  await completedRow.click();

  const detail = page.getByTestId(`match-detail-${completedRecord.submission_id}`);
  await expect(detail).toBeVisible();
  await expect(detail.getByRole("heading", { name: completedRecord.match_id, exact: true })).toBeVisible();
  await expect(detail.getByText(completedRecord.submission_id, { exact: true })).toBeVisible();
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

  if (captureArtifacts) {
    await page.screenshot({
      fullPage: true,
      path: path.join(artifactDir, "completed-detail.png"),
    });
    await context.tracing.stop({ path: path.join(artifactDir, "operator-ui-flow.zip") });
  }
});

async function currentSubmissionIDs(request) {
  const activeItems = await listItems(request, `${backendBaseURL}/api/v1/matches/active`);
  const completedItems = await listItems(request, `${backendBaseURL}/api/v1/matches/completed`);
  return new Set([...activeItems, ...completedItems].map((item) => item.submission_id));
}

async function listItems(request, url) {
  const response = await request.get(url);
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  return payload.items ?? [];
}

async function waitForRecord(request, probe, description) {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    const record = await probe(request);
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
