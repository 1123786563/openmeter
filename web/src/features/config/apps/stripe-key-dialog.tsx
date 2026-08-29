import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { AppStripe } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useUpdateApp } from '@/api/hooks'
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
import { PasswordInput } from '@/components/password-input'

const schema = z.object({
  secretApiKey: z.string().trim().min(1),
})

type StripeKeyFormValues = z.infer<typeof schema>

type StripeKeyDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  app: AppStripe | null
}

/**
 * Replace the Stripe secret API key of an installed Stripe app. The PUT body
 * requires name+type and replaces the record, so existing description/labels
 * are echoed back unchanged to avoid clearing them.
 */
export function StripeKeyDialog({
  open,
  onOpenChange,
  app,
}: StripeKeyDialogProps) {
  const { t } = useTranslation()
  const updateMutation = useUpdateApp()

  const form = useForm<StripeKeyFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { secretApiKey: '' },
  })

  useEffect(() => {
    if (open) form.reset({ secretApiKey: '' })
  }, [open, form])

  const onSubmit = (values: StripeKeyFormValues) => {
    if (!app) return
    updateMutation.mutate(
      {
        appId: app.id,
        body: {
          type: 'stripe',
          name: app.name,
          ...(app.description ? { description: app.description } : {}),
          ...(app.labels ? { labels: app.labels } : {}),
          secretApiKey: values.secretApiKey,
        },
      },
      {
        onSuccess: () => {
          toast.success(t('config.apps.stripeKey.toast.updated'))
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>
            {t('config.apps.stripeKey.title', { name: app?.name ?? '' })}
          </DialogTitle>
          <DialogDescription>
            {t('config.apps.stripeKey.description', {
              masked: app?.maskedApiKey ?? '',
            })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='secretApiKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.apps.stripeKey.newKey')}</FormLabel>
                  <FormControl>
                    <PasswordInput
                      placeholder='sk_live_...'
                      autoComplete='new-password'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('config.apps.stripeKey.hint')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.secretApiKey
                      ? t('config.apps.stripeKey.validation.required')
                      : undefined}
                  </FormMessage>
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
              <Button type='submit' disabled={updateMutation.isPending}>
                {updateMutation.isPending
                  ? t('common.submitting')
                  : t('config.apps.stripeKey.confirm')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
