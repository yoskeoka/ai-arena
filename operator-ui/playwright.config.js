import { defineConfig, devices } from "@playwright/test";

const testScenario = process.env.OPERATOR_UI_TEST_SCENARIO ?? "local";
const artifactDir = process.env.OPERATOR_UI_ARTIFACT_DIR ?? "./test-results";
const reportDir = process.env.OPERATOR_UI_REPORT_DIR ?? "./playwright-report";
const backendPort = process.env.OPERATOR_UI_BACKEND_PORT ?? "10000";
const frontendPort = process.env.OPERATOR_UI_FRONTEND_PORT ?? "4173";
const browserChannel = testScenario === "ci" ? process.env.OPERATOR_UI_BROWSER_CHANNEL ?? "chrome" : undefined;

export default defineConfig({
  testDir: "./tests",
  outputDir: artifactDir,
  reporter: [["list"], ["html", { open: "never", outputFolder: reportDir }]],
  testMatch: testScenario === "ci" ? /.*\.ci\.spec\.js/ : /^(?!.*\.ci\.spec\.js$).*\.spec\.js$/,
  use: {
    baseURL: `http://127.0.0.1:${frontendPort}`,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    video: testScenario === "ci" ? "off" : "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], ...(browserChannel ? { channel: browserChannel } : {}) },
    },
  ],
  webServer: [
    {
      command:
        testScenario === "ci"
          ? "./tools/dev/operator-ui-backend.sh"
          : `GOPATH=/tmp/ai-arena-operator-ui-go GOMODCACHE=/tmp/ai-arena-operator-ui-go/pkg/mod GOCACHE=/tmp/ai-arena-operator-ui-go-build go run ./cmd/operator-ui-fixture --listen-addr 127.0.0.1:${backendPort}`,
      cwd: "..",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      url: `http://127.0.0.1:${backendPort}/healthz`,
    },
    {
      command:
        testScenario === "ci"
          ? "./tools/dev/operator-ui-frontend.sh"
          : `pnpm_config_store_dir=/tmp/pnpm-store-ai-arena pnpm exec vite --host 127.0.0.1 --port ${frontendPort} --strictPort`,
      cwd: testScenario === "ci" ? ".." : ".",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      url: `http://127.0.0.1:${frontendPort}`,
    },
  ],
});
