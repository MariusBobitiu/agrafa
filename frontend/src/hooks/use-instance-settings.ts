import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { instanceSettingsApi } from "@/data/instance-settings.ts";
import type { InstanceSettingsPatchPayload } from "@/types/instance-setting.ts";

export function useInstanceSettings() {
  return useQuery({
    queryKey: ["instance-settings"],
    queryFn: () => instanceSettingsApi.list(),
  });
}

export function useSaveInstanceSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: InstanceSettingsPatchPayload) =>
      instanceSettingsApi.patch(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["instance-settings"] }),
  });
}
