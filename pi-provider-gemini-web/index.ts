import {
  CONFIG_DIR_NAME,
  getAgentDir,
  type ExtensionAPI,
  type ExtensionFactory,
} from "@earendil-works/pi-coding-agent";
import { existsSync } from "node:fs";

import { type GeminiWebConfig, getProjectConfigPath, loadGeminiWebConfig } from "./src/config.ts";
import {
  API_ID,
  discoverGeminiWebModels,
  FALLBACK_MODELS,
  type GeminiWebModelDefinition,
  PROVIDER_ID,
} from "./src/models.ts";
import { createGeminiWebStream } from "./src/stream.ts";
import { buildResearchTools } from "./src/tools/research.ts";

interface GeminiWebExtensionOptions {
  agentDir?: string;
  cwd?: string;
  fetch?: typeof fetch;
}

interface RuntimeState {
  baseUrl?: string;
  config?: GeminiWebConfig;
  freshSessionIds: Set<string>;
  modelsByBaseUrl: Map<string, GeminiWebModelDefinition[]>;
}

export function GeminiWeb(options: GeminiWebExtensionOptions = {}): ExtensionFactory {
  return async (pi: ExtensionAPI) => {
    const agentDir = options.agentDir ?? getAgentDir();
    const initialCwd = options.cwd ?? process.cwd();
    const fetchFn = options.fetch ?? fetch;
    const state: RuntimeState = {
      freshSessionIds: new Set(),
      modelsByBaseUrl: new Map(),
    };
    const streamSimple = createGeminiWebStream(state);

    const registerProvider = async (config: GeminiWebConfig): Promise<void> => {
      let models = state.modelsByBaseUrl.get(config.baseUrl);
      if (!models) {
        models = await discoverGeminiWebModels(config.baseUrl, fetchFn);
        state.modelsByBaseUrl.set(config.baseUrl, models);
      }
      state.baseUrl = config.baseUrl;
      state.config = config;
      pi.registerProvider(PROVIDER_ID, {
        name: "Gemini Web",
        baseUrl: `${config.baseUrl}/v1`,
        apiKey: "gemini-web-internal",
        api: API_ID,
        streamSimple,
        models,
      });
    };

    let globalConfig: GeminiWebConfig | undefined;
    try {
      globalConfig = loadGeminiWebConfig({
        agentDir,
        cwd: initialCwd,
        projectTrusted: false,
        configDirName: CONFIG_DIR_NAME,
      });
    } catch (error) {
      console.error(`[gemini-web] ${error instanceof Error ? error.message : String(error)}`);
    }
    if (globalConfig) {
      try {
        await registerProvider(globalConfig);
      } catch (error) {
        console.error(`[gemini-web] ${error instanceof Error ? error.message : String(error)}`);
      }
    } else if (existsSync(getProjectConfigPath(initialCwd, CONFIG_DIR_NAME))) {
      // Project config cannot be read before trust is resolved. Register stable
      // model names so --model can resolve during CLI startup; session_start
      // replaces this provisional registration with discovered models.
      pi.registerProvider(PROVIDER_ID, {
        name: "Gemini Web",
        baseUrl: "http://127.0.0.1:8080/v1",
        apiKey: "gemini-web-internal",
        api: API_ID,
        streamSimple,
        models: FALLBACK_MODELS,
      });
    }

    for (const tool of buildResearchTools({
      getBaseUrl: () => state.config?.baseUrl,
    })) {
      pi.registerTool(tool);
    }

    pi.on("session_start", async (event, ctx) => {
      if (event.reason === "fork") {
        state.freshSessionIds.add(ctx.sessionManager.getSessionId());
      }

      let config: GeminiWebConfig | undefined;
      try {
        config = loadGeminiWebConfig({
          agentDir,
          cwd: ctx.cwd,
          projectTrusted: ctx.isProjectTrusted(),
          configDirName: CONFIG_DIR_NAME,
        });
      } catch (error) {
        ctx.ui.notify(error instanceof Error ? error.message : String(error), "error");
        pi.unregisterProvider(PROVIDER_ID);
        state.baseUrl = undefined;
        state.config = undefined;
        return;
      }

      if (!config) {
        pi.unregisterProvider(PROVIDER_ID);
        state.baseUrl = undefined;
        state.config = undefined;
        ctx.ui.notify(
          `Gemini Web is not configured. Add ${CONFIG_DIR_NAME}/pi-provider-gemini-web.json to the project or configure the global extension file.`,
          "warning",
        );
        return;
      }

      try {
        await registerProvider(config);
      } catch (error) {
        pi.unregisterProvider(PROVIDER_ID);
        state.baseUrl = undefined;
        state.config = undefined;
        ctx.ui.notify(error instanceof Error ? error.message : String(error), "error");
      }
    });
  };
}

export default GeminiWeb();
