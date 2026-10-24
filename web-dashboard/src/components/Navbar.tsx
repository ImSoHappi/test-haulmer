"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

export function Navbar() {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const merchantId = searchParams.get("merchant_id") || "1001";

  const handleMerchantChange = (newMerchant: string) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set("merchant_id", newMerchant);
    router.push(`${pathname}?${params.toString()}`);
  };

  return (
    <header className="border-b border-slate-800 bg-slate-950/80 backdrop-blur sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-2 text-white font-bold tracking-tight">
            <span className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center text-white font-black text-sm shadow-indigo-500/20 shadow-lg">
              H
            </span>
            <span className="text-lg">Haulmer</span>
            <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 rounded">
              Dashboard
            </span>
          </Link>

          <nav className="hidden sm:flex items-center gap-1">
            <Link
              href={`/?merchant_id=${merchantId}`}
              className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${pathname === "/"
                  ? "bg-slate-800 text-white"
                  : "text-slate-400 hover:text-slate-200 hover:bg-slate-900"
                }`}
            >
              Transacciones
            </Link>
            <Link
              href={`/sandbox?merchant_id=${merchantId}`}
              className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${pathname === "/sandbox"
                  ? "bg-slate-800 text-white"
                  : "text-slate-400 hover:text-slate-200 hover:bg-slate-900"
                }`}
            >
              Simulador Sandbox
            </Link>
          </nav>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-xs">
            <span className="text-slate-400 hidden sm:inline">Comercio:</span>
            <select
              value={merchantId}
              onChange={(e) => handleMerchantChange(e.target.value)}
              className="bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded-lg px-2.5 py-1.5 focus:ring-1 focus:ring-indigo-500 focus:outline-none"
            >
              <option value="1001">1001 (Activo - Store Principal)</option>
              <option value="1002">1002 (Inactivo - Anti-Carding Gate)</option>
            </select>
          </div>
        </div>
      </div>
    </header>
  );
}
