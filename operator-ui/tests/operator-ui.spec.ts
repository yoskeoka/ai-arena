import { expect, test } from "@playwright/test";

test("local operator UI browser lane covers queue, active, completed detail, and artifact access", async ({ page, request }) => {
  const health = await request.get("http://127.0.0.1:10000/healthz");
  expect(health.ok()).toBeTruthy();

  await page.goto("/");

  await expect(page.getByRole("heading", { name: "AI Arena Minimal Operator UI" })).toBeVisible();
  await expect(page.getByTestId("operator-panel-preset-queue")).toBeVisible();
  await expect(page.getByTestId("operator-panel-active-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-matches")).toBeVisible();
  await expect(page.getByTestId("operator-panel-completed-detail")).toBeVisible();

  const activePanel = page.getByTestId("operator-panel-active-matches");
  const completedPanel = page.getByTestId("operator-panel-completed-matches");
  const detail = page.getByTestId("match-detail-sub-completed-local");

  await expect(activePanel.getByTestId("match-row-sub-active-queued")).toBeVisible();
  await expect(completedPanel.getByTestId("match-row-sub-completed-local")).toBeVisible();
  await expect(detail).toBeVisible();
  await expect(detail.getByText("match-completed-local")).toBeVisible();
  await expect(detail.getByText("sub-completed-local")).toBeVisible();
  await expect(detail.getByText("Status")).toBeVisible();
  await expect(detail.getByText("completed")).toBeVisible();
  await expect(detail.getByTestId("artifact-entry-result-summary")).toBeVisible();
  await expect(detail.getByRole("link", { name: "open delegated download" })).toHaveAttribute(
    "href",
    "http://127.0.0.1:10000/fixture-artifacts/result-summary.json",
  );

  const initialActiveRows = await activePanel.locator('[data-testid^="match-row-"]').count();
  await page.getByTestId("preset-queue-action-echo-reference").click();

  await expect
    .poll(async () => activePanel.locator('[data-testid^="match-row-"]').count())
    .toBeGreaterThan(initialActiveRows);
});
