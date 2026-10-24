"use client";

import { useState, Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { TEST_CARDS, TestCard } from "@/lib/test-cards";
import { ProcessPaymentResult } from "@/lib/api";
import { processPaymentAction } from "./actions";
import { StatusBadge } from "@/components/StatusBadge";
import { formatCLP } from "@/lib/formatters";

function generateOrderRef(): string {
  const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
  let result = "";
  for (let i = 0; i < 6; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

function generateIdempotencyKey(): string {
  return "idem-" + Date.now() + "-" + Math.random().toString(36).substring(2, 7);
}

export default function SandboxPage() {
  return (
    <Suspense fallback={<div className="p-8 text-center text-slate-500">Cargando simulador...</div>}>
      <SandboxContent />
    </Suspense>
  );
}

function SandboxContent() {
  const searchParams = useSearchParams();
  const merchantId = searchParams.get("merchant_id") || "1001";

  // Form State
  const [selectedCardId, setSelectedCardId] = useState<string>("approved");
  const [orderReference, setOrderReference] = useState<string>(generateOrderRef());
  const [amount, setAmount] = useState<number>(25990);
  const [idempotencyKey, setIdempotencyKey] = useState<string>(generateIdempotencyKey());
  const [cardNumber, setCardNumber] = useState<string>(TEST_CARDS[0].pan);
  const [cardExpiryMonth, setCardExpiryMonth] = useState<number>(TEST_CARDS[0].expMonth);
  const [cardExpiryYear, setCardExpiryYear] = useState<number>(TEST_CARDS[0].expYear);
  const [cardCvv, setCardCvv] = useState<string>(TEST_CARDS[0].cvv);
  const [cardHolderName, setCardHolderName] = useState<string>(TEST_CARDS[0].holder);

  // Submission State
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [result, setResult] = useState<ProcessPaymentResult | null>(null);

  const handleSelectCard = (card: TestCard) => {
    setSelectedCardId(card.id);
    setCardNumber(card.pan);
    setCardExpiryMonth(card.expMonth);
    setCardExpiryYear(card.expYear);
    setCardCvv(card.cvv);
    setCardHolderName(card.holder);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (isSubmitting) return;

    setIsSubmitting(true);
    setResult(null);

    const res = await processPaymentAction(
      {
        merchant_id: merchantId,
        order_reference: orderReference,
        amount: Number(amount),
        currency: "CLP",
        card_number: cardNumber.replace(/\s+/g, ""),
        card_expiry_month: Number(cardExpiryMonth),
        card_expiry_year: Number(cardExpiryYear),
        card_cvv: cardCvv,
        card_holder_name: cardHolderName,
      },
      idempotencyKey
    );

    setResult(res);
    setIsSubmitting(false);
  };

  return (
    <div className="space-y-8 max-w-5xl mx-auto">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-white">
          Simulador de Checkout (Sandbox)
        </h1>
        <p className="text-sm text-slate-400 mt-1">
          Dispara cobros en vivo contra la API de pagos para el comercio{" "}
          <span className="font-mono text-indigo-400 font-semibold">{merchantId}</span>.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Left Column: Preset Cards Selector */}
        <div className="lg:col-span-5 space-y-3">
          <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
            1. Elige un escenario de prueba
          </h2>
          <div className="space-y-2">
            {TEST_CARDS.map((c) => {
              const isSelected = selectedCardId === c.id;
              return (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => handleSelectCard(c)}
                  className={`w-full text-left p-3.5 rounded-xl border transition-all text-xs ${
                    isSelected
                      ? "bg-slate-800/90 border-indigo-500 shadow-md shadow-indigo-500/10 ring-1 ring-indigo-500/50"
                      : "bg-slate-900/40 border-slate-800 hover:border-slate-700 hover:bg-slate-900/80"
                  }`}
                >
                  <div className="flex items-center justify-between gap-2 mb-1">
                    <span className="font-semibold text-slate-200">{c.name}</span>
                    <span className={`text-[10px] px-1.5 py-0.5 rounded border ${c.badgeColor}`}>
                      {c.expectedStatus}
                    </span>
                  </div>
                  <p className="text-[11px] text-slate-400 leading-relaxed mb-2">
                    {c.description}
                  </p>
                  <span className="font-mono text-slate-500 text-[10px]">
                    PAN: {c.pan.slice(0, 4)}••••{c.pan.slice(-4)}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Right Column: Checkout Form & Result */}
        <div className="lg:col-span-7 space-y-6">
          <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6 shadow-sm">
            <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-4">
              2. Datos del Cobro
            </h2>

            <form onSubmit={handleSubmit} className="space-y-4 text-xs">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-slate-400 mb-1">Monto en CLP</label>
                  <input
                    type="number"
                    value={amount}
                    onChange={(e) => setAmount(Number(e.target.value))}
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white font-mono focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                    required
                  />
                  <span className="text-[10px] text-slate-500 mt-1 block">
                    {formatCLP(amount || 0)}
                  </span>
                </div>

                <div>
                  <label className="block text-slate-400 mb-1">Referencia Orden (6 chars)</label>
                  <div className="flex gap-1.5">
                    <input
                      type="text"
                      maxLength={6}
                      value={orderReference}
                      onChange={(e) => setOrderReference(e.target.value.toUpperCase())}
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white font-mono uppercase focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                      required
                    />
                    <button
                      type="button"
                      onClick={() => setOrderReference(generateOrderRef())}
                      title="Generar nueva orden"
                      className="px-2.5 bg-slate-800 hover:bg-slate-700 rounded-lg text-slate-300"
                    >
                      🎲
                    </button>
                  </div>
                </div>
              </div>

              <div>
                <label className="block text-slate-400 mb-1">Clave de Idempotencia</label>
                <div className="flex gap-1.5">
                  <input
                    type="text"
                    value={idempotencyKey}
                    onChange={(e) => setIdempotencyKey(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-slate-300 font-mono text-[11px] focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setIdempotencyKey(generateIdempotencyKey())}
                    title="Generar nueva clave"
                    className="px-2.5 bg-slate-800 hover:bg-slate-700 rounded-lg text-slate-300"
                  >
                    🔄
                  </button>
                </div>
                <span className="text-[10px] text-slate-500 mt-0.5 block">
                  Mantén la misma clave para probar el replay idempotente.
                </span>
              </div>

              <div className="border-t border-slate-800/80 pt-4 space-y-3">
                <div>
                  <label className="block text-slate-400 mb-1">Número de Tarjeta (PAN)</label>
                  <input
                    type="text"
                    value={cardNumber}
                    onChange={(e) => setCardNumber(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white font-mono tracking-wider focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                    required
                  />
                </div>

                <div className="grid grid-cols-3 gap-3">
                  <div>
                    <label className="block text-slate-400 mb-1">Mes Exp</label>
                    <input
                      type="number"
                      min={1}
                      max={12}
                      value={cardExpiryMonth}
                      onChange={(e) => setCardExpiryMonth(Number(e.target.value))}
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white font-mono focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-slate-400 mb-1">Año Exp</label>
                    <input
                      type="number"
                      min={2024}
                      max={2040}
                      value={cardExpiryYear}
                      onChange={(e) => setCardExpiryYear(Number(e.target.value))}
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white font-mono focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-slate-400 mb-1">CVV</label>
                    <input
                      type="text"
                      maxLength={4}
                      value={cardCvv}
                      onChange={(e) => setCardCvv(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white font-mono focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                      required
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-slate-400 mb-1">Nombre del Titular</label>
                  <input
                    type="text"
                    value={cardHolderName}
                    onChange={(e) => setCardHolderName(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-white focus:ring-1 focus:ring-indigo-500 focus:outline-none"
                    required
                  />
                </div>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="w-full py-3 px-4 bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-900 disabled:text-indigo-400 disabled:cursor-not-allowed text-white font-semibold rounded-lg shadow-lg shadow-indigo-600/20 text-sm transition-all flex items-center justify-center gap-2"
                >
                  {isSubmitting ? (
                    <>
                      <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                      <span>Procesando pago con la API...</span>
                    </>
                  ) : (
                    <span>Procesar Pago ({formatCLP(amount || 0)})</span>
                  )}
                </button>
              </div>
            </form>
          </div>

          {/* Result Card */}
          {result && (
            <div
              className={`p-5 rounded-xl border shadow-sm transition-all ${
                result.ok
                  ? result.data?.status === "APPROVED"
                    ? "bg-emerald-950/20 border-emerald-500/40 text-emerald-300"
                    : "bg-amber-950/20 border-amber-500/40 text-amber-300"
                  : "bg-rose-950/20 border-rose-500/40 text-rose-300"
              }`}
            >
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-mono font-bold px-2 py-0.5 rounded bg-black/40 border border-white/10">
                    HTTP {result.status}
                  </span>
                  {result.data && <StatusBadge status={result.data.status} />}
                  {result.data?.is_replayed && (
                    <span className="text-[10px] bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 px-2 py-0.5 rounded font-mono">
                      Idempotent-Replayed: true
                    </span>
                  )}
                </div>
              </div>

              {result.ok && result.data && (
                <div className="space-y-2 text-xs">
                  <p>
                    <span className="text-slate-400">ID Transacción:</span>{" "}
                    <span className="font-mono text-white select-all">
                      {result.data.transaction_id}
                    </span>
                  </p>
                  {result.data.authorization_code && (
                    <p>
                      <span className="text-slate-400">Código Autorización:</span>{" "}
                      <span className="font-mono font-semibold text-emerald-400">
                        {result.data.authorization_code}
                      </span>
                    </p>
                  )}
                  {result.data.result_message && (
                    <p>
                      <span className="text-slate-400">Mensaje:</span>{" "}
                      <span className="text-slate-200">{result.data.result_message}</span>
                    </p>
                  )}

                  <div className="pt-3">
                    <Link
                      href={`/payments/${result.data.transaction_id}?merchant_id=${merchantId}`}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg font-medium text-xs shadow-sm transition-colors"
                    >
                      <span>Auditar Timeline de esta Transacción →</span>
                    </Link>
                  </div>
                </div>
              )}

              {!result.ok && result.error && (
                <div className="space-y-1 text-xs">
                  <p className="font-semibold text-rose-200">
                    Error {result.error.code}: {result.error.message}
                  </p>
                  {result.error.correlation_id && (
                    <p className="text-[11px] font-mono text-rose-400/80">
                      Correlation ID: {result.error.correlation_id}
                    </p>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
