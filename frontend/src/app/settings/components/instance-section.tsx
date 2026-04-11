import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Separator } from "@/components/ui/separator.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { useInstanceSettings, useSaveInstanceSettings } from "@/hooks/use-instance-settings.ts";
import type { InstanceSetting, InstanceSettingPatchItem } from "@/types/instance-setting.ts";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function groupByGroup(settings: InstanceSetting[]): Record<string, InstanceSetting[]> {
  return settings.reduce<Record<string, InstanceSetting[]>>((acc, s) => {
    (acc[s.group] ??= []).push(s);
    return acc;
  }, {});
}

// ─── Individual setting row ───────────────────────────────────────────────────

function SettingRow({
  setting,
  onChange,
}: {
  setting: InstanceSetting;
  onChange: (key: string, value: unknown) => void;
}) {
  const [passwordDraft, setPasswordDraft] = useState("");

  const isDisabled = !setting.is_editable;

  function handleBoolChange(checked: boolean) {
    onChange(setting.key, checked);
  }

  function handleStringChange(val: string) {
    if (setting.is_sensitive) {
      setPasswordDraft(val);
      // signal change via sentinel — resolved in parent before sending
      onChange(setting.key, val === "" ? null : val);
    } else {
      onChange(setting.key, val);
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-start justify-between gap-6">
        {/* Label + description */}
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-foreground leading-snug">{setting.label}</p>
          {setting.description && (
            <p className="mt-0.5 text-xs text-muted-foreground leading-relaxed">
              {setting.description}
            </p>
          )}
        </div>

        {/* Control */}
        <div className="shrink-0 flex items-center pt-0.5">
          {setting.type === "bool" ? (
            <Switch
              checked={setting.value === true}
              onCheckedChange={handleBoolChange}
              disabled={isDisabled}
            />
          ) : setting.is_sensitive ? (
            <div className="flex flex-col items-end gap-1">
              <Input
                type="password"
                placeholder={setting.is_configured ? "••••••••" : "Enter value"}
                value={passwordDraft}
                onChange={(e) => handleStringChange(e.target.value)}
                disabled={isDisabled}
                className="w-56 text-sm"
                autoComplete="new-password"
              />
              {setting.is_configured && passwordDraft === "" && (
                <span className="text-[11px] text-muted-foreground/60 font-medium">Configured</span>
              )}
            </div>
          ) : (
            <Input
              type="text"
              value={typeof setting.value === "string" ? setting.value : ""}
              onChange={(e) => handleStringChange(e.target.value)}
              disabled={isDisabled}
              className="w-56 text-sm"
            />
          )}
        </div>
      </div>

      {/* Env override notice */}
      {setting.is_env_overridden && (
        <p className="text-[11px] text-amber-400/80 leading-snug">
          This value is currently overridden by environment configuration.
        </p>
      )}
    </div>
  );
}

// ─── Email group card ─────────────────────────────────────────────────────────

function EmailSettingsCard({
  settings,
  onSettingChange,
}: {
  settings: InstanceSetting[];
  onSettingChange: (key: string, value: unknown) => void;
}) {
  return (
    <div className="rounded-xl border border-border overflow-hidden">
      <div className="px-6 py-5">
        <h2 className="text-sm font-semibold">Email</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          Configure outbound email delivery for notifications and alerts.
        </p>
      </div>
      <Separator />
      <div className="px-6 py-5 space-y-6">
        {settings.map((s) => (
          <SettingRow key={s.key} setting={s} onChange={onSettingChange} />
        ))}
      </div>
    </div>
  );
}

// ─── Section ──────────────────────────────────────────────────────────────────

export function InstanceSection() {
  const { data, isLoading } = useInstanceSettings();
  const save = useSaveInstanceSettings();

  // pending overrides: key → new value (null means "sensitive field left blank, skip")
  const [overrides, setOverrides] = useState<Map<string, unknown>>(new Map());

  function handleSettingChange(key: string, value: unknown) {
    setOverrides((prev) => {
      const next = new Map(prev);
      next.set(key, value);
      return next;
    });
  }

  // Merge overrides into the displayed settings so controls show updated state
  const emailSettings = (data?.settings ?? [])
    .filter((s) => s.group === "email")
    .map((s) => {
      if (!overrides.has(s.key)) return s;
      // For bool settings, reflect the override value directly
      if (s.type === "bool") return { ...s, value: overrides.get(s.key) };
      // For non-sensitive strings, reflect the override
      if (!s.is_sensitive) return { ...s, value: overrides.get(s.key) };
      // For sensitive fields, keep masked — the input manages its own draft state
      return s;
    });

  async function handleSave() {
    if (!data) return;

    const originalMap = new Map(data.settings.map((s) => [s.key, s]));
    const patch: InstanceSettingPatchItem[] = [];

    for (const [key, newValue] of overrides.entries()) {
      const original = originalMap.get(key);
      if (!original) continue;

      if (original.is_sensitive) {
        // Only send if user typed something (non-empty, non-null)
        if (newValue !== null && newValue !== "") {
          patch.push({ key, value: newValue });
        }
        // else: left blank, skip
      } else {
        // Send if changed
        if (newValue !== original.value) {
          patch.push({ key, value: newValue });
        }
      }
    }

    if (patch.length === 0) {
      toast.info("No changes to save.");
      return;
    }

    await save.mutateAsync(
      { settings: patch },
      {
        onSuccess: () => {
          toast.success("Instance settings saved.");
          setOverrides(new Map());
        },
        onError: () => toast.error("Couldn't save settings. Try again."),
      },
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-48 w-full rounded-xl" />
      </div>
    );
  }

  const grouped = groupByGroup(data?.settings ?? []);
  const hasEmail = (grouped["email"]?.length ?? 0) > 0;

  return (
    <div className="space-y-6">
      {hasEmail && (
        <EmailSettingsCard
          settings={emailSettings}
          onSettingChange={handleSettingChange}
        />
      )}

      <div className="flex justify-end">
        <Button
          size="sm"
          onClick={handleSave}
          disabled={save.isPending || overrides.size === 0}
        >
          {save.isPending ? "Saving…" : "Save"}
        </Button>
      </div>
    </div>
  );
}
