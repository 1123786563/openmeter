ALTER TABLE "refund_requests" ADD COLUMN "credit_quantum" bigint NOT NULL DEFAULT 10;
ALTER TABLE "refund_requests" ADD COLUMN "refund_quantum_fen" bigint NOT NULL DEFAULT 1;
ALTER TABLE "refund_requests" ADD COLUMN "reserved_credits" bigint NOT NULL DEFAULT 0;
ALTER TABLE "refund_requests" ADD COLUMN "refund_fen" bigint NOT NULL DEFAULT 0;
ALTER TABLE "refund_requests" ADD COLUMN "remainder_credits" bigint NOT NULL DEFAULT 0;
ALTER TABLE "refund_requests" ADD COLUMN "provider_name" varchar NOT NULL DEFAULT '';
ALTER TABLE "refund_requests" ADD COLUMN "provider_refund_id" varchar NOT NULL DEFAULT '';
ALTER TABLE "refund_requests" ADD COLUMN "fence_sequence" varchar NOT NULL DEFAULT '';
ALTER TABLE "refund_requests" ADD COLUMN "failure_reason" varchar NULL;
