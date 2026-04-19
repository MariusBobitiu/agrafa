import { CopyButton } from "@/components/animate-ui/components/buttons/copy";
import { toast } from "sonner";

type CodeBlockProps = {
  label?: string;
  value: string;
};

export function CodeBlock({ label, value }: CodeBlockProps) {
  return (
    <div className="flex-1 space-y-2">
      {label && (
        <p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground">
          {label}
        </p>
      )}
      <div className="relative min-w-0 overflow-hidden rounded-xl border border-border/70 bg-muted/20">
        <pre className="max-h-48 overflow-auto px-4 py-3 pr-14 text-sm leading-6 text-foreground">
          {value}
        </pre>
        <CopyButton
          content={value}
          variant="secondary"
          size="sm"
          className="absolute right-2 top-2"
          aria-label="Copy command"
          onCopyError={() => {
            toast.error("Couldn't copy the command. Copy it manually.");
          }}
        />
      </div>
    </div>
  );
}
