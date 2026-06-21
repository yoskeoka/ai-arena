import { defineConfig, devices } from "@playwright/test";

const testScenario = process.env.OPERATOR_UI_TEST_SCENARIO ?? "local";
const artifactDir = process.env.OPERATOR_UI_ARTIFACT_DIR ?? "./test-results";
const reportDir = process.env.OPERATOR_UI_REPORT_DIR ?? "./playwright-report";
const backendPort = process.env.OPERATOR_UI_BACKEND_PORT ?? "10000";
const frontendPort = process.env.OPERATOR_UI_FRONTEND_PORT ?? "4173";
const frontendHost = process.env.OPERATOR_UI_FRONTEND_HOST ?? "127.0.0.1";
const browserChannel = testScenario === "ci" ? process.env.OPERATOR_UI_BROWSER_CHANNEL ?? "chrome" : undefined;
const usesFixtureBackend = testScenario === "local";
const usesManagedBackend = !usesFixtureBackend;
const usesRemoteServers = testScenario === "remote";
const remoteBaseURL = process.env.OPERATOR_UI_BASE_URL;
const captureArtifacts = process.env.OPERATOR_UI_CAPTURE_ARTIFACTS === "1";

if (usesRemoteServers && !remoteBaseURL) {
  throw new Error("OPERATOR_UI_BASE_URL is required when OPERATOR_UI_TEST_SCENARIO=remote");
}

export default defineConfig({
  testDir: "./tests",
  outputDir: artifactDir,
  reporter: [["list"], ["html", { open: "never", outputFolder: reportDir }]],
  testMatch: usesFixtureBackend ? /^(?!.*\.ci\.spec\.js$).*\.spec\.js$/ : /.*\.ci\.spec\.js/,
  use: {
    baseURL: usesRemoteServers ? remoteBaseURL : `http://${frontendHost}:${frontendPort}`,
    screenshot: "only-on-failure",
    trace: captureArtifacts || testScenario === "real-local" ? "off" : "retain-on-failure",
    video: usesManagedBackend ? "off" : "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], ...(browserChannel ? { channel: browserChannel } : {}) },
    },
  ],
  webServer: usesRemoteServers
    ? undefined
    : [
        {
          command:
            usesManagedBackend
              ? "./tools/dev/operator-ui-backend.sh"
              : `GOPATH=/tmp/ai-arena-operator-ui-go GOMODCACHE=/tmp/ai-arena-operator-ui-go/pkg/mod GOCACHE=/tmp/ai-arena-operator-ui-go-build go run ./cmd/operator-ui-fixture --listen-addr 127.0.0.1:${backendPort}`,
          cwd: "..",
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
          url: `http://127.0.0.1:${backendPort}/healthz`,
        },
        {
          command:
            usesManagedBackend
              ? "./tools/dev/operator-ui-frontend.sh"
              : `pnpm_config_store_dir=/tmp/pnpm-store-ai-arena pnpm exec vite --host ${frontendHost} --port ${frontendPort} --strictPort`,
          cwd: usesManagedBackend ? ".." : ".",
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
          url: `http://${frontendHost}:${frontendPort}`,
        },
      ],
});
