import { useEffect, useEffectEvent } from "react";
import { getApiBaseUrl } from "@/lib/fetch-client.ts";

type UseSSEOptions<T> = {
  enabled: boolean;
  path: string;
  onMessage: (payload: T) => void;
  onError?: (event: Event) => void;
};

function getSseUrl(path: string) {
  return `${getApiBaseUrl()}/v1${path}`;
}

export function useSSE<T>({ enabled, path, onMessage, onError }: UseSSEOptions<T>) {
  const handleMessage = useEffectEvent((event: MessageEvent<string>) => {
    if (!event.data) return;

    try {
      onMessage(JSON.parse(event.data) as T);
    } catch {
      // Ignore malformed stream payloads and keep the page usable.
    }
  });

  const handleError = useEffectEvent((event: Event) => {
    onError?.(event);
  });

  useEffect(() => {
    if (!enabled) return;

    const source = new EventSource(getSseUrl(path), { withCredentials: true });
    source.onmessage = handleMessage;
    source.onerror = handleError;

    return () => {
      source.close();
    };
  }, [enabled, path]);
}
