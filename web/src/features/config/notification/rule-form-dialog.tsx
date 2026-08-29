import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useCreateRule,
  useNotificationChannels,
  useUpdateRule,
} from '@/api/hooks'
import type {
  NotificationRuleCreateRequest,
  NotificationRuleInvoiceCreated,
  NotificationRuleInvoiceUpdated,
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

/**
 * Rule types editable in this dialog. The API also defines
 * `entitlements.balance.threshold` and `entitlements.reset`; their editors
 * (threshold rows / feature multi-select) land in a follow-up change and the
 * union below is extended there.
 */
const RULE_TYPES = ['invoice.created', 'invoice.updated'] as const
export type RuleFormType = (typeof RULE_TYPES)[number]

const ruleFormBase = {
  name: z.string().min(1).max(256),
  channels: z.array(z.string()).min(1),
  disabled: z.boolean(),
}

const ruleFormSchema = z.discriminatedUnion('type', [
  z.object({ type: z.literal('invoice.created'), ...ruleFormBase }),
  z.object({ type: z.literal('invoice.updated'), ...ruleFormBase }),
])

type RuleFormValues = z.infer<typeof ruleFormSchema>

const CREATE_DEFAULT: RuleFormValues = {
  type: 'invoice.created',
  name: '',
  channels: [],
  disabled: false,
}

type EditableRule =
  | NotificationRuleInvoiceCreated
  | NotificationRuleInvoiceUpdated

type RuleFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Present when editing an existing invoice-type rule. */
  rule?: EditableRule
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

  const form = useForm<RuleFormValues>({
    resolver: zodResolver(ruleFormSchema),
    defaultValues: CREATE_DEFAULT,
  })

  useEffect(() => {
    if (!open) return
    form.reset(
      rule
        ? {
            type: rule.type,
            name: rule.name,
            channels: rule.channels.map((channel) => channel.id),
            disabled: rule.disabled,
          }
        : CREATE_DEFAULT
    )
  }, [open, rule, form])

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  const onSubmit = (values: RuleFormValues) => {
    const body: NotificationRuleCreateRequest = {
      type: values.type,
      name: values.name.trim(),
      disabled: values.disabled,
      channels: values.channels,
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
                    onValueChange={field.onChange}
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
                    <Input placeholder='Invoice created' {...field} />
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
