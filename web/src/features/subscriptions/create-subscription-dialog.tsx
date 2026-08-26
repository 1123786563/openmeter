import { useState } from 'react'
import type { Customer, Plan } from '@openmeter/client'
import { ArrowLeft, ArrowRight, CheckCircle2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateSubscription, usePlans } from '@/api/hooks'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableRow } from '@/components/ui/table'
import { CustomerPicker } from '@/features/customers/customer-picker'

const STEPS = ['customer', 'plan', 'confirm'] as const
type Step = (typeof STEPS)[number]

export function CreateSubscriptionDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>('customer')
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [plan, setPlan] = useState<Plan | null>(null)
  const [settlementMode, setSettlementMode] = useState<
    'credit_then_invoice' | 'credit_only'
  >('credit_then_invoice')

  const { data: plans, isLoading: plansLoading } = usePlans()
  const createMutation = useCreateSubscription()

  const reset = () => {
    setStep('customer')
    setCustomer(null)
    setPlan(null)
    setSettlementMode('credit_then_invoice')
  }

  const close = (next: boolean) => {
    if (!next) reset()
    onOpenChange(next)
  }

  const submit = () => {
    if (!customer || !plan) return
    createMutation.mutate(
      {
        customer: { id: customer.id },
        plan: { id: plan.id },
        settlementMode,
      },
      {
        onSuccess: () => {
          toast.success(t('subscriptions.toast.created'))
          reset()
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  const stepIndex = STEPS.indexOf(step)
  const canNext =
    (step === 'customer' && Boolean(customer)) ||
    (step === 'plan' && Boolean(plan))

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('subscriptions.create.title')}</DialogTitle>
          <DialogDescription>
            {t(`subscriptions.create.steps.${step}`)} ({stepIndex + 1}/
            {STEPS.length})
          </DialogDescription>
        </DialogHeader>

        {step === 'customer' && (
          <div className='space-y-2'>
            <Label>{t('subscriptions.create.customer')}</Label>
            <CustomerPicker value={customer} onChange={setCustomer} />
          </div>
        )}

        {step === 'plan' && (
          <div className='space-y-2'>
            <Label>{t('subscriptions.create.plan')}</Label>
            {plansLoading ? (
              <Skeleton className='h-40 w-full' />
            ) : (
              <div className='max-h-64 space-y-2 overflow-y-auto pe-1'>
                {(plans ?? []).map((candidate) => (
                  <button
                    key={candidate.id}
                    type='button'
                    onClick={() => setPlan(candidate)}
                    className={`flex w-full items-center justify-between rounded-md border px-3 py-2 text-start text-sm transition-colors ${
                      plan?.id === candidate.id
                        ? 'border-primary bg-primary/5'
                        : 'hover:bg-accent'
                    }`}
                  >
                    <span className='min-w-0'>
                      <span className='block truncate font-medium'>
                        {candidate.name}
                      </span>
                      <span className='block text-xs text-muted-foreground'>
                        {candidate.key} · v{candidate.version} ·{' '}
                        {candidate.currency} · {candidate.billingCadence}
                      </span>
                    </span>
                    {candidate.status === 'active' && (
                      <CheckCircle2 className='ms-2 size-4 shrink-0 text-emerald-600' />
                    )}
                  </button>
                ))}
                {(plans ?? []).length === 0 && (
                  <p className='py-6 text-center text-sm text-muted-foreground'>
                    {t('subscriptions.create.noPlans')}
                  </p>
                )}
              </div>
            )}
          </div>
        )}

        {step === 'confirm' && customer && plan && (
          <div className='space-y-4'>
            <Table>
              <TableBody>
                <TableRow>
                  <TableCell className='text-muted-foreground'>
                    {t('subscriptions.create.customer')}
                  </TableCell>
                  <TableCell className='font-medium'>
                    {customer.name} ({customer.key})
                  </TableCell>
                </TableRow>
                <TableRow>
                  <TableCell className='text-muted-foreground'>
                    {t('subscriptions.create.plan')}
                  </TableCell>
                  <TableCell className='font-medium'>
                    {plan.name} · v{plan.version}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
            <div className='space-y-2'>
              <Label>{t('subscriptions.fields.settlementMode')}</Label>
              <RadioGroup
                value={settlementMode}
                onValueChange={(value) =>
                  setSettlementMode(
                    value as 'credit_then_invoice' | 'credit_only'
                  )
                }
              >
                <div className='flex items-center gap-2'>
                  <RadioGroupItem
                    value='credit_then_invoice'
                    id='sm-credit-then-invoice'
                  />
                  <Label
                    htmlFor='sm-credit-then-invoice'
                    className='font-normal'
                  >
                    {t('subscriptions.settlementMode.credit_then_invoice')}
                  </Label>
                </div>
                <div className='flex items-center gap-2'>
                  <RadioGroupItem value='credit_only' id='sm-credit-only' />
                  <Label htmlFor='sm-credit-only' className='font-normal'>
                    {t('subscriptions.settlementMode.credit_only')}
                  </Label>
                </div>
              </RadioGroup>
            </div>
          </div>
        )}

        <DialogFooter className='gap-2 sm:gap-0'>
          {stepIndex > 0 && (
            <Button
              variant='outline'
              onClick={() => setStep(STEPS[stepIndex - 1])}
            >
              <ArrowLeft className='size-4' />
              {t('common.back')}
            </Button>
          )}
          {stepIndex < STEPS.length - 1 ? (
            <Button
              disabled={!canNext}
              onClick={() => setStep(STEPS[stepIndex + 1])}
            >
              {t('common.next')}
              <ArrowRight className='size-4' />
            </Button>
          ) : (
            <Button onClick={submit} disabled={createMutation.isPending}>
              {createMutation.isPending
                ? t('common.submitting')
                : t('subscriptions.create.submit')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
