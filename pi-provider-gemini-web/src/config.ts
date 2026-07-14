import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

export const CONFIG_FILE_NAME = "pi-provider-gemini-web.json";

export interface GeminiWebConfig {
  baseUrl: string;
  source: "global" | "project";
}

export interface LoadGeminiWebConfigOptions {
  agentDir: string;
  cwd: string;
  projectTrusted: boolean;
  configDirName?: string;
}

export class ConfigError extends Error {
  public readonly configPath: string;

  constructor(message: string, configPath: string) {
    super(`${configPath}: ${message}`);
    this.name = "ConfigError";
    this.configPath = configPath;
  }
}

function parseConfig(path: string, source: GeminiWebConfig["source"]): GeminiWebConfig {
  let parsed: unknown;
  try {
    parsed = JSON.parse(readFileSync(path, "utf8"));
  } catch (error) {
    throw new ConfigError(
      `invalid JSON: ${error instanceof Error ? error.message : String(error)}`,
      path,
    );
  }

  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    throw new ConfigError("expected a JSON object", path);
  }

  const record = parsed as Record<string, unknown>;
  const unknownKeys = Object.keys(record).filter((key) => key !== "baseUrl");
  if (unknownKeys.length > 0) {
    throw new ConfigError(`unknown field ${JSON.stringify(unknownKeys[0])}`, path);
  }
  if (typeof record.baseUrl !== "string" || record.baseUrl.trim() === "") {
    throw new ConfigError("baseUrl must be a non-empty string", path);
  }

  let url: URL;
  try {
    url = new URL(record.baseUrl);
  } catch {
    throw new ConfigError("baseUrl must be an absolute URL", path);
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new ConfigError("baseUrl must use http or https", path);
  }
  if (url.username || url.password) {
    throw new ConfigError("baseUrl must not contain credentials", path);
  }

  return { baseUrl: url.toString().replace(/\/$/, ""), source };
}

export function getGlobalConfigPath(agentDir: string): string {
  return join(agentDir, "extensions", CONFIG_FILE_NAME);
}

export function getProjectConfigPath(cwd: string, configDirName = ".pi"): string {
  return join(cwd, configDirName, CONFIG_FILE_NAME);
}

export function loadGeminiWebConfig(
  options: LoadGeminiWebConfigOptions,
): GeminiWebConfig | undefined {
  const globalPath = getGlobalConfigPath(options.agentDir);
  let config = existsSync(globalPath) ? parseConfig(globalPath, "global") : undefined;

  if (options.projectTrusted) {
    const projectPath = getProjectConfigPath(options.cwd, options.configDirName ?? ".pi");
    if (existsSync(projectPath)) config = parseConfig(projectPath, "project");
  }

  return config;
}
