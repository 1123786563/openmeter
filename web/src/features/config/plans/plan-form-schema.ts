import { z } from 'zod'
import type {
  CreatePlanRequestInput,
  PlanPhaseInput,
  RateCardInput,
} from '@openmeter/client'

const RESOURCE_KEY = /^[a-z0-9]+(?:_[a-z0-9]+)*$/
const ISO8601_DURATION =
  /^P(?:\d+(?:\.\d+)?Y)?(?:\d+(?:\.\d+)?M)?(?:\d+(?:\.\d+)?W)?(?:\d+(?:\.\d+)?D)?(?:T(?:\d+(?:\.\d+)?H)?(?:\d+(?:\.\d+)?M)?(?:\d+(?:\.\d+)?S)?)?$/
const AMOUNT = /^\d+(\.\d+)?$/
const CURRENCY = /^[A-Z]{3}$|^[A-Za-z0-9]{4,24}$/

/**
 * PriceForm 契约（本文件是唯一定义处）：
 * - #6 本任务：free | flat
 * - #7 扩展：+ { kind:'unit', amount }
 * - #8 扩展：+ { kind:'tiered', mode, tiers }
 * zod message 一律是 i18n key，由 FieldError 组件翻译。
 */
export const priceFormSchema = z.discriminatedUnion('kind', [
  z.object({ kind: z.literal('free') }),
  z.object({
    kind: z.literal('flat'),
    amount: z
      .string()
      .refine((value) => AMOUNT.test(value), 'config.plans.wizard.errors.amount'),
  }),
])

export type PriceFormValue = z.infer<typeof priceFormSchema>

export const rateCardSchema = z
  .object({
    key: z
      .string()
      .min(1, 'config.plans.wizard.errors.required')
      .max(64)
      .refine((value) => RESOURCE_KEY.test(value), 'config.plans.wizard.errors.keyFormat'),
    name: z.string().min(1, 'config.plans.wizard.errors.required').max(256),
    // 契约含两个值；#6 的 UI 只提供 flat_fee 选项，usage_based 在 #7 接入。
    type: z.enum(['flat_fee', 'usage_based']),
    // usage_based 必填（#7 的 superRefine 强制）；flat_fee 可空。
    featureId: z.string().optional(),
    // null = 一次性（仅固定费价卡合法）。v3 wire 上 rate card 的
    // billing_cadence 是 *ISO8601Duration，JSON null 与省略等价，映射时发 undefined。
    billingCadence: z.enum(['P1M', 'P1Y']).nullable(),
    price: priceFormSchema,
  })
  .superRefine((card, ctx) => {
    if (
      card.type === 'flat_fee' &&
      card.price.kind !== 'free' &&
      card.price.kind !== 'flat'
    ) {
      ctx.addIssue({
        code: 'custom',
        path: ['type'],
        message: 'config.plans.wizard.errors.flatFeePriceKind',
      })
    }
  })

export type RateCardFormValues = z.infer<typeof rateCardSchema>

export const phaseSchema = z.object({
  key: z
    .string()
    .min(1, 'config.plans.wizard.errors.required')
    .max(64)
    .refine((value) => RESOURCE_KEY.test(value), 'config.plans.wizard.errors.keyFormat'),
  name: z.string().min(1, 'config.plans.wizard.errors.required').max(256),
  // '' = 无期限（仅最后阶段允许，由 phasesSchema refine 定位到具体行）。
  duration: z
    .string()
    .refine(
      (value) => value === '' || ISO8601_DURATION.test(value),
      'config.plans.wizard.errors.durationFormat'
    ),
  rateCards: z
    .array(rateCardSchema)
    .min(1, 'config.plans.wizard.errors.rateCardsRequired'),
})

export type PhaseFormValues = z.infer<typeof phaseSchema>

export const phasesSchema = z
  .array(phaseSchema)
  .min(1, 'config.plans.wizard.errors.phasesRequired')
  .superRefine((phases, ctx) => {
    phases.forEach((phase, index) => {
      const isLast = index === phases.length - 1
      if (!isLast && phase.duration.trim() === '') {
        ctx.addIssue({
          code: 'custom',
          path: [index, 'duration'],
          message: 'config.plans.wizard.errors.phaseDurationRequired',
        })
      }
    })
  })

export const planWizardSchema = z.object({
  name: z.string().min(1, 'config.plans.wizard.errors.required').max(256),
  key: z
    .string()
    .min(1, 'config.plans.wizard.errors.required')
    .max(64)
    .refine((value) => RESOURCE_KEY.test(value), 'config.plans.wizard.errors.keyFormat'),
  description: z.string().max(1024).optional(),
  // 法币 3 位大写，或自定义货币代码 4-24 位（BillingCurrencyCode）。
  currency: z
    .string()
    .refine((value) => CURRENCY.test(value), 'config.plans.wizard.errors.currency'),
  billingCadence: z.enum(['P1M', 'P1Y']),
  phases: phasesSchema,
})

export type PlanWizardValues = z.infer<typeof planWizardSchema>

export function defaultRateCard(): RateCardFormValues {
  return {
    key: '',
    name: '',
    type: 'flat_fee',
    featureId: '',
    billingCadence: 'P1M',
    price: { kind: 'flat', amount: '' },
  }
}

export function defaultPhase(): PhaseFormValues {
  return { key: '', name: '', duration: '', rateCards: [defaultRateCard()] }
}

export const EMPTY_PLAN: PlanWizardValues = {
  name: '',
  key: '',
  description: '',
  currency: 'CNY',
  billingCadence: 'P1M',
  phases: [defaultPhase()],
}

function toPriceInput(price: PriceFormValue): RateCardInput['price'] {
  switch (price.kind) {
    case 'free':
      return { type: 'free' }
    case 'flat':
      return { type: 'flat', amount: price.amount }
  }
}

function toRateCardInput(card: RateCardFormValues): RateCardInput {
  return {
    key: card.key.trim(),
    name: card.name.trim(),
    // null=一次性：v3 上与省略等价（*ISO8601Duration 的 nil），发 undefined。
    billingCadence: card.billingCadence ?? undefined,
    price: toPriceInput(card.price),
  }
}

/** phases 的映射在 #9（PUT 编辑）中原样复用，故独立导出。 */
export function toPlanPhases(values: PlanWizardValues): PlanPhaseInput[] {
  return values.phases.map((phase) => ({
    key: phase.key.trim(),
    name: phase.name.trim(),
    duration: phase.duration.trim() || undefined,
    rateCards: phase.rateCards.map(toRateCardInput),
  }))
}

export function toCreatePlanRequest(
  values: PlanWizardValues
): CreatePlanRequestInput {
  return {
    name: values.name.trim(),
    description: values.description?.trim() || undefined,
    key: values.key.trim(),
    currency: values.currency.trim(),
    billingCadence: values.billingCadence,
    phases: toPlanPhases(values),
  }
}
