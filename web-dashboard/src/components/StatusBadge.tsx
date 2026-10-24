import { TransactionStatus } from "@/types/payment";

interface StatusBadgeProps {
  status: TransactionStatus | string;
  size?: "sm" | "md";
}

export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const normalized = status.toUpperCase();

  const configs: Record<
    string,
    { label: string; bg: string; text: string; border: string; dot: string }
  > = {
    APPROVED: {
      label: "Aprobada",
      bg: "bg-emerald-950/40",
      text: "text-emerald-400",
      border: "border-emerald-500/30",
      dot: "bg-emerald-400",
    },
    DECLINED: {
      label: "Rechazada",
      bg: "bg-amber-950/40",
      text: "text-amber-400",
      border: "border-amber-500/30",
      dot: "bg-amber-400",
    },
    FAILED: {
      label: "Fallida",
      bg: "bg-rose-950/40",
      text: "text-rose-400",
      border: "border-rose-500/30",
      dot: "bg-rose-400",
    },
    PROCESSING: {
      label: "Procesando",
      bg: "bg-blue-950/40",
      text: "text-blue-400",
      border: "border-blue-500/30",
      dot: "bg-blue-400 animate-ping",
    },
    PENDING: {
      label: "Pendiente",
      bg: "bg-slate-900/60",
      text: "text-slate-400",
      border: "border-slate-700/40",
      dot: "bg-slate-400",
    },
  };

  const cfg = configs[normalized] || {
    label: normalized,
    bg: "bg-slate-900/50",
    text: "text-slate-300",
    border: "border-slate-700",
    dot: "bg-slate-400",
  };

  const sizeClass = size === "sm" ? "px-2 py-0.5 text-xs" : "px-2.5 py-1 text-xs";

  return (
    <span
      className={`inline-flex items-center gap-1.5 font-medium rounded-full border ${cfg.bg} ${cfg.text} ${cfg.border} ${sizeClass}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />
      {cfg.label}
    </span>
  );
}
