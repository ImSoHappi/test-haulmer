export interface TestCard {
  id: string;
  name: string;
  pan: string;
  expMonth: number;
  expYear: number;
  cvv: string;
  holder: string;
  description: string;
  expectedStatus: string;
  badgeColor: string;
}

export const TEST_CARDS: TestCard[] = [
  {
    id: "approved",
    name: "Pago Aprobado (201 Created)",
    pan: "4242424242424242",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "Ada Lovelace",
    description: "Flujo exitoso estándar. El banco aprueba inmediatamente.",
    expectedStatus: "APPROVED",
    badgeColor: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
  },
  {
    id: "declined-funds",
    name: "Fondos Insuficientes (402 Payment Required)",
    pan: "4000000000009995",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "Charles Babbage",
    description: "Rechazo determinista con código 51 por saldo insuficiente.",
    expectedStatus: "DECLINED",
    badgeColor: "bg-amber-500/10 text-amber-400 border-amber-500/20",
  },
  {
    id: "persistent-503",
    name: "Caída Bancaria 503 (3 Reintentos)",
    pan: "4000000000000119",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "Alan Turing",
    description: "El adquirente simula caída persistente. La API reintenta 3 veces.",
    expectedStatus: "FAILED",
    badgeColor: "bg-rose-500/10 text-rose-400 border-rose-500/20",
  },
  {
    id: "flaky-503",
    name: "Falla Intermitente (Recupera al 3er intento)",
    pan: "4000000000000341",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "Grace Hopper",
    description: "Falla en los intentos 1 y 2, y se recupera con éxito en el 3ero.",
    expectedStatus: "APPROVED",
    badgeColor: "bg-blue-500/10 text-blue-400 border-blue-500/20",
  },
  {
    id: "timeout-no-auth",
    name: "Timeout sin Autorización (Reconciliación 404)",
    pan: "4000000000000101",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "John von Neumann",
    description: "Timeout de red; API consulta reconciliación, banco responde 404.",
    expectedStatus: "FAILED",
    badgeColor: "bg-rose-500/10 text-rose-400 border-rose-500/20",
  },
  {
    id: "timeout-with-auth",
    name: "Timeout con Autorización (Reconciliación 200)",
    pan: "4000000000000259",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "Margaret Hamilton",
    description: "Timeout de red; API reconcilia, detecta cobro previo y aprueba.",
    expectedStatus: "APPROVED",
    badgeColor: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
  },
  {
    id: "invalid-luhn",
    name: "Anti-Carding: Tarjeta Inválida (Luhn 422)",
    pan: "4242424242424241",
    expMonth: 12,
    expYear: 2030,
    cvv: "123",
    holder: "Carding Bot Tester",
    description: "Dígito verificador Luhn inválido. Falla en memoria sin tocar BD.",
    expectedStatus: "422 Error",
    badgeColor: "bg-purple-500/10 text-purple-400 border-purple-500/20",
  },
];
