import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { CostBasis, CurrencyCustom } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateCostBasis, useFiatCurrencies } from '@/api/hooks'
import { formatShortDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
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

const NON_NEGATIVE_DECIMAL = /^\d+(\.\d+)?$/

type CostBasisFormValues = {
  fiatCode: string
  rate: string
  effectiveFrom: string
}

const EMPTY_VALUES: CostBasisFormValues = {
  fiatCode: '',
  rate: '',
  effectiveFrom: '',
}

function buildSchema(t: (key: string) => string) {
  return z.object({
    fiatCode: z
      .string()
      .min(1, t('config.currencies.costBasis.validation.required')),
    rate: z
      .string()
      .trim()
      .min(1, t('config.currencies.costBasis.validation.required'))
      .refine(
        (value) => NON_NEGATIVE_DECIMAL.test(value),
        t('config.currencies.costBasis.validation.rate')
      ),
    effectiveFrom: z.string(),
  })
}

/**
 * Cost bases of one custom currency: read-only timeline (append-only API —
 * a new row supersedes the previous one for its period) plus an append form.
 * The dialog stays open after a successful append so the refreshed list and
 * the next entry are visible in one place.
 */
export function CostBasisDialog({
  open,
  onOpenChange,
  currency,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  currency: CurrencyCustom | null
}) {
  const { t } = useTranslation()
  const createMutation = useCreateCostBasis()
  const { data: fiat } = useFiatCurrencies()

  const schema = buildSchema(t)
  const form = useForm<CostBasisFormValues>({
    resolver: zodResolver(schema),
    defaultValues: EMPTY_VALUES,
  })

  useEffect(() => {
    if (open) form.reset(EMPTY_VALUES)
  }, [open, form])

  if (!currency) return null

  const costBases = [...(currency.costBasis ?? [])].sort(
    (a, b) =>
      (a.effectiveFrom ?? a.createdAt).getTime() -
      (b.effectiveFrom ?? b.createdAt).getTime()
  )

  const onSubmit = (values: CostBasisFormValues) => {
    createMutation.mutate(
      {
        currencyId: currency.id,
        body: {
          fiatCode: values.fiatCode,
          rate: values.rate.trim(),
          ...(values.effectiveFrom
            ? { effectiveFrom: new Date(values.effectiveFrom) }
            : {}),
        },
      },
      {
        onSuccess: () => {
          toast.success(t('config.currencies.costBasis.toast.added'))
          form.reset(EMPTY_VALUES)
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
            {t('config.currencies.costBasis.title', { code: currency.code })}
          </DialogTitle>
          <DialogDescription>
            {t('config.currencies.costBasis.description')}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-2'>
          {costBases.length === 0 && (
            <p className='text-sm text-muted-foreground'>
              {t('config.currencies.costBasis.empty')}
            </p>
          )}
          {costBases.map((costBasis: CostBasis) => (
            <div
              key={costBasis.id}
              className='flex flex-wrap items-center gap-x-3 gap-y-1 rounded-md border px-3 py-2 text-sm'
            >
              <Badge variant='outline' className='font-mono'>
                {costBasis.fiatCode}
              </Badge>
              <span className='font-medium tabular-nums'>{costBasis.rate}</span>
              <span className='text-muted-foreground'>
                {t('config.currencies.costBasis.period', {
                  from: formatShortDateTime(
                    costBasis.effectiveFrom ?? costBasis.createdAt
                  ),
                  to: costBasis.effectiveTo
                    ? formatShortDateTime(costBasis.effectiveTo)
                    : t('config.currencies.costBasis.openEnded'),
                })}
              </span>
            </div>
          ))}
        </div>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='fiatCode'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.currencies.costBasis.form.fiatCode')}
                    </FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {(fiat ?? []).map((fiatCurrency) => (
                          <SelectItem
                            key={fiatCurrency.code}
                            value={fiatCurrency.code}
                          >
                            {fiatCurrency.code}（{fiatCurrency.name}）
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='rate'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.currencies.costBasis.form.rate')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        inputMode='decimal'
                        placeholder='0.015'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='effectiveFrom'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.currencies.costBasis.form.effectiveFrom')}（
                    {t('common.optional')}）
                  </FormLabel>
                  <FormControl>
                    <Input type='date' {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('config.currencies.costBasis.form.effectiveFromHint')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='flex justify-end'>
              <Button type='submit' disabled={createMutation.isPending}>
                {createMutation.isPending
                  ? t('common.submitting')
                  : t('config.currencies.costBasis.form.add')}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
