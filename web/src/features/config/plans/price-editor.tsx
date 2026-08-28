import { useWatch, type Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
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
import type { PlanWizardValues, PriceFormValue } from './plan-form-schema'

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
    <div className='grid grid-cols-2 gap-4'>
      <FormField
        control={control}
        name={pricePath as never}
        render={({ field, fieldState }) => (
          <FormItem>
            <FormLabel>{t('config.plans.wizard.fields.priceKind')}</FormLabel>
            <Select
              value={kind}
              onValueChange={(value) =>
                field.onChange(
                  value === 'free'
                    ? { kind: 'free' }
                    : { kind: 'flat', amount: '' }
                )
              }
            >
              <FormControl>
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectItem value='free'>{t('plan.priceType.free')}</SelectItem>
                <SelectItem value='flat'>{t('plan.priceType.flat')}</SelectItem>
              </SelectContent>
            </Select>
            <FieldError message={fieldState.error?.message} />
          </FormItem>
        )}
      />
      {kind === 'flat' && (
        <FormField
          control={control}
          name={`${pricePath}.amount` as never}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>
                {t('config.plans.wizard.fields.amount', { currency })}
              </FormLabel>
              <FormControl>
                <Input inputMode='decimal' placeholder='99.00' {...field} />
              </FormControl>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
      )}
    </div>
  )
}
