import Link from "next/link";
import { notFound } from "next/navigation";
import { fetchPaymentById } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import { EventTimeline } from "@/components/EventTimeline";
import { CopyButton } from "@/components/CopyButton";
import { formatDate, formatCLP } from "@/lib/formatters";

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

  const maskedCard = payment
    ? `${payment.card_bin || "••••••"} •••••• ${payment.card_last4 || "••••"}`
    : "";

  const sandboxRetryUrl = payment
    ? `/sandbox?merchant_id=${merchantId}&order_reference=${encodeURIComponent(
        payment.order_reference || ""
      )}&amount=${payment.amount || ""}&idempotency_key=${encodeURIComponent(
        payment.idempotency_key || ""
      )}`
    : `/sandbox?merchant_id=${merchantId}`;

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      {/* Header bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-slate-800/80">
        <div>
          <Link
            href={`/?merchant_id=${merchantId}`}
            className="text-xs text-indigo-400 hover:text-indigo-300 inline-flex items-center gap-1 mb-2"
          >
            ← Volver al listado de pagos
          </Link>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white tracking-tight">
              Detalle de Transacción
            </h1>
            {payment && <StatusBadge status={payment.status} />}
          </div>
          <div className="flex items-center gap-1.5 text-xs text-slate-400 mt-1 font-mono">
            <span>ID:</span>
            <span className="text-slate-300">{id}</span>
            <CopyButton text={id} label="Copiar ID de transacción" />
          </div>
        </div>

        {payment && (
          <Link
            href={sandboxRetryUrl}
            className="inline-flex items-center gap-2 px-3.5 py-2 bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/40 text-xs font-semibold rounded-lg shadow-sm transition-colors self-start sm:self-auto"
          >
            <span>⚡ Probar en Sandbox con estos datos</span>
          </Link>
        )}
      </div>

      {errorMsg && (
        <div className="p-4 rounded-xl border border-rose-500/30 bg-rose-950/20 text-rose-300 text-sm">
          <p className="font-semibold">Error al recuperar la transacción:</p>
          <p className="text-xs text-rose-400 mt-1 font-mono">{errorMsg}</p>
        </div>
      )}

      {payment && (
        <>
          {/* Order Summary & Financial Banner */}
          {payment.amount !== undefined && payment.amount > 0 && (
            <div className="bg-gradient-to-r from-slate-900 via-slate-900 to-indigo-950/30 border border-slate-800 rounded-xl p-5 flex flex-wrap items-center justify-between gap-4 shadow-sm">
              <div>
                <span className="text-[11px] text-slate-400 uppercase tracking-wider block font-medium">
                  Monto de la Orden
                </span>
                <span className="text-2xl font-bold text-white tracking-tight">
                  {formatCLP(payment.amount)}
                </span>
                <span className="text-xs text-slate-400 ml-2 font-mono">
                  {payment.currency || "CLP"}
                </span>
              </div>

              {payment.order_reference && (
                <div className="bg-slate-950/60 px-3.5 py-2 rounded-lg border border-slate-800 flex items-center gap-2">
                  <div>
                    <span className="text-[10px] text-slate-400 block uppercase">
                      Orden Comercial
                    </span>
                    <span className="font-mono text-xs text-indigo-300 font-semibold">
                      {payment.order_reference}
                    </span>
                  </div>
                  <CopyButton
                    text={payment.order_reference}
                    label="Copiar referencia de orden"
                  />
                </div>
              )}
            </div>
          )}

          {/* Main Info Card */}
          <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6 shadow-sm">
            <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-4">
              Atributos de Pago & Conciliación
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
              <div>
                <span className="text-slate-500 block mb-0.5">Comercio (Tenant)</span>
                <span className="font-mono text-slate-200 font-medium">
                  {payment.merchant_id}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block mb-0.5">Fecha de Registro</span>
                <span className="text-slate-200 font-medium">
                  {formatDate(payment.created_at)}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block mb-0.5">Tarjeta Utilizada</span>
                <div className="flex items-center gap-1.5">
                  <span className="font-mono text-slate-200 font-medium">
                    {maskedCard}
                  </span>
                  <CopyButton text={maskedCard} label="Copiar tarjeta enmascarada" />
                </div>
              </div>

              <div>
                <span className="text-slate-500 block mb-0.5">Código de Autorización</span>
                <span className="font-mono text-emerald-400 font-semibold">
                  {payment.authorization_code || "Sin autorización"}
                </span>
              </div>

              <div>
                <span className="text-slate-500 block mb-0.5">Clave de Idempotencia</span>
                <div className="flex items-center gap-1.5">
                  <span className="font-mono text-slate-300 truncate max-w-[240px]">
                    {payment.idempotency_key}
                  </span>
                  <CopyButton text={payment.idempotency_key} label="Copiar clave de idempotencia" />
                </div>
              </div>

              <div>
                <span className="text-slate-500 block mb-0.5">Correlation ID (Trazabilidad)</span>
                <div className="flex items-center gap-1.5">
                  <span className="font-mono text-indigo-400 truncate max-w-[240px]">
                    {payment.correlation_id || "N/A"}
                  </span>
                  {payment.correlation_id && (
                    <CopyButton text={payment.correlation_id} label="Copiar Correlation ID" />
                  )}
                </div>
              </div>

              {payment.decline_source && (
                <div>
                  <span className="text-slate-500 block mb-0.5">Origen del Rechazo</span>
                  <span className="font-medium text-amber-400">
                    {payment.decline_source}
                  </span>
                </div>
              )}

              {payment.result_code && (
                <div>
                  <span className="text-slate-500 block mb-0.5">Código de Resultado</span>
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
