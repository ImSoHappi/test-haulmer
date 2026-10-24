import {
  ListPaymentsResponse,
  ProcessPaymentRequest,
  ProcessPaymentResponse,
  TransactionDetail,
} from "@/types/payment";

function getBaseUrl(): string {
  const url = process.env.PAYMENTS_API_URL;
  if (!url) {
    throw new Error("PAYMENTS_API_URL environment variable is required");
  }
  return url;
}

export async function fetchPayments(
  merchantId: string,
  status?: string,
  limit = 20,
  offset = 0
): Promise<ListPaymentsResponse> {
  const url = new URL(`${getBaseUrl()}/v1/payments`);
  if (status && status !== "ALL") {
    url.searchParams.set("status", status);
  }
  url.searchParams.set("limit", limit.toString());
  url.searchParams.set("offset", offset.toString());

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: {
      "X-Merchant-ID": merchantId,
      Accept: "application/json",
    },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorBody = await res.text();
    throw new Error(`Error ${res.status}: ${errorBody || res.statusText}`);
  }

  return res.json();
}

export async function fetchPaymentById(
  id: string,
  merchantId: string
): Promise<TransactionDetail> {
  const res = await fetch(`${getBaseUrl()}/v1/payments/${id}`, {
    method: "GET",
    headers: {
      "X-Merchant-ID": merchantId,
      Accept: "application/json",
    },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorBody = await res.text();
    throw new Error(`Error ${res.status}: ${errorBody || res.statusText}`);
  }

  return res.json();
}

export interface ProcessPaymentResult {
  ok: boolean;
  status: number;
  data?: ProcessPaymentResponse;
  error?: {
    code: string;
    message: string;
    correlation_id?: string;
  };
}

export async function submitPayment(
  req: ProcessPaymentRequest,
  idempotencyKey: string
): Promise<ProcessPaymentResult> {
  try {
    const res = await fetch(`${getBaseUrl()}/v1/payments`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": idempotencyKey,
      },
      body: JSON.stringify(req),
      cache: "no-store",
    });

    const isJson = res.headers.get("content-type")?.includes("json");
    const body = isJson ? await res.json() : await res.text();

    if (!res.ok) {
      return {
        ok: false,
        status: res.status,
        error: isJson
          ? body
          : {
              code: "http_error",
              message: String(body || res.statusText),
            },
      };
    }

    return {
      ok: true,
      status: res.status,
      data: body,
    };
  } catch (err) {
    return {
      ok: false,
      status: 0,
      error: {
        code: "network_error",
        message: err instanceof Error ? err.message : "Error de red al conectar con payments-api",
      },
    };
  }
}
