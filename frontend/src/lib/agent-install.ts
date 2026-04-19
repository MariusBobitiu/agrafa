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
  const lines = [
    `curl -fsSL ${scriptUrl} | bash -s -- \\`,
    `  --server-url ${shellQuote(serverUrl)} \\`,
    `  --token ${shellQuote(token)}`,
  ];

  if (nodeName?.trim()) {
    lines.push(`  --node-name ${shellQuote(nodeName.trim())}`);
  }

  return lines.join("\n");
}
