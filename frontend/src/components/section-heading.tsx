type SectionHeadingProps = {
  icon: React.ReactNode;
  label: string;
  aside?: React.ReactNode;
  action?: React.ReactNode;
};

export function SectionHeading({ icon, label, aside, action }: SectionHeadingProps) {
  return (
    <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
      <div className="flex shrink-0 items-center gap-2">
        <span className="text-muted-foreground">{icon}</span>
        <h2 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
          {label}
        </h2>
        {aside}
      </div>
      {action}
    </div>
  );
}
