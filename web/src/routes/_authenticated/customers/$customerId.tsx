import { createFileRoute } from '@tanstack/react-router'
import { CustomerDetail } from '@/features/customers/customer-detail'

export const Route = createFileRoute('/_authenticated/customers/$customerId')({
  component: CustomerDetail,
})
