import { EventItem } from "@/types/payment";
import { formatDate } from "@/lib/formatters";

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

  const getEventBadge = (type: string) => {
    switch (type) {
      case "TRANSACTION_CREATED":
        return {
          title: "Transacción Creada",
          color: "bg-blue-500/10 text-blue-400 border-blue-500/30",
          icon: "📝",
        };
      case "STATUS_CHANGED":
        return {
          title: "Cambio de Estado",
          color: "bg-amber-500/10 text-amber-400 border-amber-500/30",
          icon: "🔄",
        };
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
    <div className="relative pl-6 space-y-8 before:absolute before:left-2.5 before:top-3 before:bottom-3 before:w-0.5 before:bg-slate-800">
      {events.map((e, index) => {
        const meta = getEventBadge(e.event_type);
        const payload = e.payload || {};

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

              {/* Event Specific Highlights */}
              {e.event_type === "ACQUIRER_ATTEMPT" && (
                <div className="flex flex-wrap gap-2 my-2 text-xs">
                  {"latency_ms" in payload && (
                    <span className="px-2 py-1 bg-slate-800 text-slate-300 rounded border border-slate-700 font-mono">
                      ⏱ Latencia: {String(payload.latency_ms)} ms
                    </span>
                  )}
                  {"code" in payload && (
                    <span className="px-2 py-1 bg-slate-800 text-slate-300 rounded border border-slate-700 font-mono">
                      Código Banco: {String(payload.code)}
                    </span>
                  )}
                  {"message" in payload && (
                    <span className="px-2 py-1 bg-slate-800 text-slate-300 rounded border border-slate-700">
                      Mensaje: {String(payload.message)}
                    </span>
                  )}
                </div>
              )}

              {e.event_type === "STATUS_CHANGED" && (
                <div className="flex items-center gap-2 my-2 text-xs font-mono">
                  {payload.from_status ? (
                    <>
                      <span className="px-2 py-0.5 bg-slate-800 text-slate-400 rounded">
                        {String(payload.from_status)}
                      </span>
                      <span className="text-slate-500">→</span>
                      <span className="px-2 py-0.5 bg-indigo-950 text-indigo-300 rounded border border-indigo-800">
                        {String(payload.to_status)}
                      </span>
                    </>
                  ) : payload.to_status ? (
                    <span className="px-2 py-0.5 bg-indigo-950 text-indigo-300 rounded border border-indigo-800">
                      Estado: {String(payload.to_status)}
                    </span>
                  ) : null}
                </div>
              )}

              {/* Collapsed/Formatted payload details */}
              {Object.keys(payload).length > 0 && (
                <details className="mt-3 text-xs text-slate-400 cursor-pointer">
                  <summary className="hover:text-slate-200 select-none">
                    Ver payload técnico completo
                  </summary>
                  <pre className="mt-2 p-3 bg-slate-950/80 rounded border border-slate-800/60 overflow-x-auto text-[11px] font-mono text-slate-300">
                    {JSON.stringify(payload, null, 2)}
                  </pre>
                </details>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
