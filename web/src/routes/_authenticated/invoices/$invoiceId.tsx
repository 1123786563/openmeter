import { createFileRoute } from '@tanstack/react-router'
import { InvoiceDetail } from '@/features/invoices/invoice-detail'

export const Route = createFileRoute('/_authenticated/invoices/$invoiceId')({
  component: InvoiceDetail,
})
