import { api } from "@/lib/fetch-client.ts";
import type {
  InstanceSettingsPatchPayload,
  InstanceSettingsResponse,
} from "@/types/instance-setting.ts";

export const instanceSettingsApi = {
  list: (): Promise<InstanceSettingsResponse> =>
    api.get("/instance-settings"),

  patch: (payload: InstanceSettingsPatchPayload): Promise<InstanceSettingsResponse> =>
    api.patch("/instance-settings", payload),
};
