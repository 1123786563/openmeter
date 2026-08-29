import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { PlanAddon } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useAddons,
  useCreatePlanAddon,
  usePlan,
  useUpdatePlanAddon,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

const POSITIVE_INT = /^[1-9]\d*$/
/** Stable empty fallback so the reset effect's deps never churn. */
const EMPTY_PHASES: { key: string; name: string }[] = []

const planAddonFormSchema = z.object({
  name: z.string().min(1).max(256),
  addonId: z.string().min(1),
  fromPlanPhase: z.string().min(1),
  maxQuantity: z
    .string()
    .refine((value) => value === '' || POSITIVE_INT.test(value), 'invalid'),
  description: z.string().max(1024).optional(),
})

type PlanAddonFormValues = z.infer<typeof planAddonFormSchema>

type PlanAddonFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  planId: string
  /** Present when editing an existing association (addon is immutable). */
  planAddon?: PlanAddon
}

export function PlanAddonFormDialog({
  open,
  onOpenChange,
  planId,
  planAddon,
}: PlanAddonFormDialogProps) {
  const { t } = useTranslation()
  const isCreate = !planAddon
  const { data: plan } = usePlan(planId)
  const { data: addonsData } = useAddons()
  const createMutation = useCreatePlanAddon(planId)
  const updateMutation = useUpdatePlanAddon(planId)

  // Only active addons are purchasable within a plan.
  const activeAddons = (addonsData?.data ?? []).filter(
    (addon) => addon.status === 'active'
  )
  const addonById = new Map(activeAddons.map((addon) => [addon.id, addon]))
  const phases = plan?.phases ?? EMPTY_PHASES

  const form = useForm<PlanAddonFormValues>({
    resolver: zodResolver(planAddonFormSchema),
    defaultValues: {
      name: '',
      addonId: '',
      fromPlanPhase: '',
      maxQuantity: '',
      description: '',
    },
  })
  const selectedAddonId = form.watch('addonId')
  const selectedAddon = addonById.get(selectedAddonId)
  const isMultiple = selectedAddon?.instanceType === 'multiple'

  useEffect(() => {
    if (!open) return
    if (planAddon) {
      form.reset({
        name: planAddon.name,
        addonId: planAddon.addon.id,
        fromPlanPhase: planAddon.fromPlanPhase,
        maxQuantity: planAddon.maxQuantity?.toString() ?? '',
        description: planAddon.description ?? '',
      })
    } else {
      form.reset({
        name: '',
        addonId: '',
        fromPlanPhase: phases[0]?.key ?? '',
        maxQuantity: '',
        description: '',
      })
    }
  }, [open, planAddon, form, phases])

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  const onSubmit = (values: PlanAddonFormValues) => {
    // max_quantity is only valid for multi-instance addons (spec).
    const maxQuantity =
      isMultiple && values.maxQuantity ? Number(values.maxQuantity) : undefined
    if (isCreate) {
      createMutation.mutate(
        {
          body: {
            name: values.name.trim(),
            addon: { id: values.addonId },
            fromPlanPhase: values.fromPlanPhase,
            description: values.description?.trim() || undefined,
            ...(maxQuantity !== undefined ? { maxQuantity } : {}),
          },
        },
        {
          onSuccess: () => {
            toast.success(t('config.planAddons.toast.created'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    } else if (planAddon) {
      updateMutation.mutate(
        {
          planAddonId: planAddon.id,
          body: {
            name: values.name.trim(),
            fromPlanPhase: values.fromPlanPhase,
            description: values.description?.trim() || undefined,
            ...(maxQuantity !== undefined ? { maxQuantity } : {}),
          },
        },
        {
          onSuccess: () => {
            toast.success(t('config.planAddons.toast.updated'))
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
              ? t('config.planAddons.form.createTitle')
              : t('config.planAddons.form.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {isCreate
              ? t('config.planAddons.form.createDescription')
              : t('config.planAddons.form.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='addonId'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.planAddons.fields.addon')}</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={(value) => {
                      field.onChange(value)
                      if (isCreate) {
                        form.setValue('name', addonById.get(value)?.name ?? '')
                      }
                    }}
                    disabled={!isCreate}
                  >
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue
                          placeholder={t('config.planAddons.form.selectAddon')}
                        />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {activeAddons.map((addon) => (
                        <SelectItem key={addon.id} value={addon.id}>
                          {addon.name} ({addon.key})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {!isCreate && (
                    <FormDescription>
                      {t('config.planAddons.form.addonImmutable')}
                    </FormDescription>
                  )}
                  <FormMessage>
                    {form.formState.errors.addonId
                      ? t('config.planAddons.form.validation.addon')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='fromPlanPhase'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.planAddons.fields.fromPlanPhase')}
                  </FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className='w-full'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {phases.map((phase) => (
                        <SelectItem key={phase.key} value={phase.key}>
                          {phase.name} ({phase.key})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.planAddons.fields.name')}</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('config.planAddons.form.validation.name')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='maxQuantity'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.planAddons.fields.maxQuantity')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      inputMode='numeric'
                      placeholder='10'
                      disabled={!isMultiple}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {isMultiple
                      ? t('config.planAddons.form.maxQuantityHint')
                      : t('config.planAddons.form.maxQuantitySingle')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.maxQuantity
                      ? t('config.planAddons.form.validation.maxQuantity')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='description'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.planAddons.fields.description')}
                  </FormLabel>
                  <FormControl>
                    <Textarea rows={2} {...field} />
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
