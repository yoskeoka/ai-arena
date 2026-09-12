import { expect, test } from "@playwright/test";

test("operator route alias serves the same operator surface", async ({ page }) => {
  const presetRequests = [];
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === "/api/v1/preset-matches") {
      presetRequests.push(request);
    }
  });

  await page.goto("/operator");

  await expect(page.getByRole("heading", { name: "AI Arena Operator Console" })).toBeVisible();
  await expect(page.getByTestId("operator-nav-overview")).toBeVisible();
  await expect(page.getByTestId("operator-nav-invites")).toBeVisible();
  await expect(page.getByTestId("operator-nav-games")).toBeVisible();
  await expect(page.getByTestId("operator-nav-submissions")).toBeVisible();
  await expect(page.getByTestId("operator-nav-requests")).toBeVisible();
  await expect(page.getByTestId("operator-nav-rankings")).toBeVisible();
  await expect(page.getByTestId("operator-panel-active-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-detail")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Preset Queue", exact: true })).toHaveCount(0);
  await expect(page.getByText("Echo Reference", { exact: true })).toHaveCount(0);
  expect(presetRequests).toHaveLength(0);

  await page.getByTestId("operator-nav-invites").click();
  await expect(page.getByTestId("operator-form-invites")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Issue Signup Invite" })).toBeVisible();

  await page.getByTestId("operator-nav-submissions").click();
  await expect(page.getByTestId("operator-form-submissions")).toBeVisible();
  await expect(page.getByLabel("AI bundle ZIP")).toBeVisible();
  await expect(page.getByLabel("Uploaded AI artifact ID")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Create AI submission" })).toHaveCount(0);
});

test("local operator UI browser lane covers active, completed detail, artifact access, and no preset request", async ({ page, request }) => {
  const health = await request.get("http://127.0.0.1:10000/healthz");
  expect(health.status()).toBe(200);
  expect(await health.json()).toEqual({ api: "OK", worker: "OK" });

  const presetRequests = [];
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === "/api/v1/preset-matches") {
      presetRequests.push(request);
    }
  });

  await page.goto("/");

  await expect(page.getByRole("heading", { name: "AI Arena Operator Console" })).toBeVisible();
  await expect(page.getByTestId("operator-panel-active-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-detail")).toBeVisible();

  const activePanel = page.getByTestId("operator-panel-active-matches");
  const completedPanel = page.getByTestId("operator-panel-completed-matches");
  const completedRow = completedPanel.getByTestId("match-row-run-completed-local");
  expect(presetRequests).toHaveLength(0);

  await expect(activePanel.getByTestId("match-row-run-active-queued")).toBeVisible();
  await expect(completedRow).toBeVisible();
  await completedRow.click();

  const detail = page.getByTestId("match-detail-run-completed-local");
  await expect(detail).toBeVisible();
  await expect(detail.getByRole("heading", { name: "match-completed-local", exact: true })).toBeVisible();
  await expect(detail.getByText("run-completed-local", { exact: true })).toBeVisible();
  await expect(detail.getByText("Status", { exact: true })).toBeVisible();
  await expect(detail.getByText("completed", { exact: true })).toBeVisible();
  const resultSummaryArtifact = detail.getByTestId("artifact-entry-result-summary");
  await expect(resultSummaryArtifact).toBeVisible();
  await expect(resultSummaryArtifact.getByRole("link", { name: "open delegated download" })).toHaveAttribute(
    "href",
    "http://127.0.0.1:10000/fixture-artifacts/result-summary.json",
  );

  await page.goto("/operator/runs/run-completed-local");
  const runDetail = page.getByTestId("match-detail-run-completed-local");
  await expect(runDetail).toBeVisible();
  const compactSummary = runDetail.locator(".bg-ink");
  for (const label of ["Attempt", "Game", "Ruleset", "Output Dir", "Result Summary"]) {
    const metadata = compactSummary.getByText(label, { exact: true }).locator("..");
    await expect(metadata.locator("dt")).toHaveClass(/text-paper\/70/);
    await expect(metadata.locator("dd")).toHaveClass(/(?:^|\s)text-paper(?:\s|$)/);
  }
});
