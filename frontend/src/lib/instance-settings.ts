import type { InstanceSetting } from "@/types/instance-setting.ts";

const EMAIL_ENABLED_KEY = "email.enabled";
const EMAIL_PROVIDER_KEY = "email.provider";
const EMAIL_RESEND_API_KEY = "email.resend_api_key";
const EMAIL_RESEND_DOMAIN_KEY = "email.resend_domain";

function getInstanceSetting(settings: InstanceSetting[], key: string) {
  return settings.find((setting) => setting.key === key);
}

function getBooleanValue(setting: InstanceSetting | undefined) {
  return setting?.value === true;
}

function getStringValue(setting: InstanceSetting | undefined) {
  return typeof setting?.value === "string" ? setting.value.trim() : "";
}

export function isEmailDeliveryAvailableOnInstance(settings: InstanceSetting[]) {
  const emailEnabled = getBooleanValue(getInstanceSetting(settings, EMAIL_ENABLED_KEY));
  if (!emailEnabled) {
    return false;
  }

  const provider = getStringValue(getInstanceSetting(settings, EMAIL_PROVIDER_KEY));
  if (!provider) {
    return false;
  }

  switch (provider) {
    case "resend":
      return Boolean(
        getInstanceSetting(settings, EMAIL_RESEND_API_KEY)?.is_configured &&
        getStringValue(getInstanceSetting(settings, EMAIL_RESEND_DOMAIN_KEY)),
      );
    default:
      return false;
  }
}
