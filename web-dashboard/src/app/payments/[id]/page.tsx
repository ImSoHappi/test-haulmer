import Link from "next/link";
import { notFound } from "next/navigation";
import { fetchPaymentById } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import { EventTimeline } from "@/components/EventTimeline";
import { formatDate } from "@/lib/formatters";

export const dynamic = "force-dynamic";

interface DetailPageProps {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ merchant_id?: string }>;
}

export default async function PaymentDetailPage(props: DetailPageProps) {
  const params = await props.params;
  const searchParams = await props.searchParams;
  const merchantId = searchParams.merchant_id || "1001";
  const id = params.id;

  let payment = null;
  let errorMsg: string | null = null;

  try {
    payment = await fetchPaymentById(id, merchantId);
  } catch (err) {
    errorMsg = err instanceof Error ? err.message : "No se pudo cargar la transacción";
  }

  if (errorMsg && errorMsg.includes("404")) {
    notFound();
  }

  return (
    <div className="space-y-8 max-w-4xl mx-auto">
      {/* Back button and page title */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <Link
            href={`/?merchant_id=${merchantId}`}
            className="text-xs text-indigo-400 hover:text-indigo-300 inline-flex items-center gap-1 mb-2"
          >
            ← Volver al listado
          </Link>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white tracking-tight">
              Detalle de Transacción
            </h1>
            {payment && <StatusBadge status={payment.status} />}
          </div>
          <p className="text-xs text-slate-400 mt-1 font-mono">
            ID: <span className="text-slate-300">{id}</span>
          </p>
        </div>

        <Link
          href={`/sandbox?merchant_id=${merchantId}`}
          className="px-3 py-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-700 text-xs font-medium rounded-lg text-slate-200 self-start sm:self-auto"
        >
          Probar otra en Sandbox
        </Link>
      </div>

      {errorMsg && (
        <div className="p-4 rounded-xl border border-rose-500/30 bg-rose-950/20 text-rose-300 text-sm">
          <p className="font-semibold">Error al recuperar la transacción:</p>
          <p className="text-xs text-rose-400 mt-1 font-mono">{errorMsg}</p>
        </div>
      )}

      {payment && (
        <>
          {/* Main Info Card */}
          <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6 shadow-sm">
            <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider mb-4">
              Atributos de Pago & Conciliación
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
              <div>
                <span className="text-slate-500 block">Comercio (Tenant)</span>
                <span className="font-mono text-slate-200 font-medium">
                  {payment.merchant_id}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block">Fecha de Registro</span>
                <span className="text-slate-200 font-medium">
                  {formatDate(payment.created_at)}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block">Tarjeta Utilizada</span>
                <span className="font-mono text-slate-200 font-medium">
                  {payment.card_bin || "••••••"} •••••• {payment.card_last4 || "••••"}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block">Código de Autorización</span>
                <span className="font-mono text-emerald-400 font-semibold">
                  {payment.authorization_code || "Sin autorización (Rechazada/Pendiente)"}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block">Clave de Idempotencia</span>
                <span className="font-mono text-slate-300 truncate block">
                  {payment.idempotency_key}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block">Correlation ID (Trazabilidad)</span>
                <span className="font-mono text-indigo-400 truncate block">
                  {payment.correlation_id || "N/A"}
                </span>
              </div>

              {payment.decline_source && (
                <div>
                  <span className="text-slate-500 block">Origen del Rechazo</span>
                  <span className="font-medium text-amber-400">
                    {payment.decline_source}
                  </span>
                </div>
              )}

              {payment.result_code && (
                <div>
                  <span className="text-slate-500 block">Código de Resultado</span>
                  <span className="font-mono text-slate-300">
                    {payment.result_code}: {payment.result_message || ""}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Audit Event Timeline */}
          <div className="bg-slate-900/30 border border-slate-800 rounded-xl p-6 shadow-sm">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">
                  Línea de Tiempo de Auditoría (Inmutable)
                </h2>
                <p className="text-xs text-slate-500 mt-0.5">
                  Secuencia exacta de eventos persistidos en PostgreSQL
                </p>
              </div>
              <span className="text-xs font-mono text-slate-400 bg-slate-800 px-2 py-0.5 rounded">
                {payment.events?.length || 0} eventos
              </span>
            </div>

            <EventTimeline events={payment.events || []} />
          </div>
        </>
      )}
    </div>
  );
}
