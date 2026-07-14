import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import { ConfigError, loadGeminiWebConfig } from "../src/config.ts";

function fixture() {
  const root = mkdtempSync(join(tmpdir(), "gemini-web-config-"));
  const agentDir = join(root, "agent");
  const cwd = join(root, "project");
  mkdirSync(join(agentDir, "extensions"), { recursive: true });
  mkdirSync(join(cwd, ".pi"), { recursive: true });
  return { agentDir, cwd };
}

function writeJson(path: string, value: unknown) {
  writeFileSync(path, `${JSON.stringify(value)}\n`, "utf8");
}

test("loads the global extension configuration", () => {
  const { agentDir, cwd } = fixture();
  writeJson(join(agentDir, "extensions", "pi-provider-gemini-web.json"), {
    baseUrl: "http://gemini.internal:8080/",
  });

  assert.deepEqual(loadGeminiWebConfig({ agentDir, cwd, projectTrusted: false }), {
    baseUrl: "http://gemini.internal:8080",
    source: "global",
  });
});

test("trusted project configuration overrides global configuration", () => {
  const { agentDir, cwd } = fixture();
  writeJson(join(agentDir, "extensions", "pi-provider-gemini-web.json"), {
    baseUrl: "http://global.internal:8080",
  });
  writeJson(join(cwd, ".pi", "pi-provider-gemini-web.json"), {
    baseUrl: "https://project.internal/gemini/",
  });

  assert.deepEqual(loadGeminiWebConfig({ agentDir, cwd, projectTrusted: true }), {
    baseUrl: "https://project.internal/gemini",
    source: "project",
  });
});

test("untrusted project configuration is never read", () => {
  const { agentDir, cwd } = fixture();
  writeJson(join(agentDir, "extensions", "pi-provider-gemini-web.json"), {
    baseUrl: "http://global.internal:8080",
  });
  writeFileSync(join(cwd, ".pi", "pi-provider-gemini-web.json"), "not json", "utf8");

  assert.equal(
    loadGeminiWebConfig({ agentDir, cwd, projectTrusted: false })?.baseUrl,
    "http://global.internal:8080",
  );
});

test("supports a trusted project-only configuration", () => {
  const { agentDir, cwd } = fixture();
  writeJson(join(cwd, ".pi", "pi-provider-gemini-web.json"), {
    baseUrl: "http://project.internal:8080",
  });

  assert.equal(loadGeminiWebConfig({ agentDir, cwd, projectTrusted: true })?.source, "project");
});

test("returns undefined when no configuration exists", () => {
  const { agentDir, cwd } = fixture();
  assert.equal(loadGeminiWebConfig({ agentDir, cwd, projectTrusted: true }), undefined);
});

test("rejects invalid URLs and unknown fields", () => {
  const { agentDir, cwd } = fixture();
  const path = join(agentDir, "extensions", "pi-provider-gemini-web.json");
  writeJson(path, { baseUrl: "file:///tmp/socket", apiKey: "not-supported" });

  assert.throws(
    () => loadGeminiWebConfig({ agentDir, cwd, projectTrusted: false }),
    (error: unknown) => error instanceof ConfigError && error.configPath === path,
  );
});
