import Link from "next/link";
import { TransactionListItem } from "@/types/payment";
import { StatusBadge } from "@/components/StatusBadge";
import { formatTimeAgo } from "@/lib/formatters";

interface PaymentsTableProps {
  items: TransactionListItem[];
  merchantId: string;
}

export function PaymentsTable({ items, merchantId }: PaymentsTableProps) {
  if (!items || items.length === 0) {
    return (
      <div className="text-center py-12 px-4 bg-slate-900/40 rounded-xl border border-slate-800">
        <p className="text-slate-400 text-sm">No se encontraron transacciones registradas.</p>
        <p className="text-slate-500 text-xs mt-1">
          Puedes disparar pagos de prueba desde el{" "}
          <Link
            href={`/sandbox?merchant_id=${merchantId}`}
            className="text-indigo-400 underline hover:text-indigo-300"
          >
            Simulador Sandbox
          </Link>
          .
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/30">
      <table className="w-full text-left border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-800 bg-slate-900/60 text-xs font-semibold text-slate-400 uppercase tracking-wider">
            <th className="py-3 px-4">Fecha / Hora</th>
            <th className="py-3 px-4">ID Transacción</th>
            <th className="py-3 px-4">Tarjeta</th>
            <th className="py-3 px-4">Código Auth / Info</th>
            <th className="py-3 px-4 text-center">Estado</th>
            <th className="py-3 px-4 text-right">Detalle</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800/60">
          {items.map((tx) => (
            <tr
              key={tx.id}
              className="hover:bg-slate-800/30 transition-colors group"
            >
              <td className="py-3.5 px-4 whitespace-nowrap text-xs text-slate-400">
                {formatTimeAgo(tx.created_at)}
              </td>
              <td className="py-3.5 px-4 whitespace-nowrap font-mono text-xs text-slate-300">
                {tx.id.slice(0, 8)}...
              </td>
              <td className="py-3.5 px-4 whitespace-nowrap text-slate-300 font-mono text-xs">
                •••• {tx.card_last4 || "••••"}
                <span className="text-slate-500 text-[10px] ml-1.5 font-normal">
                  ({tx.card_bin || "N/A"})
                </span>
              </td>
              <td className="py-3.5 px-4 whitespace-nowrap text-xs text-slate-300 font-mono">
                {tx.authorization_code ? (
                  <span className="text-emerald-400">{tx.authorization_code}</span>
                ) : tx.result_code ? (
                  <span className="text-amber-400/90">
                    {tx.result_code}: {tx.result_message || tx.decline_source}
                  </span>
                ) : (
                  <span className="text-slate-500">-</span>
                )}
              </td>
              <td className="py-3.5 px-4 whitespace-nowrap text-center">
                <StatusBadge status={tx.status} />
              </td>
              <td className="py-3.5 px-4 whitespace-nowrap text-right text-xs">
                <Link
                  href={`/payments/${tx.id}?merchant_id=${merchantId}`}
                  className="inline-flex items-center gap-1 text-indigo-400 hover:text-indigo-300 font-medium group-hover:underline"
                >
                  Auditar →
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
