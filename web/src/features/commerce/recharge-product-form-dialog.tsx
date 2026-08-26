import { useEffect } from 'react'
import { z } from 'zod'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { CommerceRechargeProduct } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateRechargeProduct, useUpdateRechargeProduct } from '@/api/hooks'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

const PRODUCT_KINDS = [
  'plan_purchase',
  'subscription_renewal',
  'wallet_top_up',
] as const

const REFUND_POLICIES = ['none', 'unspent', 'full_window'] as const

/**
 * Amounts are entered in the major currency unit (yuan) to match the list
 * display (`formatFen`), and converted to fen on submit.
 */
const POSITIVE_AMOUNT = /^\d+(\.\d{1,2})?$/
const NON_NEGATIVE_INT = /^\d+$/

/**
 * Shared form shape. Create sends every field; edit only sends the PATCH's
 * mutable subset (display name, price, listing state) — `sku`, `kind`,
 * `credits`, `currency`, and the policy/description fields are immutable and
 * rendered read-only (or omitted) in edit mode.
 */
const productSchema = z.object({
  sku: z.string().min(1).max(64),
  displayName: z.string().min(1).max(256),
  kind: z.enum(PRODUCT_KINDS),
  credits: z
    .string()
    .refine(
      (value) => NON_NEGATIVE_INT.test(value) && Number(value) > 0,
      'invalid'
    ),
  amountYuan: z
    .string()
    .refine(
      (value) => POSITIVE_AMOUNT.test(value) && Number(value) > 0,
      'invalid'
    ),
  currency: z.string().min(3).max(3),
  displayOrder: z
    .string()
    .refine((value) => value === '' || NON_NEGATIVE_INT.test(value), 'invalid'),
  refundPolicy: z.enum(['', ...REFUND_POLICIES]),
  description: z.string().max(1024).optional(),
  active: z.boolean(),
})

type ProductFormValues = z.infer<typeof productSchema>

const EMPTY_VALUES: ProductFormValues = {
  sku: '',
  displayName: '',
  kind: 'wallet_top_up',
  credits: '',
  amountYuan: '',
  currency: 'CNY',
  displayOrder: '',
  refundPolicy: '',
  description: '',
  active: true,
}

type RechargeProductFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Present when editing an existing product. */
  product?: CommerceRechargeProduct
}

export function RechargeProductFormDialog({
  open,
  onOpenChange,
  product,
}: RechargeProductFormDialogProps) {
  const { t } = useTranslation()
  const isCreate = !product

  const createMutation = useCreateRechargeProduct()
  const updateMutation = useUpdateRechargeProduct()

  const form = useForm<ProductFormValues>({
    resolver: zodResolver(productSchema),
    defaultValues: EMPTY_VALUES,
  })

  useEffect(() => {
    if (open) {
      form.reset(
        product
          ? {
              sku: '',
              displayName: product.name,
              kind: 'wallet_top_up',
              credits: product.credits.toString(),
              amountYuan: (Number(product.priceFen) / 100).toString(),
              currency: product.currency,
              displayOrder: product.displayOrder?.toString() ?? '',
              refundPolicy: '',
              description: '',
              active: product.active,
            }
          : EMPTY_VALUES
      )
    }
  }, [open, product, form])

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  // Live currency for the amount label; `useWatch` keeps the compiler happy.
  const currency = useWatch({ control: form.control, name: 'currency' })

  const onSubmit = (values: ProductFormValues) => {
    const amountFen = BigInt(Math.round(Number(values.amountYuan) * 100))
    if (isCreate) {
      createMutation.mutate(
        {
          sku: values.sku.trim(),
          displayName: values.displayName.trim(),
          kind: values.kind,
          credits: BigInt(values.credits),
          amountFen,
          currency: values.currency.trim().toUpperCase(),
          displayOrder: values.displayOrder
            ? Number(values.displayOrder)
            : undefined,
          refundPolicy: values.refundPolicy || undefined,
          description: values.description || undefined,
        },
        {
          onSuccess: () => {
            toast.success(t('commerce.rechargeProducts.toast.created'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    } else if (product) {
      updateMutation.mutate(
        {
          productId: product.id,
          body: {
            displayName: values.displayName.trim(),
            amountFen,
            active: values.active,
          },
        },
        {
          onSuccess: () => {
            toast.success(t('commerce.rechargeProducts.toast.updated'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {isCreate
              ? t('commerce.rechargeProducts.form.createTitle')
              : t('commerce.rechargeProducts.form.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {isCreate
              ? t('commerce.rechargeProducts.form.createDescription')
              : t('commerce.rechargeProducts.form.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            {isCreate && (
              <div className='grid grid-cols-2 gap-4'>
                <FormField
                  control={form.control}
                  name='sku'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.sku')}
                      </FormLabel>
                      <FormControl>
                        <Input
                          placeholder='points-1000'
                          autoComplete='off'
                          {...field}
                        />
                      </FormControl>
                      <FormMessage>
                        {form.formState.errors.sku
                          ? t(
                              'commerce.rechargeProducts.form.validation.required'
                            )
                          : undefined}
                      </FormMessage>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='kind'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.kind')}
                      </FormLabel>
                      <Select
                        value={field.value}
                        onValueChange={field.onChange}
                      >
                        <FormControl>
                          <SelectTrigger className='w-full'>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {PRODUCT_KINDS.map((kind) => (
                            <SelectItem key={kind} value={kind}>
                              {t(`commerce.orderKind.${kind}`)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </FormItem>
                  )}
                />
              </div>
            )}
            <FormField
              control={form.control}
              name='displayName'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('commerce.rechargeProducts.fields.displayName')}
                  </FormLabel>
                  <FormControl>
                    <Input placeholder='1,000 Points Pack' {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.displayName
                      ? t('commerce.rechargeProducts.form.validation.required')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            {isCreate && (
              <div className='grid grid-cols-2 gap-4'>
                <FormField
                  control={form.control}
                  name='credits'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.credits')}
                      </FormLabel>
                      <FormControl>
                        <Input
                          inputMode='numeric'
                          placeholder='1000'
                          {...field}
                        />
                      </FormControl>
                      <FormMessage>
                        {form.formState.errors.credits
                          ? t(
                              'commerce.rechargeProducts.form.validation.positiveInteger'
                            )
                          : undefined}
                      </FormMessage>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='currency'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.currency')}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder='CNY' maxLength={3} {...field} />
                      </FormControl>
                      <FormMessage>
                        {form.formState.errors.currency
                          ? t(
                              'commerce.rechargeProducts.form.validation.currency'
                            )
                          : undefined}
                      </FormMessage>
                    </FormItem>
                  )}
                />
              </div>
            )}
            <FormField
              control={form.control}
              name='amountYuan'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('commerce.rechargeProducts.fields.amountYuan', {
                      currency: currency || 'CNY',
                    })}
                  </FormLabel>
                  <FormControl>
                    <Input inputMode='decimal' placeholder='19.90' {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('commerce.rechargeProducts.form.amountHint')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.amountYuan
                      ? t('commerce.rechargeProducts.form.validation.amount')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            {isCreate && (
              <div className='grid grid-cols-2 gap-4'>
                <FormField
                  control={form.control}
                  name='displayOrder'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.displayOrder')}
                      </FormLabel>
                      <FormControl>
                        <Input inputMode='numeric' placeholder='0' {...field} />
                      </FormControl>
                      <FormMessage>
                        {form.formState.errors.displayOrder
                          ? t(
                              'commerce.rechargeProducts.form.validation.nonNegativeInteger'
                            )
                          : undefined}
                      </FormMessage>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='refundPolicy'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.refundPolicy')}
                      </FormLabel>
                      <Select
                        value={field.value || 'default'}
                        onValueChange={(value) =>
                          field.onChange(
                            value === 'default'
                              ? ''
                              : (value as ProductFormValues['refundPolicy'])
                          )
                        }
                      >
                        <FormControl>
                          <SelectTrigger className='w-full'>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value='default'>
                            {t(
                              'commerce.rechargeProducts.form.refundPolicyDefault'
                            )}
                          </SelectItem>
                          {REFUND_POLICIES.map((policy) => (
                            <SelectItem key={policy} value={policy}>
                              {t(
                                `commerce.rechargeProducts.refundPolicy.${policy}`
                              )}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </FormItem>
                  )}
                />
              </div>
            )}
            {isCreate && (
              <FormField
                control={form.control}
                name='description'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('commerce.rechargeProducts.fields.description')}
                    </FormLabel>
                    <FormControl>
                      <Textarea rows={2} {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
            )}
            {!isCreate && (
              <FormField
                control={form.control}
                name='active'
                render={({ field }) => (
                  <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                    <div className='space-y-0.5'>
                      <FormLabel>
                        {t('commerce.rechargeProducts.fields.active')}
                      </FormLabel>
                      <FormDescription>
                        {t('commerce.rechargeProducts.form.activeHint')}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            )}
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => onOpenChange(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type='submit' disabled={isSubmitting}>
                {isSubmitting ? t('common.submitting') : t('common.confirm')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
