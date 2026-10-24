"use client";

import { useState } from "react";

interface CopyButtonProps {
  text: string;
  label?: string;
  className?: string;
}

export function CopyButton({ text, label, className = "" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Fallback
      setCopied(false);
    }
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={label || "Copiar al portapapeles"}
      title={label || "Copiar al portapapeles"}
      className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-mono transition-all ${
        copied
          ? "bg-emerald-950/60 text-emerald-300 border border-emerald-600/50"
          : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/80 border border-transparent"
      } ${className}`}
    >
      {copied ? (
        <>
          <svg className="w-3 h-3 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          <span className="text-[10px] font-sans font-medium text-emerald-400">¡Copiado!</span>
        </>
      ) : (
        <svg className="w-3 h-3 opacity-70 hover:opacity-100" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
          />
        </svg>
      )}
    </button>
  );
}
