import { z } from 'zod'
import type {
  CreatePlanRequestInput,
  Plan,
  PlanPhaseInput,
  Price,
  RateCard,
  RateCardInput,
  UpsertPlanRequestInput,
} from '@openmeter/client'

const RESOURCE_KEY = /^[a-z0-9]+(?:_[a-z0-9]+)*$/
const ISO8601_DURATION =
  /^P(?:\d+(?:\.\d+)?Y)?(?:\d+(?:\.\d+)?M)?(?:\d+(?:\.\d+)?W)?(?:\d+(?:\.\d+)?D)?(?:T(?:\d+(?:\.\d+)?H)?(?:\d+(?:\.\d+)?M)?(?:\d+(?:\.\d+)?S)?)?$/
const AMOUNT = /^\d+(\.\d+)?$/
const NON_NEGATIVE_INT = /^\d+$/
const CURRENCY = /^[A-Z]{3}$|^[A-Za-z0-9]{4,24}$/

/**
 * PriceForm 契约（本文件是唯一定义处）：
 * - #6 本任务：free | flat
 * - #7 扩展：+ { kind:'unit', amount }
 * - #8 扩展：+ { kind:'tiered', mode, tiers }
 * zod message 一律是 i18n key，由 FieldError 组件翻译。
 */
export const tierSchema = z.object({
  firstUnit: z
    .string()
    .refine(
      (value) => NON_NEGATIVE_INT.test(value),
      'config.plans.wizard.errors.tierBound'
    ),
  // '' = 无上限（仅末档允许，由 rateCardSchema.superRefine 强制）。
  lastUnit: z
    .string()
    .refine(
      (value) => value === '' || NON_NEGATIVE_INT.test(value),
      'config.plans.wizard.errors.tierBound'
    ),
  unitAmount: z
    .string()
    .refine((value) => AMOUNT.test(value), 'config.plans.wizard.errors.amount'),
  // '' = 该档不收固定费（映射时省略 flat_price）。
  flatAmount: z
    .string()
    .refine(
      (value) => value === '' || AMOUNT.test(value),
      'config.plans.wizard.errors.amount'
    ),
})

export type TierFormValues = z.infer<typeof tierSchema>

export const priceFormSchema = z.discriminatedUnion('kind', [
  z.object({ kind: z.literal('free') }),
  z.object({
    kind: z.literal('flat'),
    amount: z
      .string()
      .refine(
        (value) => AMOUNT.test(value),
        'config.plans.wizard.errors.amount'
      ),
  }),
  z.object({
    kind: z.literal('unit'),
    amount: z
      .string()
      .refine(
        (value) => AMOUNT.test(value),
        'config.plans.wizard.errors.amount'
      ),
  }),
  z.object({
    kind: z.literal('tiered'),
    mode: z.enum(['graduated', 'volume']),
    tiers: z
      .array(tierSchema)
      .min(1, 'config.plans.wizard.errors.tiersRequired'),
  }),
])

export type PriceFormValue = z.infer<typeof priceFormSchema>

export const rateCardSchema = z
  .object({
    key: z
      .string()
      .min(1, 'config.plans.wizard.errors.required')
      .max(64)
      .refine(
        (value) => RESOURCE_KEY.test(value),
        'config.plans.wizard.errors.keyFormat'
      ),
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
    if (card.type === 'usage_based') {
      if (!card.featureId || card.featureId.trim() === '') {
        ctx.addIssue({
          code: 'custom',
          path: ['featureId'],
          message: 'config.plans.wizard.errors.featureRequired',
        })
      }
      if (card.price.kind !== 'unit' && card.price.kind !== 'tiered') {
        ctx.addIssue({
          code: 'custom',
          path: ['type'],
          message: 'config.plans.wizard.errors.usagePriceKind',
        })
      }
    }
    if (
      card.billingCadence === null &&
      card.price.kind !== 'free' &&
      card.price.kind !== 'flat'
    ) {
      ctx.addIssue({
        code: 'custom',
        path: ['billingCadence'],
        message: 'config.plans.wizard.errors.oneTimeFlatOnly',
      })
    }
    if (card.price.kind === 'tiered') {
      const { tiers } = card.price
      // 首档必须从 0 起。
      if (tiers.length > 0 && Number(tiers[0].firstUnit) !== 0) {
        ctx.addIssue({
          code: 'custom',
          path: ['price', 'tiers', 0, 'firstUnit'],
          message: 'config.plans.wizard.errors.tierFirstFromZero',
        })
      }
      tiers.forEach((tier, index) => {
        const isLast = index === tiers.length - 1
        // 末档必须开区间（lastUnit 空）。
        if (isLast && tier.lastUnit !== '') {
          ctx.addIssue({
            code: 'custom',
            path: ['price', 'tiers', index, 'lastUnit'],
            message: 'config.plans.wizard.errors.tierLastOpen',
          })
        }
        // 非末档必须闭合。
        if (!isLast && tier.lastUnit === '') {
          ctx.addIssue({
            code: 'custom',
            path: ['price', 'tiers', index, 'lastUnit'],
            message: 'config.plans.wizard.errors.tierLastRequired',
          })
        }
        if (
          tier.lastUnit !== '' &&
          Number(tier.lastUnit) < Number(tier.firstUnit)
        ) {
          ctx.addIssue({
            code: 'custom',
            path: ['price', 'tiers', index, 'lastUnit'],
            message: 'config.plans.wizard.errors.tierRange',
          })
        }
        // 区间连续无重叠：lastUnit + 1 必须等于下档 firstUnit，否则在下档 firstUnit 报缺口/重叠。
        const next = tiers[index + 1]
        if (
          next &&
          tier.lastUnit !== '' &&
          Number(tier.lastUnit) + 1 !== Number(next.firstUnit)
        ) {
          ctx.addIssue({
            code: 'custom',
            path: ['price', 'tiers', index + 1, 'firstUnit'],
            message: 'config.plans.wizard.errors.tierGap',
          })
        }
      })
    }
  })

export type RateCardFormValues = z.infer<typeof rateCardSchema>

export const phaseSchema = z.object({
  key: z
    .string()
    .min(1, 'config.plans.wizard.errors.required')
    .max(64)
    .refine(
      (value) => RESOURCE_KEY.test(value),
      'config.plans.wizard.errors.keyFormat'
    ),
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
    .refine(
      (value) => RESOURCE_KEY.test(value),
      'config.plans.wizard.errors.keyFormat'
    ),
  description: z.string().max(1024).optional(),
  // 法币 3 位大写，或自定义货币代码 4-24 位（BillingCurrencyCode）。
  currency: z
    .string()
    .refine(
      (value) => CURRENCY.test(value),
      'config.plans.wizard.errors.currency'
    ),
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

export function defaultTier(): TierFormValues {
  return { firstUnit: '0', lastUnit: '', unitAmount: '', flatAmount: '' }
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
    case 'unit':
      return { type: 'unit', amount: price.amount }
    case 'tiered':
      return {
        type: price.mode,
        tiers: price.tiers.map((tier) => ({
          upToAmount: tier.lastUnit === '' ? undefined : tier.lastUnit,
          unitPrice: { type: 'unit' as const, amount: tier.unitAmount },
          ...(tier.flatAmount === ''
            ? {}
            : { flatPrice: { type: 'flat' as const, amount: tier.flatAmount } }),
        })),
      }
  }
}

/** 单张价卡 → RateCardInput；plans 向导与 addons 表单（#10）共用。 */
export function toRateCardInput(card: RateCardFormValues): RateCardInput {
  return {
    key: card.key.trim(),
    name: card.name.trim(),
    feature: card.featureId ? { id: card.featureId } : undefined,
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

/** 把单张价卡已选 feature 展开回 featureId（flat_fee 卡为 ''，提交时省略 feature）；addons 编辑（#10）共用。 */
export function fromRateCardToForm(card: RateCard): RateCardFormValues {
  return {
    key: card.key,
    name: card.name,
    type:
      card.price.type === 'free' || card.price.type === 'flat'
        ? 'flat_fee'
        : 'usage_based',
    featureId: card.feature?.id ?? '',
    // 只回填 P1M/P1Y；原卡周期为其他 ISO8601 值（API 直建的少见情形）收敛为 null（一次性），
    // 保存前界面会在该行下拉中显式重选，不会静默改写非既有值。
    billingCadence:
      card.billingCadence === 'P1Y' || card.billingCadence === 'P1M'
        ? card.billingCadence
        : null,
    price: fromPriceToForm(card.price),
  }
}

function fromPriceToForm(price: Price): PriceFormValue {
  switch (price.type) {
    case 'free':
      return { kind: 'free' }
    case 'flat':
      return { kind: 'flat', amount: price.amount }
    case 'unit':
      return { kind: 'unit', amount: price.amount }
    case 'graduated':
    case 'volume': {
      // spec 的区间起点是隐式的（上一档 up_to_amount+1），表单需要显式 firstUnit。
      let previousUpTo = -1
      const tiers = price.tiers.map((tier) => {
        const firstUnit = String(previousUpTo + 1)
        if (tier.upToAmount !== undefined)
          previousUpTo = Number(tier.upToAmount)
        return {
          firstUnit,
          lastUnit:
            tier.upToAmount === undefined ? '' : String(tier.upToAmount),
          unitAmount: tier.unitPrice?.amount ?? '0',
          flatAmount: tier.flatPrice?.amount ?? '',
        }
      })
      return { kind: 'tiered', mode: price.type, tiers }
    }
  }
}

/**
 * v3 GET Plan → 向导初始值。key/currency/billingCadence 不可变（PUT 不提交）：
 * billingCadence 非 P1M/P1Y 时收敛显示为 P1M，不回传服务端，无数据破坏。
 */
export function fromPlanToWizardValues(plan: Plan): PlanWizardValues {
  return {
    name: plan.name,
    key: plan.key,
    description: plan.description ?? '',
    currency: plan.currency,
    billingCadence: plan.billingCadence === 'P1Y' ? 'P1Y' : 'P1M',
    phases: plan.phases.map((phase) => ({
      key: phase.key,
      name: phase.name,
      duration: phase.duration ?? '',
      rateCards: phase.rateCards.map(fromRateCardToForm),
    })),
  }
}

/** PUT /openmeter/plans/{id}：仅 name/description/phases（key/currency/billing_cadence 不可变）。 */
export function toUpsertPlanRequest(
  values: PlanWizardValues
): UpsertPlanRequestInput {
  return {
    name: values.name.trim(),
    description: values.description?.trim() || undefined,
    phases: toPlanPhases(values),
  }
}
