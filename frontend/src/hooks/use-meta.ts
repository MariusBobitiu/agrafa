// hooks/useMeta.ts
import { useEffect } from "react";

export function useMeta({ title, description }: { title?: string; description?: string }) {
  useEffect(() => {
    if (title) {
      document.title = `${title} • Agrafa`;
    }

    if (description) {
      let descriptionMeta = document.querySelector('meta[name="description"]');

      if (!descriptionMeta) {
        descriptionMeta = document.createElement("meta");
        descriptionMeta.setAttribute("name", "description");
        document.head.appendChild(descriptionMeta);
      }

      descriptionMeta.setAttribute("content", description);
    }
  }, [title, description]);
}
