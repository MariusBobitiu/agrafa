export type InstanceSettingType = "bool" | "string";

export type InstanceSetting = {
  key: string;
  group: string;
  label: string;
  description: string;
  type: InstanceSettingType;
  value: unknown;
  is_sensitive: boolean;
  is_encrypted: boolean;
  is_env_overridden: boolean;
  is_editable: boolean;
  is_configured?: boolean;
};

export type InstanceSettingsResponse = {
  settings: InstanceSetting[];
};

export type InstanceSettingPatchItem = {
  key: string;
  value: unknown;
};

export type InstanceSettingsPatchPayload = {
  settings: InstanceSettingPatchItem[];
};
