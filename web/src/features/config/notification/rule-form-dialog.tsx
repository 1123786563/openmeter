import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import {
  type Control,
  type FieldErrors,
  useFieldArray,
  useForm,
} from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useCreateRule,
  useFeatures,
  useNotificationChannels,
  useUpdateRule,
} from '@/api/hooks'
import type {
  NotificationRule,
  NotificationRuleCreateRequest,
} from '@/api/legacy'
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
import { MultiSelect } from '@/components/multi-select'

const RULE_TYPES = [
  'entitlements.balance.threshold',
  'entitlements.reset',
  'invoice.created',
  'invoice.updated',
] as const
export type RuleFormType = (typeof RULE_TYPES)[number]

/** Spec: NotificationRuleBalanceThresholdValueType (PERCENT/NUMBER deprecated). */
const THRESHOLD_TYPES = [
  'PERCENT',
  'NUMBER',
  'balance_value',
  'usage_percentage',
  'usage_value',
] as const

const NUMERIC_PATTERN = /^-?\d+(\.\d+)?$/

const ruleFormBase = {
  name: z.string().min(1).max(256),
  channels: z.array(z.string()).min(1),
  disabled: z.boolean(),
}

const thresholdRowSchema = z.object({
  value: z
    .string()
    .trim()
    .refine((value) => NUMERIC_PATTERN.test(value), 'number'),
  type: z.enum(THRESHOLD_TYPES),
})

const ruleFormSchema = z.discriminatedUnion('type', [
  z.object({
    type: z.literal('entitlements.balance.threshold'),
    ...ruleFormBase,
    thresholds: z.array(thresholdRowSchema).min(1).max(10),
    features: z.array(z.string()),
  }),
  z.object({
    type: z.literal('entitlements.reset'),
    ...ruleFormBase,
    features: z.array(z.string()),
  }),
  z.object({ type: z.literal('invoice.created'), ...ruleFormBase }),
  z.object({ type: z.literal('invoice.updated'), ...ruleFormBase }),
])

type RuleFormValues = z.infer<typeof ruleFormSchema>
type ThresholdFormValues = Extract<
  RuleFormValues,
  { type: 'entitlements.balance.threshold' }
>
type FeaturesAwareFormValues = Extract<
  RuleFormValues,
  { type: 'entitlements.balance.threshold' | 'entitlements.reset' }
>

const CREATE_DEFAULT: RuleFormValues = {
  type: 'invoice.created',
  name: '',
  channels: [],
  disabled: false,
}

type RuleFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Present when editing an existing rule of any type. */
  rule?: NotificationRule
}

export function RuleFormDialog({
  open,
  onOpenChange,
  rule,
}: RuleFormDialogProps) {
  const { t } = useTranslation()
  const isCreate = !rule

  const createMutation = useCreateRule()
  const updateMutation = useUpdateRule()

  const { data: channelsData } = useNotificationChannels({
    page: 1,
    pageSize: 100,
  })
  const channelOptions = useMemo(
    () =>
      (channelsData?.items ?? []).map((channel) => ({
        value: channel.id,
        label: channel.name,
      })),
    [channelsData]
  )

  const { data: featuresData } = useFeatures({ page: 1, pageSize: 100 })
  const featureOptions = useMemo(
    () =>
      (featuresData?.data ?? []).map((feature) => ({
        value: feature.id,
        label: feature.name || feature.key,
      })),
    [featuresData]
  )

  const form = useForm<RuleFormValues>({
    resolver: zodResolver(ruleFormSchema),
    defaultValues: CREATE_DEFAULT,
  })

  // `thresholds` only exists on the threshold branch; the array control is
  // bound to that branch and only rendered while it is active.
  const thresholdRows = useFieldArray({
    control: form.control as unknown as Control<ThresholdFormValues>,
    name: 'thresholds',
  })

  const watchedType = form.watch('type')

  useEffect(() => {
    if (!open) return
    form.reset(
      rule
        ? {
            name: rule.name,
            channels: rule.channels.map((channel) => channel.id),
            disabled: rule.disabled,
            ...(rule.type === 'entitlements.balance.threshold'
              ? {
                  type: rule.type,
                  thresholds: rule.thresholds.map((threshold) => ({
                    value: String(threshold.value),
                    type: threshold.type,
                  })),
                  features: (rule.features ?? []).map((feature) => feature.id),
                }
              : rule.type === 'entitlements.reset'
                ? {
                    type: rule.type,
                    features: (rule.features ?? []).map(
                      (feature) => feature.id
                    ),
                  }
                : { type: rule.type }),
          }
        : CREATE_DEFAULT
    )
  }, [open, rule, form])

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  /** Type switch must migrate the union shape (add/drop type-specific keys). */
  const switchType = (next: RuleFormType) => {
    const current = form.getValues()
    const common = {
      name: current.name,
      channels: current.channels,
      disabled: current.disabled,
    }
    const currentFeatures =
      current.type === 'entitlements.balance.threshold' ||
      current.type === 'entitlements.reset'
        ? current.features
        : []
    if (next === 'entitlements.balance.threshold') {
      form.reset({
        ...common,
        type: next,
        thresholds:
          current.type === 'entitlements.balance.threshold'
            ? current.thresholds
            : [{ value: '100', type: 'usage_value' }],
        features: currentFeatures,
      })
    } else if (next === 'entitlements.reset') {
      form.reset({ ...common, type: next, features: currentFeatures })
    } else {
      form.reset({ ...common, type: next })
    }
  }

  const onSubmit = (values: RuleFormValues) => {
    const common = {
      name: values.name.trim(),
      disabled: values.disabled,
      channels: values.channels,
    }
    // An empty features array is invalid per spec (minItems 1); omitting the
    // field entirely means "applies to all features".
    let body: NotificationRuleCreateRequest
    switch (values.type) {
      case 'entitlements.balance.threshold':
        body = {
          ...common,
          type: values.type,
          thresholds: values.thresholds.map((row) => ({
            value: Number(row.value),
            type: row.type,
          })),
          ...(values.features.length ? { features: values.features } : {}),
        }
        break
      case 'entitlements.reset':
        body = {
          ...common,
          type: values.type,
          ...(values.features.length ? { features: values.features } : {}),
        }
        break
      case 'invoice.created':
      case 'invoice.updated':
        body = { ...common, type: values.type }
        break
    }

    if (isCreate) {
      createMutation.mutate(body, {
        onSuccess: () => {
          toast.success(t('config.notification.rules.toast.created'))
          onOpenChange(false)
        },
        onError: handleServerError,
      })
    } else if (rule) {
      updateMutation.mutate(
        { ruleId: rule.id, body },
        {
          onSuccess: () => {
            toast.success(t('config.notification.rules.toast.updated'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[85vh] overflow-y-auto sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {isCreate
              ? t('config.notification.rules.form.createTitle')
              : t('config.notification.rules.form.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {isCreate
              ? t('config.notification.rules.form.createDescription')
              : t('config.notification.rules.form.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='type'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.notification.rules.fields.type')}
                  </FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={(value) => switchType(value as RuleFormType)}
                    disabled={!isCreate}
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {RULE_TYPES.map((type) => (
                        <SelectItem key={type} value={type}>
                          {t(`config.notification.rules.types.${type}`)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t('config.notification.rules.form.typeHint')}
                  </FormDescription>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.notification.rules.fields.name')}
                  </FormLabel>
                  <FormControl>
                    <Input placeholder='Balance threshold reached' {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('config.notification.rules.form.validation.required')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='channels'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.notification.rules.fields.channels')}
                  </FormLabel>
                  <FormControl>
                    <MultiSelect
                      options={channelOptions}
                      value={field.value}
                      onChange={field.onChange}
                      placeholder={t(
                        'config.notification.rules.form.channelsPlaceholder'
                      )}
                      searchPlaceholder={t('common.search')}
                      emptyText={t('config.notification.rules.form.noChannels')}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('config.notification.rules.form.channelsHint')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.channels
                      ? t('config.notification.rules.form.validation.channels')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            {watchedType === 'entitlements.balance.threshold' && (
              <div className='space-y-2'>
                <FormLabel>
                  {t('config.notification.rules.fields.thresholds')}
                </FormLabel>
                <FormDescription>
                  {t('config.notification.rules.form.thresholdsHint')}
                </FormDescription>
                {thresholdRows.fields.map((row, index) => (
                  <div key={row.id} className='space-y-1'>
                    <div className='flex items-start gap-2'>
                      <FormField
                        control={
                          form.control as unknown as Control<ThresholdFormValues>
                        }
                        name={`thresholds.${index}.value`}
                        render={({ field }) => (
                          <FormItem className='flex-1'>
                            <FormControl>
                              <Input
                                inputMode='decimal'
                                placeholder='100'
                                {...field}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={
                          form.control as unknown as Control<ThresholdFormValues>
                        }
                        name={`thresholds.${index}.type`}
                        render={({ field }) => (
                          <FormItem className='flex-1'>
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
                                {THRESHOLD_TYPES.map((type) => (
                                  <SelectItem key={type} value={type}>
                                    {t(
                                      `config.notification.rules.thresholdTypes.${type}`
                                    )}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </FormItem>
                        )}
                      />
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon'
                        className='size-9 shrink-0'
                        disabled={thresholdRows.fields.length <= 1}
                        onClick={() => thresholdRows.remove(index)}
                      >
                        <Trash2 className='size-4' />
                        <span className='sr-only'>
                          {t('config.notification.rules.form.removeThreshold')}
                        </span>
                      </Button>
                    </div>
                    {(form.formState.errors as FieldErrors<ThresholdFormValues>)
                      .thresholds?.[index]?.value && (
                      <p className='text-sm text-destructive'>
                        {t(
                          'config.notification.rules.form.validation.threshold'
                        )}
                      </p>
                    )}
                  </div>
                ))}
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={thresholdRows.fields.length >= 10}
                  onClick={() =>
                    thresholdRows.append({ value: '', type: 'usage_value' })
                  }
                >
                  <Plus className='size-4' />
                  {t('config.notification.rules.form.addThreshold')}
                </Button>
              </div>
            )}
            {(watchedType === 'entitlements.balance.threshold' ||
              watchedType === 'entitlements.reset') && (
              <FormField
                control={
                  form.control as unknown as Control<FeaturesAwareFormValues>
                }
                name='features'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.notification.rules.fields.features')}
                    </FormLabel>
                    <FormControl>
                      <MultiSelect
                        options={featureOptions}
                        value={field.value}
                        onChange={field.onChange}
                        placeholder={t(
                          'config.notification.rules.form.featuresPlaceholder'
                        )}
                        searchPlaceholder={t('common.search')}
                        emptyText={t(
                          'config.notification.rules.form.noFeatures'
                        )}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('config.notification.rules.form.featuresHint')}
                    </FormDescription>
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name='disabled'
              render={({ field }) => (
                <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                  <div className='space-y-0.5'>
                    <FormLabel>
                      {t('config.notification.rules.fields.disabled')}
                    </FormLabel>
                    <FormDescription>
                      {t('config.notification.rules.form.disabledHint')}
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
