"use server";

import { submitPayment, ProcessPaymentResult } from "@/lib/api";
import { ProcessPaymentRequest } from "@/types/payment";

export async function processPaymentAction(
  req: ProcessPaymentRequest,
  idempotencyKey: string
): Promise<ProcessPaymentResult> {
  return submitPayment(req, idempotencyKey);
}
