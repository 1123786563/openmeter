import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { CommerceReceivablePeriod } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useUpdateExternalInvoice } from '@/api/hooks'
import { generateIdempotencyKey } from '@/lib/idempotency'
import { formatShortDateTime } from '@/lib/format'
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

const schema = z.object({
  idempotencyKey: z.string().min(1),
  invoiceNumber: z.string().min(1),
  invoiceUrl: z.string().optional(),
  issuer: z.string().optional(),
  // datetime-local 字符串；提交时转 Date
  issuedAt: z.string().optional(),
})

type FormValues = z.infer<typeof schema>

/**
 * Attach/update the external invoice reference on a receivable period
 * (PUT external-invoice). Idempotency key is generated per dialog open —
 * a retry of the same submit reuses the same key, a new open starts a new
 * update.
 */
export function ExternalInvoiceDialog({
  open,
  onOpenChange,
  customerId,
  period,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  customerId: string
  period: CommerceReceivablePeriod | null
}) {
  const { t } = useTranslation()
  const updateMutation = useUpdateExternalInvoice(customerId)

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      idempotencyKey: '',
      invoiceNumber: '',
      invoiceUrl: '',
      issuer: '',
      issuedAt: '',
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        idempotencyKey: generateIdempotencyKey(),
        invoiceNumber: '',
        invoiceUrl: '',
        issuer: '',
        issuedAt: '',
      })
    }
  }, [open, form])

  const onSubmit = (values: FormValues) => {
    if (!period) return
    updateMutation.mutate(
      {
        periodId: period.id,
        body: {
          idempotencyKey: values.idempotencyKey.trim(),
          invoiceNumber: values.invoiceNumber.trim(),
          invoiceUrl: values.invoiceUrl?.trim() || undefined,
          issuer: values.issuer?.trim() || undefined,
          issuedAt: values.issuedAt ? new Date(values.issuedAt) : undefined,
        },
      },
      {
        onSuccess: () => {
          toast.success(t('customers.receivablePeriods.externalInvoice.toast'))
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
            {t('customers.receivablePeriods.externalInvoice.title')}
          </DialogTitle>
          <DialogDescription>
            {period
              ? t('customers.receivablePeriods.externalInvoice.description', {
                  from: formatShortDateTime(period.periodStart),
                  to: formatShortDateTime(period.periodEnd),
                })
              : ''}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='invoiceNumber'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('customers.receivablePeriods.externalInvoice.number')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder='INV-2026-0001'
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
              name='invoiceUrl'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('customers.receivablePeriods.externalInvoice.url')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder='https://inv.example.com/2026/0001'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('customers.receivablePeriods.externalInvoice.urlHint')}
                  </FormDescription>
                </FormItem>
              )}
            />
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='issuer'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('customers.receivablePeriods.externalInvoice.issuer')}
                    </FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='issuedAt'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('customers.receivablePeriods.externalInvoice.issuedAt')}
                    </FormLabel>
                    <FormControl>
                      <Input type='datetime-local' {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='idempotencyKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t(
                      'customers.receivablePeriods.externalInvoice.idempotencyKey'
                    )}
                  </FormLabel>
                  <FormControl>
                    <Input className='font-mono text-xs' {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'customers.receivablePeriods.externalInvoice.idempotencyHint'
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
              <Button type='submit' disabled={updateMutation.isPending}>
                {updateMutation.isPending
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
