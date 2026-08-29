import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useCreateCustomCurrency,
  useCurrencies,
  useFiatCurrencies,
} from '@/api/hooks'
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

const NON_NEGATIVE_INT = /^\d+$/

type CustomCurrencyFormValues = {
  name: string
  code: string
  precision: string
  decimalMark: string
  thousandSeparator: string
  symbol: string
}

const EMPTY_VALUES: CustomCurrencyFormValues = {
  name: '',
  code: '',
  precision: '2',
  decimalMark: '.',
  thousandSeparator: ',',
  symbol: '',
}

/**
 * The schema is rebuilt whenever the fetched lists or active locale change:
 * the code field is validated against live data (the custom code, 4-24
 * chars, must not equal an existing fiat code from the v1 list —
 * belt-and-braces since fiat codes are 3 chars — nor an already-created
 * custom code, case-insensitive), and shadcn's FormMessage renders
 * error.message, so the validation copy must live on the zod checks
 * themselves.
 */
function buildSchema(conflictingCodes: string[], t: (key: string) => string) {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, t('config.currencies.custom.form.validation.required'))
      .max(256, t('config.currencies.custom.form.validation.required')),
    code: z
      .string()
      .trim()
      .min(4, t('config.currencies.custom.form.validation.codeLength'))
      .max(24, t('config.currencies.custom.form.validation.codeLength'))
      .refine(
        (code) => !conflictingCodes.includes(code.toUpperCase()),
        t('config.currencies.custom.form.validation.codeConflict')
      ),
    precision: z
      .string()
      .refine(
        (value) => NON_NEGATIVE_INT.test(value) && Number(value) <= 12,
        t('config.currencies.custom.form.validation.precision')
      ),
    decimalMark: z
      .string()
      .trim()
      .min(1, t('config.currencies.custom.form.validation.singleChar'))
      .max(1, t('config.currencies.custom.form.validation.singleChar')),
    thousandSeparator: z
      .string()
      .trim()
      .min(1, t('config.currencies.custom.form.validation.singleChar'))
      .max(1, t('config.currencies.custom.form.validation.singleChar')),
    symbol: z.string().trim().max(16),
  })
}

type CustomCurrencyDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * Create-only dialog: the currencies API exposes no update/delete endpoint
 * for custom currencies, so creation is permanent.
 */
export function CustomCurrencyDialog({
  open,
  onOpenChange,
}: CustomCurrencyDialogProps) {
  const { t } = useTranslation()
  const createMutation = useCreateCustomCurrency()

  const { data: fiat } = useFiatCurrencies()
  const { data: custom } = useCurrencies({ type: 'custom' })

  const conflictingCodes = useMemo(() => {
    const codes = new Set<string>()
    for (const currency of fiat ?? []) codes.add(currency.code.toUpperCase())
    for (const currency of custom?.data ?? [])
      codes.add(currency.code.toUpperCase())
    return [...codes]
  }, [fiat, custom])

  const schema = useMemo(
    () => buildSchema(conflictingCodes, t),
    [conflictingCodes, t]
  )

  const form = useForm<CustomCurrencyFormValues>({
    resolver: zodResolver(schema),
    defaultValues: EMPTY_VALUES,
  })

  useEffect(() => {
    if (open) form.reset(EMPTY_VALUES)
  }, [open, form])

  const onSubmit = (values: CustomCurrencyFormValues) => {
    createMutation.mutate(
      {
        name: values.name.trim(),
        code: values.code.trim(),
        precision: Number(values.precision),
        decimalMark: values.decimalMark.trim(),
        thousandSeparator: values.thousandSeparator.trim(),
        symbol: values.symbol.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t('config.currencies.custom.toast.created'))
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
            {t('config.currencies.custom.form.createTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('config.currencies.custom.form.createDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.currencies.custom.name')}</FormLabel>
                  <FormControl>
                    <Input placeholder='Credit Points' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='code'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.currencies.custom.code')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder='CREDIT_POINTS'
                      autoComplete='off'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('config.currencies.custom.form.codeHint')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='grid grid-cols-3 gap-4'>
              <FormField
                control={form.control}
                name='precision'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('config.currencies.custom.precision')}</FormLabel>
                    <FormControl>
                      <Input inputMode='numeric' placeholder='2' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='decimalMark'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.currencies.custom.decimalMark')}
                    </FormLabel>
                    <FormControl>
                      <Input maxLength={1} placeholder='.' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='thousandSeparator'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.currencies.custom.thousandSeparator')}
                    </FormLabel>
                    <FormControl>
                      <Input maxLength={1} placeholder=',' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='symbol'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.currencies.custom.symbol')}（{t('common.optional')}）
                  </FormLabel>
                  <FormControl>
                    <Input placeholder='⭐' {...field} />
                  </FormControl>
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
