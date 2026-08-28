export type SettingsTab = "notifications" | "alert-rules" | "members" | "project" | "danger-zone";

export function settingsTabFromSearchParams(
  searchParams: URLSearchParams,
  canDeleteProject: boolean,
): SettingsTab {
  const tab = searchParams.get("tab");
  if (tab === "alert-rules" || tab === "members" || tab === "project") return tab;
  if (tab === "danger-zone" && canDeleteProject) return tab;
  return "notifications";
}
