import Link from "next/link";
import { fetchPayments } from "@/lib/api";
import { PaymentsTable } from "@/components/PaymentsTable";

export const dynamic = "force-dynamic";

interface PageProps {
  searchParams: Promise<{
    merchant_id?: string;
    status?: string;
    offset?: string;
  }>;
}

export default async function DashboardPage(props: PageProps) {
  const searchParams = await props.searchParams;
  const merchantId = searchParams.merchant_id || "1001";
  const statusFilter = searchParams.status || "ALL";
  const limit = 20;
  const offset = parseInt(searchParams.offset || "0", 10) || 0;

  let paymentsData = null;
  let errorMsg: string | null = null;

  try {
    paymentsData = await fetchPayments(merchantId, statusFilter, limit, offset);
  } catch (err) {
    errorMsg = err instanceof Error ? err.message : "Error al conectar con la API de pagos";
  }

  const items = paymentsData?.items || [];
  const totalCount = paymentsData?.total_count || 0;

  // Simple KPI metrics calculation from current list
  const approvedCount = items.filter((i) => i.status === "APPROVED").length;
  const declinedCount = items.filter((i) => i.status === "DECLINED").length;
  const failedCount = items.filter((i) => i.status === "FAILED").length;

  const statuses = [
    { key: "ALL", label: "Todos los estados" },
    { key: "APPROVED", label: "Aprobadas" },
    { key: "DECLINED", label: "Rechazadas" },
    { key: "FAILED", label: "Fallidas" },
    { key: "PROCESSING", label: "Procesando" },
  ];

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">
            Transacciones del Comercio
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Monitoreo en tiempo real para el comercio{" "}
            <span className="font-mono text-indigo-400 font-semibold">{merchantId}</span>
          </p>
        </div>
        <div>
          <Link
            href={`/sandbox?merchant_id=${merchantId}`}
            className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg shadow-sm transition-colors"
          >
            <span>+ Simular Pago en Sandbox</span>
          </Link>
        </div>
      </div>

      {/* Error Banner if API is down */}
      {errorMsg && (
        <div className="p-4 rounded-xl border border-rose-500/30 bg-rose-950/20 text-rose-300 text-sm">
          <p className="font-semibold">No se pudo cargar el listado desde payments-api:</p>
          <p className="text-xs text-rose-400 mt-1 font-mono">{errorMsg}</p>
          <p className="text-xs text-slate-400 mt-2">
            Asegúrate de que la API esté corriendo en el puerto 8080 (o define{" "}
            <code>PAYMENTS_API_URL</code>).
          </p>
        </div>
      )}

      {/* Metrics Summary Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-5 rounded-xl border border-slate-800 bg-slate-900/50">
          <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">
            Total en Sistema
          </p>
          <p className="text-2xl font-bold text-white mt-2">{totalCount}</p>
          <p className="text-[11px] text-slate-500 mt-1">Transacciones registradas</p>
        </div>
        <div className="p-5 rounded-xl border border-emerald-900/30 bg-emerald-950/10">
          <p className="text-xs font-medium text-emerald-400 uppercase tracking-wider">
            Aprobadas (Página)
          </p>
          <p className="text-2xl font-bold text-emerald-400 mt-2">{approvedCount}</p>
          <p className="text-[11px] text-emerald-500/70 mt-1">Cobros autorizados con éxito</p>
        </div>
        <div className="p-5 rounded-xl border border-amber-900/30 bg-amber-950/10">
          <p className="text-xs font-medium text-amber-400 uppercase tracking-wider">
            Rechazadas (Página)
          </p>
          <p className="text-2xl font-bold text-amber-400 mt-2">{declinedCount}</p>
          <p className="text-[11px] text-amber-500/70 mt-1">Por fondos o validación</p>
        </div>
        <div className="p-5 rounded-xl border border-rose-900/30 bg-rose-950/10">
          <p className="text-xs font-medium text-rose-400 uppercase tracking-wider">
            Fallidas (Página)
          </p>
          <p className="text-2xl font-bold text-rose-400 mt-2">{failedCount}</p>
          <p className="text-[11px] text-rose-500/70 mt-1">Errores de sistema o timeout</p>
        </div>
      </div>

      {/* Status Filter Tabs */}
      <div className="flex flex-wrap items-center gap-2 border-b border-slate-800 pb-4">
        <span className="text-xs text-slate-400 mr-2">Filtrar por estado:</span>
        {statuses.map((s) => {
          const isActive = statusFilter === s.key;
          return (
            <Link
              key={s.key}
              href={`/?merchant_id=${merchantId}&status=${s.key}&offset=0`}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                isActive
                  ? "bg-indigo-600 text-white shadow-sm"
                  : "bg-slate-900 text-slate-400 hover:text-slate-200 hover:bg-slate-800"
              }`}
            >
              {s.label}
            </Link>
          );
        })}
      </div>

      {/* Transactions Table */}
      <PaymentsTable items={items} merchantId={merchantId} />

      {/* Pagination Controls */}
      {totalCount > limit && (
        <div className="flex items-center justify-between text-xs text-slate-400 pt-4">
          <span>
            Mostrando {offset + 1} a {Math.min(offset + limit, totalCount)} de {totalCount} resultados
          </span>
          <div className="flex items-center gap-2">
            {offset > 0 && (
              <Link
                href={`/?merchant_id=${merchantId}&status=${statusFilter}&offset=${Math.max(
                  0,
                  offset - limit
                )}`}
                className="px-3 py-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-lg text-slate-200"
              >
                ← Anterior
              </Link>
            )}
            {offset + limit < totalCount && (
              <Link
                href={`/?merchant_id=${merchantId}&status=${statusFilter}&offset=${offset + limit}`}
                className="px-3 py-1.5 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-lg text-slate-200"
              >
                Siguiente →
              </Link>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
