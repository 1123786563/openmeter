import { useEffect, useState } from 'react'
import {
  useFieldArray,
  useForm,
  useWatch,
  type FieldPath,
  type UseFormReturn,
} from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { Feature, Plan } from '@openmeter/client'
import { ArrowLeft, ArrowRight, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAllFeatures, useCreatePlan, useUpdatePlan } from '@/api/hooks'
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
  FormField,
  FormItem,
  FormLabel,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  EMPTY_PLAN,
  defaultPhase,
  defaultRateCard,
  fromPlanToWizardValues,
  planWizardSchema,
  toCreatePlanRequest,
  toUpsertPlanRequest,
  type PlanWizardValues,
} from './plan-form-schema'
import { FieldError, PriceEditor } from './price-editor'

const STEPS = ['basics', 'phases', 'rateCards'] as const
type Step = (typeof STEPS)[number]

/** #7 起 usage_based 可用；#8 扩展价格类型时不改此列表。 */
const RATE_CARD_TYPES = ['flat_fee', 'usage_based'] as const

export type PlanFormWizardProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 传入 draft 计划进入编辑模式（回填 + PUT 提交）。 */
  plan?: Plan
}

export function PlanFormWizard({ open, onOpenChange, plan }: PlanFormWizardProps) {
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>('basics')

  const isEdit = Boolean(plan)
  const createPlan = useCreatePlan()
  const updatePlan = useUpdatePlan()

  const form = useForm<PlanWizardValues>({
    resolver: zodResolver(planWizardSchema),
    defaultValues: EMPTY_PLAN,
    mode: 'onChange',
  })

  useEffect(() => {
    if (open) {
      form.reset(plan ? fromPlanToWizardValues(plan) : EMPTY_PLAN)
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reopen must return to the first wizard step
      setStep('basics')
    }
  }, [open, plan, form])

  const {
    fields: phaseFields,
    append: appendPhase,
    remove: removePhase,
  } = useFieldArray({ control: form.control, name: 'phases' })

  const currency = useWatch({ control: form.control, name: 'currency' })

  const next = async () => {
    if (step === 'basics') {
      if (
        await form.trigger([
          'name',
          'key',
          'currency',
          'billingCadence',
          'description',
        ])
      ) {
        setStep('phases')
      }
    } else if (step === 'phases') {
      // Gate on the leaf paths mounted in this step. Triggering the whole
      // 'phases' subtree would always fail here: the resolver returns the full
      // error tree and default rate cards are only fixable in the next step.
      const phaseFieldNames = phaseFields.flatMap<FieldPath<PlanWizardValues>>(
        (_, i) => [
          `phases.${i}.name`,
          `phases.${i}.key`,
          `phases.${i}.duration`,
        ]
      )
      if (await form.trigger(phaseFieldNames)) {
        setStep('rateCards')
      }
    }
  }

  const onSubmit = (values: PlanWizardValues) => {
    if (plan) {
      updatePlan.mutate(
        { planId: plan.id, body: toUpsertPlanRequest(values) },
        {
          onSuccess: () => {
            toast.success(t('config.plans.wizard.toast.updated'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    } else {
      // api.plans.create 的请求形状就是 body 本体（SDK CreatePlanRequest 无包装）。
      createPlan.mutate(toCreatePlanRequest(values), {
        onSuccess: () => {
          toast.success(t('config.plans.wizard.toast.created'))
          onOpenChange(false)
        },
        onError: handleServerError,
      })
    }
  }

  const stepIndex = STEPS.indexOf(step)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>
            {isEdit
              ? t('config.plans.wizard.editTitle')
              : t('config.plans.wizard.createTitle')}
          </DialogTitle>
          <DialogDescription>
            {t(`config.plans.wizard.steps.${step}`)} ({stepIndex + 1}/
            {STEPS.length})
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form
            id='plan-form-wizard'
            onSubmit={form.handleSubmit(onSubmit)}
            className='space-y-4'
          >
            {step === 'basics' && (
              <>
                {isEdit && (
                  <p className='text-xs text-muted-foreground'>
                    {t('config.plans.wizard.immutableHint')}
                  </p>
                )}
                <div className='grid grid-cols-2 gap-4'>
                  <FormField
                    control={form.control}
                    name='name'
                    render={({ field, fieldState }) => (
                      <FormItem>
                        <FormLabel>{t('config.plans.fields.name')}</FormLabel>
                        <FormControl>
                          <Input placeholder='专业版' {...field} />
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
                        <FormLabel>{t('config.plans.fields.key')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder='pro_plan'
                            autoComplete='off'
                            disabled={isEdit}
                            {...field}
                          />
                        </FormControl>
                        <FieldError message={fieldState.error?.message} />
                      </FormItem>
                    )}
                  />
                </div>
                <div className='grid grid-cols-2 gap-4'>
                  <FormField
                    control={form.control}
                    name='currency'
                    render={({ field, fieldState }) => (
                      <FormItem>
                        <FormLabel>
                          {t('config.plans.fields.currency')}
                        </FormLabel>
                        <FormControl>
                          <Input
                            placeholder='CNY'
                            maxLength={24}
                            disabled={isEdit}
                            {...field}
                          />
                        </FormControl>
                        <FieldError message={fieldState.error?.message} />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name='billingCadence'
                    render={({ field, fieldState }) => (
                      <FormItem>
                        <FormLabel>
                          {t('config.plans.fields.billingCadence')}
                        </FormLabel>
                        <RadioGroup
                          value={field.value}
                          onValueChange={(value) =>
                            field.onChange(value as 'P1M' | 'P1Y')
                          }
                          disabled={isEdit}
                        >
                          <div className='flex items-center gap-2'>
                            <RadioGroupItem
                              value='P1M'
                              id='cadence-p1m'
                              disabled={isEdit}
                            />
                            <Label
                              htmlFor='cadence-p1m'
                              className='font-normal'
                            >
                              {t('config.plans.wizard.cadence.P1M')}
                            </Label>
                          </div>
                          <div className='flex items-center gap-2'>
                            <RadioGroupItem
                              value='P1Y'
                              id='cadence-p1y'
                              disabled={isEdit}
                            />
                            <Label
                              htmlFor='cadence-p1y'
                              className='font-normal'
                            >
                              {t('config.plans.wizard.cadence.P1Y')}
                            </Label>
                          </div>
                        </RadioGroup>
                        <FieldError message={fieldState.error?.message} />
                      </FormItem>
                    )}
                  />
                </div>
                <FormField
                  control={form.control}
                  name='description'
                  render={({ field, fieldState }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.plans.fields.description')}
                      </FormLabel>
                      <FormControl>
                        <Textarea
                          rows={2}
                          maxLength={1024}
                          {...field}
                          value={field.value ?? ''}
                        />
                      </FormControl>
                      <FieldError message={fieldState.error?.message} />
                    </FormItem>
                  )}
                />
              </>
            )}

            {step === 'phases' && (
              <div className='space-y-3'>
                {phaseFields.map((phaseField, index) => (
                  <div
                    key={phaseField.id}
                    className='grid grid-cols-[1fr_1fr_1fr_auto] items-start gap-3 rounded-lg border p-3'
                  >
                    <FormField
                      control={form.control}
                      name={`phases.${index}.name`}
                      render={({ field, fieldState }) => (
                        <FormItem>
                          <FormLabel>
                            {t('config.plans.wizard.phaseName', {
                              index: index + 1,
                            })}
                          </FormLabel>
                          <FormControl>
                            <Input placeholder='标准期' {...field} />
                          </FormControl>
                          <FieldError message={fieldState.error?.message} />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`phases.${index}.key`}
                      render={({ field, fieldState }) => (
                        <FormItem>
                          <FormLabel>{t('config.plans.fields.key')}</FormLabel>
                          <FormControl>
                            <Input
                              placeholder='standard'
                              autoComplete='off'
                              {...field}
                            />
                          </FormControl>
                          <FieldError message={fieldState.error?.message} />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`phases.${index}.duration`}
                      render={({ field, fieldState }) => (
                        <FormItem>
                          <FormLabel>
                            {t('config.plans.wizard.fields.duration')}
                          </FormLabel>
                          <FormControl>
                            <Input placeholder='P1M' {...field} />
                          </FormControl>
                          <FieldError message={fieldState.error?.message} />
                          {index === phaseFields.length - 1 && (
                            <p className='text-xs text-muted-foreground'>
                              {t('config.plans.wizard.durationHint')}
                            </p>
                          )}
                        </FormItem>
                      )}
                    />
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      className='mt-7 text-destructive hover:text-destructive'
                      disabled={phaseFields.length === 1}
                      onClick={() => removePhase(index)}
                    >
                      <Trash2 className='size-4' />
                    </Button>
                  </div>
                ))}
                <FieldError
                  message={
                    form.formState.errors.phases?.message as string | undefined
                  }
                />
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => appendPhase(defaultPhase())}
                >
                  <Plus className='size-4' />
                  {t('config.plans.wizard.addPhase')}
                </Button>
              </div>
            )}

            {step === 'rateCards' &&
              phaseFields.map((phaseField, phaseIndex) => (
                <PhaseRateCardsSection
                  key={phaseField.id}
                  form={form}
                  phaseIndex={phaseIndex}
                  currency={currency ?? 'CNY'}
                />
              ))}
          </form>
        </Form>

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
            <Button onClick={() => void next()}>
              {t('common.next')}
              <ArrowRight className='size-4' />
            </Button>
          ) : (
            <Button
              type='submit'
              form='plan-form-wizard'
              disabled={createPlan.isPending || updatePlan.isPending}
            >
              {createPlan.isPending || updatePlan.isPending
                ? t('common.submitting')
                : isEdit
                  ? t('config.plans.wizard.editSubmit')
                  : t('config.plans.wizard.createSubmit')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** 一个阶段内的价目卡行编辑（hooks 不能写在循环里，故抽成子组件）。 */
function PhaseRateCardsSection({
  form,
  phaseIndex,
  currency,
}: {
  form: UseFormReturn<PlanWizardValues>
  phaseIndex: number
  currency: string
}) {
  const { t } = useTranslation()
  // feature 必选下拉的唯一数据源：全量列表（分页版会只出现第一页功能）。
  const { data: features, isLoading: featuresLoading } = useAllFeatures()
  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: `phases.${phaseIndex}.rateCards`,
  })
  const phase = useWatch({
    control: form.control,
    name: `phases.${phaseIndex}`,
  })

  return (
    <div className='space-y-3 rounded-lg border p-3'>
      <div className='flex items-baseline justify-between'>
        <span className='text-sm font-medium'>
          {t('config.plans.wizard.phaseTitle', {
            index: phaseIndex + 1,
            name: phase?.name ?? '',
          })}
          <span className='ms-2 text-xs font-normal text-muted-foreground'>
            {phase?.duration || t('config.plans.detail.noDuration')}
          </span>
        </span>
      </div>
      {fields.map((cardField, cardIndex) => (
        <RateCardRow
          key={cardField.id}
          form={form}
          phaseIndex={phaseIndex}
          cardIndex={cardIndex}
          currency={currency}
          features={features}
          featuresLoading={featuresLoading}
          onRemove={() => remove(cardIndex)}
          canRemove={fields.length > 1}
        />
      ))}
      <FieldError
        message={
          form.formState.errors.phases?.[phaseIndex]?.rateCards?.message as
            | string
            | undefined
        }
      />
      <Button
        type='button'
        variant='outline'
        size='sm'
        onClick={() => append(defaultRateCard())}
      >
        <Plus className='size-4' />
        {t('config.plans.wizard.addRateCard')}
      </Button>
    </div>
  )
}

/** 单张价目卡行（hooks 不能写在 map 里）。 */
function RateCardRow({
  form,
  phaseIndex,
  cardIndex,
  currency,
  features,
  featuresLoading,
  onRemove,
  canRemove,
}: {
  form: UseFormReturn<PlanWizardValues>
  phaseIndex: number
  cardIndex: number
  currency: string
  features: Feature[] | undefined
  featuresLoading: boolean
  onRemove: () => void
  canRemove: boolean
}) {
  const { t } = useTranslation()
  const cardType = useWatch({
    control: form.control,
    name: `phases.${phaseIndex}.rateCards.${cardIndex}.type`,
  }) as 'flat_fee' | 'usage_based'
  const priceKind = useWatch({
    control: form.control,
    name: `phases.${phaseIndex}.rateCards.${cardIndex}.price.kind`,
  })

  return (
    <div className='space-y-3 rounded-md border bg-muted/30 p-3'>
      <div className='grid grid-cols-3 gap-3'>
        <FormField
          control={form.control}
          name={`phases.${phaseIndex}.rateCards.${cardIndex}.type`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>
                {t('config.plans.wizard.fields.rateCardType')}
              </FormLabel>
              <Select
                value={field.value}
                onValueChange={(value) => {
                  field.onChange(value)
                  // 切换类型时把 price 重置为该类型的合法形态，
                  // 避免残留非法组合（usage_based+flat）卡在校验上。
                  form.setValue(
                    `phases.${phaseIndex}.rateCards.${cardIndex}.price`,
                    value === 'usage_based'
                      ? { kind: 'unit', amount: '' }
                      : { kind: 'flat', amount: '' },
                    { shouldValidate: true }
                  )
                }}
              >
                <FormControl>
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {RATE_CARD_TYPES.map((type) => (
                    <SelectItem key={type} value={type}>
                      {t(`config.plans.wizard.rateCardType.${type}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={`phases.${phaseIndex}.rateCards.${cardIndex}.name`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.plans.detail.rateCardName')}</FormLabel>
              <FormControl>
                <Input placeholder='平台费' {...field} />
              </FormControl>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={`phases.${phaseIndex}.rateCards.${cardIndex}.key`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.plans.fields.key')}</FormLabel>
              <FormControl>
                <Input
                  placeholder='platform_fee'
                  autoComplete='off'
                  {...field}
                />
              </FormControl>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
      </div>
      <div className='grid grid-cols-3 gap-3'>
        {cardType === 'usage_based' && (
          <FormField
            control={form.control}
            name={`phases.${phaseIndex}.rateCards.${cardIndex}.featureId`}
            render={({ field, fieldState }) => (
              <FormItem>
                <FormLabel>{t('config.plans.wizard.fields.feature')}</FormLabel>
                <Select
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                >
                  <FormControl>
                    <SelectTrigger className='w-full'>
                      <SelectValue
                        placeholder={t(
                          'config.plans.wizard.fields.featurePlaceholder'
                        )}
                      />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {featuresLoading ? (
                      <SelectItem value='_loading' disabled>
                        {t('common.loading')}
                      </SelectItem>
                    ) : (
                      (features ?? []).map((feature) => (
                        <SelectItem key={feature.id} value={feature.id}>
                          {feature.name}（{feature.key}）
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
                <FieldError message={fieldState.error?.message} />
              </FormItem>
            )}
          />
        )}
        <FormField
          control={form.control}
          name={`phases.${phaseIndex}.rateCards.${cardIndex}.billingCadence`}
          render={({ field, fieldState }) => (
            <FormItem>
              <FormLabel>{t('config.plans.detail.cadence')}</FormLabel>
              <Select
                value={field.value ?? 'one_time'}
                onValueChange={(value) =>
                  field.onChange(
                    value === 'one_time' ? null : (value as 'P1M' | 'P1Y')
                  )
                }
              >
                <FormControl>
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value='P1M'>
                    {t('config.plans.wizard.cadence.P1M')}
                  </SelectItem>
                  <SelectItem value='P1Y'>
                    {t('config.plans.wizard.cadence.P1Y')}
                  </SelectItem>
                  {/* spec：一次性（null cadence）仅固定费价卡合法。 */}
                  {(priceKind === 'free' || priceKind === 'flat') && (
                    <SelectItem value='one_time'>
                      {t('config.plans.wizard.cadence.oneTime')}
                    </SelectItem>
                  )}
                </SelectContent>
              </Select>
              <FieldError message={fieldState.error?.message} />
            </FormItem>
          )}
        />
      </div>
      <PriceEditor
        control={form.control}
        phaseIndex={phaseIndex}
        cardIndex={cardIndex}
        currency={currency}
      />
      <div className='flex justify-end'>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          className='text-destructive hover:text-destructive'
          disabled={!canRemove}
          onClick={onRemove}
        >
          <Trash2 className='size-4' />
          {t('config.plans.wizard.removeRateCard')}
        </Button>
      </div>
    </div>
  )
}
