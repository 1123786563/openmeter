import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { CommerceReceivablePeriod } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateOfflinePayment } from '@/api/hooks'
import { formatFen, formatShortDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { generateIdempotencyKey } from '@/lib/idempotency'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

/**
 * Amounts are entered in the major currency unit (yuan) and converted to fen
 * on submit (same precedent as the recharge product form).
 */
const POSITIVE_AMOUNT = /^\d+(\.\d{1,2})?$/

/**
 * Radix Select forbids empty-string item values, so the "no period" option
 * uses a sentinel; the form value stays '' (= key omitted from the wire body)
 * and submit maps it to undefined.
 */
const NO_PERIOD = '__none__'

/**
 * Register an offline payment for a customer (POST offline-payments).
 * Idempotency key is generated per dialog open — a retry of the same submit
 * reuses the same key, a new open starts a new payment.
 */
export function OfflinePaymentDialog({
  open,
  onOpenChange,
  customerId,
  periods,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  customerId: string
  periods: CommerceReceivablePeriod[]
}) {
  const { t } = useTranslation()
  const createMutation = useCreateOfflinePayment(customerId)

  // Schema lives inside the component so validation messages resolve through
  // i18n (FormMessage renders error.message verbatim).
  const schema = useMemo(
    () =>
      z.object({
        idempotencyKey: z.string().min(1),
        amountYuan: z
          .string()
          .regex(
            POSITIVE_AMOUNT,
            t('customers.receivablePeriods.offlinePayment.validation.amount')
          ),
        currency: z
          .string()
          .min(
            1,
            t('customers.receivablePeriods.offlinePayment.validation.currency')
          ),
        externalReference: z
          .string()
          .min(
            1,
            t(
              'customers.receivablePeriods.offlinePayment.validation.externalReference'
            )
          ),
        // datetime-local 字符串；提交时转 Date
        receivedAt: z
          .string()
          .min(
            1,
            t(
              'customers.receivablePeriods.offlinePayment.validation.receivedAt'
            )
          ),
        receivablePeriodId: z.string(),
        note: z.string(),
      }),
    [t]
  )

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      idempotencyKey: '',
      amountYuan: '',
      currency: '',
      externalReference: '',
      receivedAt: '',
      receivablePeriodId: '',
      note: '',
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        idempotencyKey: generateIdempotencyKey(),
        amountYuan: '',
        currency: '',
        externalReference: '',
        receivedAt: '',
        receivablePeriodId: '',
        note: '',
      })
    }
  }, [open, form])

  const onSubmit = (values: FormValues) => {
    createMutation.mutate(
      {
        idempotencyKey: values.idempotencyKey.trim(),
        amountFen: BigInt(Math.round(Number(values.amountYuan) * 100)),
        currency: values.currency.trim(),
        receivablePeriodId: values.receivablePeriodId || undefined,
        externalReference: values.externalReference.trim(),
        receivedAt: new Date(values.receivedAt),
        note: values.note?.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(
            t('customers.receivablePeriods.offlinePayment.toast.created')
          )
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {t('customers.receivablePeriods.offlinePayment.title')}
          </DialogTitle>
          <DialogDescription>
            {t('customers.receivablePeriods.offlinePayment.description')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='receivablePeriodId'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('customers.receivablePeriods.offlinePayment.period')}
                  </FormLabel>
                  <Select
                    value={field.value || NO_PERIOD}
                    onValueChange={(value) => {
                      field.onChange(value === NO_PERIOD ? '' : value)
                      // Selecting a period prefills the payment currency.
                      const period = periods.find(
                        (period) => period.id === value
                      )
                      if (period) {
                        form.setValue('currency', period.currency, {
                          shouldValidate: true,
                        })
                      }
                    }}
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value={NO_PERIOD}>
                        {t(
                          'customers.receivablePeriods.offlinePayment.periodPlaceholder'
                        )}
                      </SelectItem>
                      {periods.map((period) => (
                        <SelectItem key={period.id} value={period.id}>
                          {`${formatShortDateTime(period.periodStart)} ~ ${formatShortDateTime(period.periodEnd)} · ${formatFen(period.amountDueFen, period.currency)}`}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormItem>
              )}
            />
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='amountYuan'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('customers.receivablePeriods.offlinePayment.amount')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        inputMode='decimal'
                        placeholder='199.00'
                        autoComplete='off'
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'customers.receivablePeriods.offlinePayment.amountHint'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='currency'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('customers.receivablePeriods.offlinePayment.currency')}
                    </FormLabel>
                    <FormControl>
                      <Input placeholder='CNY' autoComplete='off' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='externalReference'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t(
                        'customers.receivablePeriods.offlinePayment.externalReference'
                      )}
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder='20260829-12345678'
                        autoComplete='off'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='receivedAt'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t(
                        'customers.receivablePeriods.offlinePayment.receivedAt'
                      )}
                    </FormLabel>
                    <FormControl>
                      <Input type='datetime-local' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='note'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('customers.receivablePeriods.offlinePayment.noteLabel')}
                  </FormLabel>
                  <FormControl>
                    <Textarea rows={2} {...field} />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='idempotencyKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t(
                      'customers.receivablePeriods.offlinePayment.idempotencyKey'
                    )}
                  </FormLabel>
                  <FormControl>
                    <Input className='font-mono text-xs' {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'customers.receivablePeriods.offlinePayment.idempotencyHint'
                    )}
                  </FormDescription>
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => onOpenChange(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type='submit' disabled={createMutation.isPending}>
                {createMutation.isPending
                  ? t('common.submitting')
                  : t('common.confirm')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
