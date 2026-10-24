import { EventItem } from "@/types/payment";
import { formatDate } from "@/lib/formatters";
import { CopyButton } from "@/components/CopyButton";

interface EventTimelineProps {
  events: EventItem[];
}

export function EventTimeline({ events }: EventTimelineProps) {
  if (!events || events.length === 0) {
    return (
      <div className="p-8 text-center text-slate-500 bg-slate-900/30 rounded-xl border border-slate-800">
        No se registraron eventos para esta transacción.
      </div>
    );
  }

  const getEventBadge = (type: string, payload: Record<string, unknown>) => {
    switch (type) {
      case "TRANSACTION_CREATED":
        return {
          title: "Transacción Inicializada",
          color: "bg-blue-500/10 text-blue-400 border-blue-500/30",
          icon: "📝",
        };
      case "STATUS_CHANGED": {
        const toStatus = String(payload.to_status || "");
        if (toStatus === "APPROVED") {
          return {
            title: "Pago Aprobado",
            color: "bg-emerald-500/10 text-emerald-400 border-emerald-500/30",
            icon: "✅",
          };
        }
        if (toStatus === "DECLINED") {
          return {
            title: "Pago Rechazado",
            color: "bg-rose-500/10 text-rose-400 border-rose-500/30",
            icon: "❌",
          };
        }
        if (toStatus === "FAILED") {
          return {
            title: "Fallo de Infraestructura",
            color: "bg-amber-500/10 text-amber-400 border-amber-500/30",
            icon: "⚠️",
          };
        }
        return {
          title: "Cambio de Estado",
          color: "bg-indigo-500/10 text-indigo-400 border-indigo-500/30",
          icon: "🔄",
        };
      }
      case "ACQUIRER_ATTEMPT":
        return {
          title: "Intento Bancario (Adquirente)",
          color: "bg-purple-500/10 text-purple-400 border-purple-500/30",
          icon: "🏦",
        };
      default:
        return {
          title: type,
          color: "bg-slate-500/10 text-slate-400 border-slate-500/30",
          icon: "⚡",
        };
    }
  };

  return (
    <div className="relative pl-6 space-y-6 before:absolute before:left-2.5 before:top-3 before:bottom-3 before:w-0.5 before:bg-slate-800">
      {events.map((e, index) => {
        const payload = e.payload || {};
        const meta = getEventBadge(e.event_type, payload);
        const correlationId = payload.correlation_id ? String(payload.correlation_id) : "";

        return (
          <div key={e.id || index} className="relative group">
            {/* Timeline node */}
            <div className="absolute -left-6 top-1 w-5 h-5 rounded-full bg-slate-900 border-2 border-slate-700 flex items-center justify-center text-xs group-hover:border-indigo-500 transition-colors">
              <span className="text-[10px]">{meta.icon}</span>
            </div>

            <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4 shadow-sm hover:border-slate-700 transition-colors">
              <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
                <div className="flex items-center gap-2">
                  <span
                    className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold border ${meta.color}`}
                  >
                    {meta.title}
                  </span>
                  <span className="text-xs text-slate-500">#{e.id}</span>
                </div>
                <time className="text-xs text-slate-400">
                  {formatDate(e.created_at)}
                </time>
              </div>

              {/* Status Change Details */}
              {e.event_type === "STATUS_CHANGED" && (
                <div className="space-y-2 my-2">
                  <div className="flex items-center gap-2 text-xs font-mono">
                    <span className="text-slate-400">Transición:</span>
                    {payload.from_status ? (
                      <>
                        <span className="px-2 py-0.5 bg-slate-800 text-slate-300 rounded border border-slate-700">
                          {String(payload.from_status)}
                        </span>
                        <span className="text-slate-500">→</span>
                        <span className="px-2 py-0.5 bg-indigo-950 text-indigo-300 rounded border border-indigo-800 font-semibold">
                          {String(payload.to_status)}
                        </span>
                      </>
                    ) : (
                      <span className="px-2 py-0.5 bg-indigo-950 text-indigo-300 rounded border border-indigo-800 font-semibold">
                        {String(payload.to_status || "Desconocido")}
                      </span>
                    )}
                  </div>

                  {/* Diagnostic / Error message if present */}
                  {(Boolean(payload.message) || Boolean(payload.code)) && (
                    <div className="p-2.5 rounded-lg bg-slate-950/60 border border-slate-800 text-xs">
                      <div className="flex items-center gap-2">
                        <span className="text-slate-400">Motivo / Mensaje:</span>
                        <span className="font-medium text-slate-200">
                          {payload.message ? String(payload.message) : "Sin mensaje adicional"}
                        </span>
                        {Boolean(payload.code) ? (
                          <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[11px] border border-slate-700">
                            Código: {String(payload.code)}
                          </span>
                        ) : null}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Acquirer Attempt Highlights */}
              {e.event_type === "ACQUIRER_ATTEMPT" && (
                <div className="flex flex-wrap items-center gap-2 my-2 text-xs">
                  {Boolean(payload.acquirer_attempt) ? (
                    <span className="px-2 py-1 bg-slate-800 text-purple-300 rounded border border-slate-700 font-mono">
                      Intento #{String(payload.acquirer_attempt)}
                    </span>
                  ) : null}
                  {payload.latency_ms !== undefined ? (
                    <span className="px-2 py-1 bg-slate-800 text-slate-300 rounded border border-slate-700 font-mono">
                      ⏱ Latencia: {String(payload.latency_ms)} ms
                    </span>
                  ) : null}
                  {Boolean(payload.code) ? (
                    <span className="px-2 py-1 bg-slate-800 text-slate-300 rounded border border-slate-700 font-mono">
                      Código Banco: {String(payload.code)}
                    </span>
                  ) : null}
                  {Boolean(payload.message) ? (
                    <span className="px-2 py-1 bg-slate-800 text-slate-300 rounded border border-slate-700">
                      {String(payload.message)}
                    </span>
                  ) : null}
                </div>
              )}

              {/* Correlation ID & Raw details */}
              <div className="flex flex-wrap items-center justify-between gap-2 mt-3 pt-2 border-t border-slate-800/60 text-[11px]">
                {correlationId ? (
                  <div className="flex items-center gap-1.5 text-slate-400 font-mono">
                    <span className="text-slate-500">Correlation ID:</span>
                    <span className="text-indigo-400">{correlationId}</span>
                    <CopyButton text={correlationId} label="Copiar Correlation ID" />
                  </div>
                ) : (
                  <div />
                )}

                {Object.keys(payload).length > 0 && (
                  <details className="text-slate-400 cursor-pointer">
                    <summary className="hover:text-slate-200 select-none">
                      Ver JSON crudo
                    </summary>
                    <pre className="mt-2 p-2.5 bg-slate-950 rounded border border-slate-800/80 overflow-x-auto text-[11px] font-mono text-slate-300">
                      {JSON.stringify(payload, null, 2)}
                    </pre>
                  </details>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

