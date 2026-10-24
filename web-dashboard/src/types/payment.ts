export type TransactionStatus =
  | "PENDING"
  | "PROCESSING"
  | "APPROVED"
  | "DECLINED"
  | "FAILED";

export interface TransactionListItem {
  id: string;
  payment_intent_id: string;
  merchant_id: string;
  idempotency_key: string;
  status: TransactionStatus;
  card_bin: string;
  card_last4: string;
  decline_source?: string;
  result_code?: string;
  result_message?: string;
  authorization_code?: string;
  correlation_id: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface EventItem {
  id: string;
  event_type: string;
  payload: Record<string, unknown>;
  created_at: string;
}

export interface TransactionDetail extends TransactionListItem {
  order_reference?: string;
  amount?: number;
  currency?: string;
  events: EventItem[];
}

export interface ListPaymentsResponse {
  items: TransactionListItem[];
  total_count: number;
  limit: number;
  offset: number;
}

export interface ProcessPaymentRequest {
  merchant_id: string;
  order_reference: string;
  amount: number;
  currency: string;
  card_number: string;
  card_expiry_month: number;
  card_expiry_year: number;
  card_cvv: string;
  card_holder_name: string;
}

export interface ProcessPaymentResponse {
  transaction_id: string;
  payment_intent_id: string;
  merchant_id: string;
  order_reference: string;
  amount: number;
  currency: string;
  status: TransactionStatus;
  card_bin: string;
  card_last4: string;
  authorization_code?: string;
  decline_source?: string;
  result_code?: string;
  result_message?: string;
  is_replayed: boolean;
  created_at: string;
  completed_at?: string;
}

export interface ProblemDetails {
  code: string;
  message: string;
  correlation_id?: string;
}
