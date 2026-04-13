import { motion } from "framer-motion";
import type { ReactNode } from "react";

type ErrorPageShellProps = {
  ghost: string;
  icon: ReactNode;
  title: string;
  description: string;
  actions: ReactNode;
};

export function ErrorPageShell({ ghost, icon, title, description, actions }: ErrorPageShellProps) {
  return (
    <div className="relative flex h-full min-h-[70vh] flex-col items-center justify-center overflow-hidden px-6 py-16 select-none">
      {/* Background grid */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-[0.03] dark:opacity-[0.06]"
        style={{
          backgroundImage:
            "linear-gradient(var(--foreground) 1px, transparent 1px), linear-gradient(90deg, var(--foreground) 1px, transparent 1px)",
          backgroundSize: "40px 40px",
        }}
      />
      {/* Radial fade */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 60% 55% at 50% 50%, transparent 40%, var(--background) 100%)",
        }}
      />

      <div className="relative z-10 flex flex-col items-center text-center">
        {/* Ghost text + icon */}
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.4, ease: "easeOut" }}
          className="relative mb-6"
        >
          <span
            aria-hidden
            className="text-[clamp(7rem,20vw,12rem)] font-black leading-none tracking-tighter text-foreground/6 dark:text-foreground/8"
          >
            {ghost}
          </span>
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl border border-border bg-card shadow-sm">
              {icon}
            </div>
          </div>
        </motion.div>

        {/* Text */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, delay: 0.1, ease: "easeOut" }}
        >
          <h1 className="text-2xl font-bold tracking-tight text-foreground">{title}</h1>
          <p className="mt-2 max-w-xs text-sm text-muted-foreground">{description}</p>
        </motion.div>

        {/* Actions */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, delay: 0.2, ease: "easeOut" }}
          className="mt-8 flex items-center gap-3"
        >
          {actions}
        </motion.div>
      </div>
    </div>
  );
}
