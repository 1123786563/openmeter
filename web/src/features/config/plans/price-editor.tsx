import { useFieldArray, useWatch, type Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import {
  FormControl,
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
import { Button } from '@/components/ui/button'
import {
  defaultTier,
  type PlanWizardValues,
  type PriceFormValue,
  type TierFormValues,
} from './plan-form-schema'

export type PriceEditorProps = {
  control: Control<PlanWizardValues>
  phaseIndex: number
  cardIndex: number
  currency: string
}

/** FormMessage 的替代：zod message 是 i18n key，这里完成翻译。 */
export function FieldError({ message }: { message?: string }) {
  const { t } = useTranslation()
  if (!message) return null
  return (
    <p className='text-sm font-medium text-destructive'>
      {t(message, { defaultValue: message })}
    </p>
  )
}

/** kind 切换后的重置值；tiered 分支为本任务新增。 */
function resetPrice(kind: string): PriceFormValue {
  if (kind === 'free') return { kind: 'free' }
  if (kind === 'unit') return { kind: 'unit', amount: '' }
  if (kind === 'tiered') {
    return { kind: 'tiered', mode: 'graduated', tiers: [defaultTier()] }
  }
  return { kind: 'flat', amount: '' }
}

/**
 * 编辑一张价目卡的 price 判别联合（kind 切换会重置其余字段）。
 * #7 增加 unit 分支，#8 增加 tiered 分支，props 不变。
 */
export function PriceEditor({
  control,
  phaseIndex,
  cardIndex,
  currency,
}: PriceEditorProps) {
  const { t } = useTranslation()
  const pricePath = `phases.${phaseIndex}.rateCards.${cardIndex}.price`
  const kind = useWatch({
    control,
    name: `phases.${phaseIndex}.rateCards.${cardIndex}.price.kind`,
  }) as PriceFormValue['kind']

  return (
    <div className='space-y-3'>
      <div className='grid grid-cols-2 gap-4'>
        <FormField
          control={control}
          name={pricePath as never}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.plans.wizard.fields.priceKind')}</FormLabel>
              <Select
                value={kind}
                onValueChange={(value) => field.onChange(resetPrice(value))}
              >
                <FormControl>
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value='free'>{t('plan.priceType.free')}</SelectItem>
                  <SelectItem value='flat'>{t('plan.priceType.flat')}</SelectItem>
                  <SelectItem value='unit'>{t('plan.priceType.unit')}</SelectItem>
                  <SelectItem value='tiered'>
                    {t('plan.priceType.tiered')}
                  </SelectItem>
                </SelectContent>
              </Select>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
        {(kind === 'flat' || kind === 'unit') && (
          <FormField
            control={control}
            name={`${pricePath}.amount` as never}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormLabel>
                  {kind === 'unit'
                    ? t('config.plans.wizard.fields.unitAmount', { currency })
                    : t('config.plans.wizard.fields.amount', { currency })}
                </FormLabel>
                <FormControl>
                  <Input inputMode='decimal' placeholder='0.05' {...field} />
                </FormControl>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
        )}
        {kind === 'tiered' && (
          <FormField
            control={control}
            name={`${pricePath}.mode` as never}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormLabel>{t('config.plans.wizard.fields.tierMode')}</FormLabel>
                <Select
                  value={field.value as 'graduated' | 'volume'}
                  onValueChange={(value) =>
                    field.onChange(value as 'graduated' | 'volume')
                  }
                >
                  <FormControl>
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value='graduated'>
                      {t('config.plans.wizard.tierMode.graduated')}
                    </SelectItem>
                    <SelectItem value='volume'>
                      {t('config.plans.wizard.tierMode.volume')}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
        )}
      </div>
      {kind === 'tiered' && (
        <TierEditor control={control} pricePath={pricePath} currency={currency} />
      )}
    </div>
  )
}

/** 阶梯行编辑器：起止量 × 单价 × 固定费，增删行；连续性错误由 schema 定位到行。 */
function TierEditor({
  control,
  pricePath,
  currency,
}: {
  control: Control<PlanWizardValues>
  pricePath: string
  currency: string
}) {
  const { t } = useTranslation()
  const tiersPath = `${pricePath}.tiers`
  const { fields, append, remove } = useFieldArray({
    control,
    name: tiersPath as never,
  })
  const tiers = useWatch({ control, name: tiersPath as never }) as
    | TierFormValues[]
    | undefined

  const addTier = () => {
    const current = tiers ?? []
    const last = current[current.length - 1]
    // 新档从上一档 lastUnit+1 接续；上一档从此必须闭合（校验会提醒补 lastUnit）。
    const firstUnit =
      last && last.lastUnit !== '' ? String(Number(last.lastUnit) + 1) : '0'
    append({ ...defaultTier(), firstUnit })
  }

  return (
    <div className='space-y-2 rounded-md border bg-muted/30 p-3'>
      <div className='grid grid-cols-[1fr_1fr_1fr_1fr_auto] gap-2 text-xs text-muted-foreground'>
        <span>{t('config.plans.wizard.fields.tierFirstUnit')}</span>
        <span>{t('config.plans.wizard.fields.tierLastUnit')}</span>
        <span>{t('config.plans.wizard.fields.tierUnitAmount', { currency })}</span>
        <span>{t('config.plans.wizard.fields.tierFlatAmount', { currency })}</span>
        <span />
      </div>
      {fields.map((tierField, index) => (
        <div
          key={tierField.id}
          className='grid grid-cols-[1fr_1fr_1fr_1fr_auto] items-start gap-2'
        >
          <FormField
            control={control}
            name={`${tiersPath}.${index}.firstUnit` as never}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormControl>
                  <Input
                    inputMode='numeric'
                    placeholder='0'
                    disabled={index === 0}
                    {...field}
                  />
                </FormControl>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
          <FormField
            control={control}
            name={`${tiersPath}.${index}.lastUnit` as never}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormControl>
                  <Input
                    inputMode='numeric'
                    placeholder={
                      index === fields.length - 1
                        ? t('config.plans.wizard.tierLastPlaceholder')
                        : '100'
                    }
                    disabled={index === fields.length - 1}
                    {...field}
                  />
                </FormControl>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
          <FormField
            control={control}
            name={`${tiersPath}.${index}.unitAmount` as never}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormControl>
                  <Input inputMode='decimal' placeholder='0.10' {...field} />
                </FormControl>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
          <FormField
            control={control}
            name={`${tiersPath}.${index}.flatAmount` as never}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormControl>
                  <Input inputMode='decimal' placeholder='0' {...field} />
                </FormControl>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
          <Button
            type='button'
            variant='ghost'
            size='sm'
            className='mt-1 text-destructive hover:text-destructive'
            disabled={fields.length === 1}
            onClick={() => remove(index)}
          >
            <Trash2 className='size-4' />
          </Button>
        </div>
      ))}
      <Button type='button' variant='outline' size='sm' onClick={addTier}>
        <Plus className='size-4' />
        {t('config.plans.wizard.addTier')}
      </Button>
    </div>
  )
}
