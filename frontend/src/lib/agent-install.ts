function shellQuote(value: string) {
  return `"${value.replace(/(["\\$`])/g, "\\$1")}"`;
}

export function getAgentInstallScriptUrl() {
  if (typeof window === "undefined") {
    return "/agent-install.sh";
  }

  const baseUrl = new URL(import.meta.env.BASE_URL ?? "/", window.location.origin);
  return new URL("agent-install.sh", baseUrl).toString();
}

type AgentInstallCommandInput = {
  scriptUrl: string;
  serverUrl: string;
  token: string;
  nodeName?: string | null;
};

export function buildAgentInstallCommand({
  scriptUrl,
  serverUrl,
  token,
  nodeName,
}: AgentInstallCommandInput) {
  const args = [`--server-url ${shellQuote(serverUrl)}`, `--token ${shellQuote(token)}`];

  if (nodeName?.trim()) {
    args.push(`--node-name ${shellQuote(nodeName.trim())}`);
  }

  const lines = [`curl -fsSL ${scriptUrl} | bash -s -- \\`];
  lines.push(...args.map((arg, index) => `  ${arg}${index === args.length - 1 ? "" : " \\"}`));

  return lines.join("\n");
}
