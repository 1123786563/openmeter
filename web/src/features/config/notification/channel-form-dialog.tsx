import { useEffect } from 'react'
import { z } from 'zod'
import { useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateChannel } from '@/api/hooks'
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
import { Switch } from '@/components/ui/switch'

/** Spec: ^(whsec_)?[a-zA-Z0-9+/=]{32,100}$ (base64, optional whsec_ prefix). */
const SIGNING_SECRET_PATTERN = /^(whsec_)?[a-zA-Z0-9+/=]{32,100}$/

/**
 * Header rows are free-form; rows with an empty key are dropped on submit and
 * only non-empty keys are checked for duplicates, so users can leave a blank
 * trailing row without being blocked.
 */
const channelFormSchema = z.object({
  name: z.string().min(1).max(256),
  url: z
    .string()
    .trim()
    .min(1)
    .url()
    .refine((value) => value.startsWith('https://'), 'https'),
  signingSecret: z
    .string()
    .trim()
    .refine(
      (value) => value === '' || SIGNING_SECRET_PATTERN.test(value),
      'format'
    ),
  disabled: z.boolean(),
  customHeaders: z.array(
    z.object({
      key: z.string().max(256),
      value: z.string().max(1024),
    })
  ),
})

type ChannelFormValues = z.infer<typeof channelFormSchema>

const EMPTY_VALUES: ChannelFormValues = {
  name: '',
  url: '',
  signingSecret: '',
  disabled: false,
  customHeaders: [],
}

type ChannelFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ChannelFormDialog({
  open,
  onOpenChange,
}: ChannelFormDialogProps) {
  const { t } = useTranslation()
  const createMutation = useCreateChannel()

  const form = useForm<ChannelFormValues>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: EMPTY_VALUES,
  })

  const headerRows = useFieldArray({
    control: form.control,
    name: 'customHeaders',
  })

  useEffect(() => {
    if (open) form.reset(EMPTY_VALUES)
  }, [open, form])

  const onSubmit = (values: ChannelFormValues) => {
    const customHeaders = Object.fromEntries(
      values.customHeaders
        .filter((row) => row.key.trim() !== '')
        .map((row) => [row.key.trim(), row.value])
    )
    const hasHeaders = Object.keys(customHeaders).length > 0

    createMutation.mutate(
      {
        type: 'WEBHOOK',
        name: values.name.trim(),
        url: values.url.trim(),
        disabled: values.disabled,
        ...(hasHeaders ? { customHeaders } : {}),
        ...(values.signingSecret
          ? { signingSecret: values.signingSecret }
          : {}),
      },
      {
        onSuccess: () => {
          toast.success(t('config.notification.channels.toast.created'))
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[85vh] overflow-y-auto sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {t('config.notification.channels.form.createTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('config.notification.channels.form.createDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.notification.channels.fields.name')}
                  </FormLabel>
                  <FormControl>
                    <Input placeholder='customer-webhook' {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t(
                          'config.notification.channels.form.validation.required'
                        )
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='url'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.notification.channels.fields.url')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder='https://example.com/webhook'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('config.notification.channels.form.urlHint')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.url
                      ? form.formState.errors.url.message === 'https'
                        ? t(
                            'config.notification.channels.form.validation.https'
                          )
                        : t('config.notification.channels.form.validation.url')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='signingSecret'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('config.notification.channels.fields.signingSecret')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder='whsec_...'
                      autoComplete='off'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('config.notification.channels.form.signingSecretHint')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.signingSecret
                      ? t(
                          'config.notification.channels.form.validation.signingSecret'
                        )
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <div className='space-y-2'>
              <FormLabel>
                {t('config.notification.channels.fields.customHeaders')}
              </FormLabel>
              <FormDescription>
                {t('config.notification.channels.form.customHeadersHint')}
              </FormDescription>
              {headerRows.fields.map((row, index) => (
                <div key={row.id} className='space-y-1'>
                  <div className='flex items-start gap-2'>
                    <FormField
                      control={form.control}
                      name={`customHeaders.${index}.key`}
                      render={({ field }) => (
                        <FormItem className='flex-1'>
                          <FormControl>
                            <Input placeholder='X-Custom-Header' {...field} />
                          </FormControl>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`customHeaders.${index}.value`}
                      render={({ field }) => (
                        <FormItem className='flex-1'>
                          <FormControl>
                            <Input placeholder='value' {...field} />
                          </FormControl>
                        </FormItem>
                      )}
                    />
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      className='size-9 shrink-0'
                      onClick={() => headerRows.remove(index)}
                    >
                      <X className='size-4' />
                      <span className='sr-only'>
                        {t('config.notification.channels.form.removeHeader')}
                      </span>
                    </Button>
                  </div>
                  {form.formState.errors.customHeaders?.[index]?.key && (
                    <p className='text-sm text-destructive'>
                      {t(
                        'config.notification.channels.form.validation.headerKey'
                      )}
                    </p>
                  )}
                </div>
              ))}
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={() => headerRows.append({ key: '', value: '' })}
              >
                <Plus className='size-4' />
                {t('config.notification.channels.form.addHeader')}
              </Button>
            </div>
            <FormField
              control={form.control}
              name='disabled'
              render={({ field }) => (
                <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                  <div className='space-y-0.5'>
                    <FormLabel>
                      {t('config.notification.channels.fields.disabled')}
                    </FormLabel>
                    <FormDescription>
                      {t('config.notification.channels.form.disabledHint')}
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
              <Button type='submit' disabled={createMutation.isPending}>
                {createMutation.isPending
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
