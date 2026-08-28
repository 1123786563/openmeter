import { useEffect } from 'react'
import { z } from 'zod'
import { useForm, useFieldArray } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { TaxCode } from '@openmeter/client'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useCreateTaxCode,
  useUpsertTaxCode,
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

const APP_TYPES = ['sandbox', 'stripe', 'external_invoicing'] as const

/** ResourceKey from the v3 spec: lowercase snake_case, 1-64 chars. */
const RESOURCE_KEY = /^[a-z0-9]+(?:_[a-z0-9]+)*$/

const mappingSchema = z.object({
  appType: z.enum(APP_TYPES),
  taxCode: z.string().trim().min(1),
})

const baseSchema = z.object({
  name: z.string().trim().min(1).max(256),
  description: z.string().trim().max(1024),
  // One mapping per app type: duplicate appType rows are rejected up front.
  appMappings: z
    .array(mappingSchema)
    .refine(
      (rows) => new Set(rows.map((row) => row.appType)).size === rows.length,
      'duplicateAppType'
    ),
})

const createSchema = baseSchema.extend({
  key: z.string().trim().regex(RESOURCE_KEY, 'invalid'),
})

// Edit keeps the exact createSchema shape so the control's generics stay
// unified; the key rules are relaxed because the field renders read-only and
// the upsert body has no key field anyway (same pattern as the recharge
// product dialog's immutable fields).
const editSchema = baseSchema.extend({
  key: z.string(),
})

type TaxCodeFormValues = z.infer<typeof createSchema>

const EMPTY_VALUES: TaxCodeFormValues = {
  name: '',
  key: '',
  description: '',
  appMappings: [],
}

type TaxCodeFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Present when editing an existing tax code (key becomes immutable). */
  taxCode?: TaxCode
}

export function TaxCodeFormDialog({
  open,
  onOpenChange,
  taxCode,
}: TaxCodeFormDialogProps) {
  const { t } = useTranslation()
  const isCreate = !taxCode

  const createMutation = useCreateTaxCode()
  const upsertMutation = useUpsertTaxCode()

  const form = useForm<TaxCodeFormValues>({
    resolver: zodResolver(isCreate ? createSchema : editSchema),
    defaultValues: EMPTY_VALUES,
  })

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'appMappings',
  })

  useEffect(() => {
    if (open) {
      form.reset(
        taxCode
          ? {
              name: taxCode.name,
              key: taxCode.key,
              description: taxCode.description ?? '',
              appMappings: taxCode.appMappings.map((mapping) => ({
                appType: mapping.appType,
                taxCode: mapping.taxCode,
              })),
            }
          : EMPTY_VALUES
      )
    }
  }, [open, taxCode, form])

  const isSubmitting = createMutation.isPending || upsertMutation.isPending

  const onSubmit = (values: TaxCodeFormValues) => {
    const appMappings = values.appMappings.map((mapping) => ({
      appType: mapping.appType,
      taxCode: mapping.taxCode.trim(),
    }))
    if (isCreate) {
      createMutation.mutate(
        {
          name: values.name.trim(),
          key: values.key.trim(),
          description: values.description.trim() || undefined,
          appMappings,
        },
        {
          onSuccess: () => {
            toast.success(t('config.taxCodes.toast.created'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    } else if (taxCode) {
      // UpsertTaxCodeRequest has no key field — keys are immutable.
      upsertMutation.mutate(
        {
          taxCodeId: taxCode.id,
          body: {
            name: values.name.trim(),
            description: values.description.trim() || undefined,
            appMappings,
          },
        },
        {
          onSuccess: () => {
            toast.success(t('config.taxCodes.toast.updated'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>
            {isCreate
              ? t('config.taxCodes.form.createTitle')
              : t('config.taxCodes.form.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {isCreate
              ? t('config.taxCodes.form.createDescription')
              : t('config.taxCodes.form.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('config.taxCodes.fields.name')}</FormLabel>
                    <FormControl>
                      <Input placeholder='Digital Services' {...field} />
                    </FormControl>
                    <FormMessage>
                      {form.formState.errors.name
                        ? t('config.taxCodes.form.validation.required')
                        : undefined}
                    </FormMessage>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='key'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('config.taxCodes.fields.key')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='digital_services'
                        autoComplete='off'
                        disabled={!isCreate}
                        className='font-mono'
                        {...field}
                      />
                    </FormControl>
                    {!isCreate && (
                      <FormDescription>
                        {t('config.taxCodes.form.keyImmutable')}
                      </FormDescription>
                    )}
                    <FormMessage>
                      {form.formState.errors.key
                        ? t('config.taxCodes.form.validation.key')
                        : undefined}
                    </FormMessage>
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
                    {t('config.taxCodes.fields.description')}（
                    {t('common.optional')}）
                  </FormLabel>
                  <FormControl>
                    <Textarea rows={2} {...field} />
                  </FormControl>
                </FormItem>
              )}
            />

            <div className='space-y-2'>
              <div className='flex items-center justify-between'>
                <FormLabel>{t('config.taxCodes.fields.appMappings')}</FormLabel>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  className='h-7 px-2'
                  onClick={() => append({ appType: 'sandbox', taxCode: '' })}
                >
                  <Plus className='size-4' />
                  {t('config.taxCodes.form.addMapping')}
                </Button>
              </div>
              {fields.map((field, index) => (
                <div key={field.id} className='flex items-start gap-2'>
                  <FormField
                    control={form.control}
                    name={`appMappings.${index}.appType` as const}
                    render={({ field: typeField }) => (
                      <FormItem className='w-48'>
                        <Select
                          value={typeField.value}
                          onValueChange={typeField.onChange}
                        >
                          <FormControl>
                            <SelectTrigger className='w-full'>
                              <SelectValue />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {APP_TYPES.map((appType) => (
                              <SelectItem key={appType} value={appType}>
                                {t(`config.taxCodes.appType.${appType}`)}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name={`appMappings.${index}.taxCode` as const}
                    render={({ field: codeField }) => (
                      <FormItem className='flex-1'>
                        <FormControl>
                          <Input
                            placeholder='txcd_10000000'
                            className='font-mono'
                            {...codeField}
                          />
                        </FormControl>
                        <FormMessage>
                          {form.formState.errors.appMappings?.[index]?.taxCode
                            ? t('config.taxCodes.form.validation.required')
                            : undefined}
                        </FormMessage>
                      </FormItem>
                    )}
                  />
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='size-9 text-destructive hover:text-destructive'
                    onClick={() => remove(index)}
                  >
                    <Trash2 className='size-4' />
                    <span className='sr-only'>
                      {t('config.taxCodes.form.removeMapping')}
                    </span>
                  </Button>
                </div>
              ))}
              {form.formState.errors.appMappings?.message ===
                'duplicateAppType' && (
                <p className='text-sm text-destructive'>
                  {t('config.taxCodes.form.validation.duplicateAppType')}
                </p>
              )}
              {fields.length === 0 && (
                <p className='text-sm text-muted-foreground'>
                  {t('config.taxCodes.form.noMappings')}
                </p>
              )}
            </div>

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
