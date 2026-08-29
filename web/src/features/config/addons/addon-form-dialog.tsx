import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { useFieldArray, useForm, useWatch, type Control } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { Addon } from '@openmeter/client'
import { Plus, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAllFeatures, useCreateAddon, useUpdateAddon } from '@/api/hooks'
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
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  defaultRateCard,
  fromRateCardToForm,
  rateCardSchema,
  toRateCardInput,
  type RateCardFormValues,
} from '@/features/config/plans/plan-form-schema'
import { FieldError } from '@/features/config/plans/price-editor'

// 与 plans 契约同域的 key 规则（plan-form-schema 的 RESOURCE_KEY 不导出，此处同型复刻）。
const ADDON_KEY = /^[a-z0-9]+(?:_[a-z0-9]+)*$/

type AddonFormValues = {
  name: string
  description: string
  key: string
  instanceType: 'single' | 'multiple'
  currency: string
  rateCards: RateCardFormValues[]
}

const EMPTY_VALUES: AddonFormValues = {
  name: '',
  description: '',
  key: '',
  instanceType: 'single',
  currency: '',
  rateCards: [defaultRateCard()],
}

function fromAddonToForm(addon: Addon): AddonFormValues {
  return {
    name: addon.name,
    description: addon.description ?? '',
    key: addon.key,
    instanceType: addon.instanceType,
    currency: addon.currency,
    rateCards:
      addon.rateCards.length > 0
        ? addon.rateCards.map(fromRateCardToForm)
        : [defaultRateCard()],
  }
}

function buildSchema(isCreate: boolean, t: (key: string) => string) {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, t('config.addons.form.validation.required'))
      .max(256, t('config.addons.form.validation.required')),
    description: z.string().trim().max(1024),
    // key 与 currency 创建后不可变：编辑态字段禁用且不参与校验。
    key: isCreate
      ? z
          .string()
          .trim()
          .min(1, t('config.addons.form.validation.required'))
          .max(64)
          .refine(
            (value) => ADDON_KEY.test(value),
            'config.plans.wizard.errors.keyFormat'
          )
      : z.string(),
    instanceType: z.enum(['single', 'multiple']),
    currency: isCreate
      ? z
          .string()
          .trim()
          .min(1, t('config.addons.form.validation.required'))
          .max(24, t('config.addons.form.validation.required'))
      : z.string(),
    rateCards: z
      .array(rateCardSchema)
      .min(1, 'config.addons.form.validation.rateCardsRequired'),
  })
}

/** 价格类型切换后的重置值（addons 不含 tiered）。 */
function resetPrice(kind: string): RateCardFormValues['price'] {
  if (kind === 'free') return { kind: 'free' }
  if (kind === 'flat') return { kind: 'flat', amount: '' }
  return { kind: 'unit', amount: '' }
}

/**
 * Create/edit dialog. The rate-card rows reuse the plans contract
 * (rateCardSchema + toRateCardInput/fromRateCardToForm) so both surfaces stay
 * in sync; addons expose free/flat/unit prices only (no tiered).
 */
export function AddonFormDialog({
  open,
  onOpenChange,
  addon,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  addon: Addon | null
}) {
  const { t } = useTranslation()
  const isCreate = addon === null
  const createMutation = useCreateAddon()
  const updateMutation = useUpdateAddon()
  const mutation = isCreate ? createMutation : updateMutation

  const schema = useMemo(() => buildSchema(isCreate, t), [isCreate, t])
  const form = useForm<AddonFormValues>({
    resolver: zodResolver(schema),
    defaultValues: EMPTY_VALUES,
  })
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'rateCards',
  })

  /** 计费类型切换后价格重置（free/unit），避免残留非法组合。 */
  const handleTypeChange = (
    index: number,
    value: 'flat_fee' | 'usage_based'
  ) => {
    form.setValue(`rateCards.${index}.type`, value)
    form.setValue(
      `rateCards.${index}.price`,
      value === 'flat_fee' ? resetPrice('free') : resetPrice('unit'),
      { shouldValidate: true }
    )
  }

  /** 价格类型切换后重置金额/周期形态。 */
  const handlePriceKindChange = (
    index: number,
    value: 'free' | 'flat' | 'unit'
  ) => {
    form.setValue(`rateCards.${index}.price.kind`, value)
    form.setValue(`rateCards.${index}.price`, resetPrice(value), {
      shouldValidate: true,
    })
  }

  useEffect(() => {
    if (open) form.reset(addon ? fromAddonToForm(addon) : EMPTY_VALUES)
  }, [open, addon, form])

  const onSubmit = (values: AddonFormValues) => {
    const rateCards = values.rateCards.map(toRateCardInput)
    if (isCreate) {
      createMutation.mutate(
        {
          name: values.name.trim(),
          description: values.description.trim() || undefined,
          key: values.key.trim(),
          instanceType: values.instanceType,
          currency: values.currency.trim(),
          rateCards,
        },
        {
          onSuccess: () => {
            toast.success(t('config.addons.toast.created'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
      return
    }
    updateMutation.mutate(
      {
        addonId: addon.id,
        body: {
          name: values.name.trim(),
          description: values.description.trim() || undefined,
          instanceType: values.instanceType,
          rateCards,
        },
      },
      {
        onSuccess: () => {
          toast.success(t('config.addons.toast.updated'))
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {isCreate
              ? t('config.addons.form.createTitle')
              : t('config.addons.form.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('config.addons.form.createDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='name'
                render={({ field, fieldState }) => (
                  <FormItem>
                    <FormLabel>{t('config.addons.form.name')}</FormLabel>
                    <FormControl>
                      <Input placeholder='Support Package' {...field} />
                    </FormControl>
                    <FieldError message={fieldState.error?.message} />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='key'
                render={({ field, fieldState }) => (
                  <FormItem>
                    <FormLabel>{t('config.addons.form.key')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='support_package'
                        autoComplete='off'
                        disabled={!isCreate}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('config.addons.form.keyHint')}
                    </FormDescription>
                    {isCreate && (
                      <FieldError message={fieldState.error?.message} />
                    )}
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='description'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.addons.form.description')}（
                    {t('common.optional')}）
                  </FormLabel>
                  <FormControl>
                    <Input maxLength={1024} {...field} />
                  </FormControl>
                </FormItem>
              )}
            />
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='instanceType'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.addons.form.instanceType')}
                    </FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value='single'>
                          {t('config.addons.form.instanceTypes.single')}
                        </SelectItem>
                        <SelectItem value='multiple'>
                          {t('config.addons.form.instanceTypes.multiple')}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='currency'
                render={({ field, fieldState }) => (
                  <FormItem>
                    <FormLabel>{t('config.addons.form.currency')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='CNY'
                        maxLength={24}
                        disabled={!isCreate}
                        {...field}
                      />
                    </FormControl>
                    {isCreate && (
                      <FieldError message={fieldState.error?.message} />
                    )}
                  </FormItem>
                )}
              />
            </div>

            <Separator />

            <div className='space-y-1'>
              <FormLabel>{t('config.addons.form.rateCards')}</FormLabel>
              <FieldError message={form.formState.errors.rateCards?.message} />
            </div>
            {fields.map((fieldItem, index) => (
              <RateCardFields
                key={fieldItem.id}
                control={form.control}
                index={index}
                canRemove={fields.length > 1}
                onRemove={() => remove(index)}
                onTypeChange={(value) => handleTypeChange(index, value)}
                onPriceKindChange={(value) =>
                  handlePriceKindChange(index, value)
                }
              />
            ))}
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => append(defaultRateCard())}
            >
              <Plus className='size-4' />
              {t('config.addons.form.addRateCard')}
            </Button>

            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => onOpenChange(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type='submit' disabled={mutation.isPending}>
                {mutation.isPending
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

type RateCardFieldsProps = {
  control: Control<AddonFormValues>
  index: number
  canRemove: boolean
  onRemove: () => void
  onTypeChange: (value: 'flat_fee' | 'usage_based') => void
  onPriceKindChange: (value: 'free' | 'flat' | 'unit') => void
}

/**
 * One rate-card row. `useWatch` (not form.watch) keeps the conditional
 * fields reactive without breaking the react-hooks lint contract; type and
 * price-kind switches reset the dependent fields via the parent handlers.
 */
function RateCardFields({
  control,
  index,
  canRemove,
  onRemove,
  onTypeChange,
  onPriceKindChange,
}: RateCardFieldsProps) {
  const { t } = useTranslation()
  const { data: features } = useAllFeatures()
  const type = useWatch({ control, name: `rateCards.${index}.type` })
  const priceKind = useWatch({
    control,
    name: `rateCards.${index}.price.kind`,
  })

  return (
    <div className='space-y-3 rounded-md border p-3'>
      <div className='grid grid-cols-2 gap-3'>
        <FormField
          control={control}
          name={`rateCards.${index}.name`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.addons.form.card.name')}</FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`rateCards.${index}.key`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.addons.form.card.key')}</FormLabel>
              <FormControl>
                <Input
                  placeholder='support_monthly'
                  autoComplete='off'
                  {...field}
                />
              </FormControl>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
      </div>
      <div className='grid grid-cols-2 gap-3'>
        <FormField
          control={control}
          name={`rateCards.${index}.type`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.addons.form.card.type')}</FormLabel>
              <Select
                onValueChange={(value) =>
                  onTypeChange(value as 'flat_fee' | 'usage_based')
                }
                value={field.value}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value='flat_fee'>
                    {t('config.addons.form.card.types.flat_fee')}
                  </SelectItem>
                  <SelectItem value='usage_based'>
                    {t('config.addons.form.card.types.usage_based')}
                  </SelectItem>
                </SelectContent>
              </Select>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
        {type === 'usage_based' && (
          <FormField
            control={control}
            name={`rateCards.${index}.featureId`}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormLabel>{t('config.addons.form.card.feature')}</FormLabel>
                <Select
                  onValueChange={field.onChange}
                  value={field.value || undefined}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {(features ?? []).map((feature) => (
                      <SelectItem key={feature.id} value={feature.id}>
                        {feature.name}（{feature.key}）
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
        )}
      </div>
      <div className='grid grid-cols-3 gap-3'>
        <FormField
          control={control}
          name={`rateCards.${index}.price.kind`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('config.addons.form.card.priceKind')}</FormLabel>
              <Select
                onValueChange={(value) =>
                  onPriceKindChange(value as 'free' | 'flat' | 'unit')
                }
                value={field.value}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value='free'>
                    {t('config.addons.form.card.priceKinds.free')}
                  </SelectItem>
                  <SelectItem value='flat'>
                    {t('config.addons.form.card.priceKinds.flat')}
                  </SelectItem>
                  <SelectItem value='unit'>
                    {t('config.addons.form.card.priceKinds.unit')}
                  </SelectItem>
                </SelectContent>
              </Select>
            </FormItem>
          )}
        />
        {priceKind !== 'free' && (
          <FormField
            control={control}
            name={`rateCards.${index}.price.amount`}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormLabel>{t('config.addons.form.card.amount')}</FormLabel>
                <FormControl>
                  <Input inputMode='decimal' placeholder='10' {...field} />
                </FormControl>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
        )}
        {priceKind === 'flat' && (
          <FormField
            control={control}
            name={`rateCards.${index}.billingCadence`}
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('config.addons.form.card.billingCadence')}
                </FormLabel>
                <Select
                  onValueChange={(value) =>
                    field.onChange(value === 'ONE_TIME' ? null : value)
                  }
                  value={field.value ?? 'ONE_TIME'}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value='ONE_TIME'>
                      {t('config.addons.form.card.cadences.oneTime')}
                    </SelectItem>
                    <SelectItem value='P1M'>
                      {t('config.addons.form.card.cadences.P1M')}
                    </SelectItem>
                    <SelectItem value='P1Y'>
                      {t('config.addons.form.card.cadences.P1Y')}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </FormItem>
            )}
          />
        )}
      </div>
      <div className='flex justify-end'>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          disabled={!canRemove}
          onClick={onRemove}
        >
          <X className='size-4' />
          {t('config.addons.form.removeRateCard')}
        </Button>
      </div>
    </div>
  )
}
