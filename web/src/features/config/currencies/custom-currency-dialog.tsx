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
 * The schema is rebuilt whenever the fetched lists change so the code field
 * is validated against live data: the custom code (4-24 chars) must not equal
 * an existing fiat code (v1 list; belt-and-braces since fiat codes are 3
 * chars) nor an already-created custom code (v3 list, case-insensitive).
 */
function buildSchema(conflictingCodes: string[]) {
  return z.object({
    name: z.string().trim().min(1).max(256),
    code: z
      .string()
      .trim()
      .min(4)
      .max(24)
      .refine(
        (code) => !conflictingCodes.includes(code.toUpperCase()),
        'conflict'
      ),
    precision: z
      .string()
      .refine((value) => NON_NEGATIVE_INT.test(value) && Number(value) <= 12, 'invalid'),
    decimalMark: z.string().trim().min(1).max(1),
    thousandSeparator: z.string().trim().min(1).max(1),
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

  const schema = useMemo(() => buildSchema(conflictingCodes), [conflictingCodes])

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
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('config.currencies.custom.form.validation.required')
                      : undefined}
                  </FormMessage>
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
                  <FormMessage>
                    {form.formState.errors.code?.message === 'conflict'
                      ? t('config.currencies.custom.form.validation.codeConflict')
                      : form.formState.errors.code
                        ? t('config.currencies.custom.form.validation.codeLength')
                        : undefined}
                  </FormMessage>
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
                    <FormMessage>
                      {form.formState.errors.precision
                        ? t('config.currencies.custom.form.validation.precision')
                        : undefined}
                    </FormMessage>
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
                    <FormMessage>
                      {form.formState.errors.decimalMark
                        ? t('config.currencies.custom.form.validation.singleChar')
                        : undefined}
                    </FormMessage>
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
                    <FormMessage>
                      {form.formState.errors.thousandSeparator
                        ? t('config.currencies.custom.form.validation.singleChar')
                        : undefined}
                    </FormMessage>
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
