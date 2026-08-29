import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { AppCatalogItem } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useInstallApp } from '@/api/hooks'
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
import { PasswordInput } from '@/components/password-input'

/**
 * The API key is required only for Stripe installs (issue acceptance item),
 * so the schema is built per catalog item type; the caller remounts this
 * dialog with a per-type key to keep the resolver in sync.
 */
function createInstallSchema(isStripe: boolean) {
  return z
    .object({
      name: z.string().trim().min(1),
      apiKey: z.string().trim(),
      createBillingProfile: z.boolean(),
    })
    .superRefine((values, ctx) => {
      if (isStripe && values.apiKey.length === 0) {
        ctx.addIssue({
          code: 'custom',
          path: ['apiKey'],
          message: 'config.apps.install.validation.apiKeyRequired',
        })
      }
    })
}

type InstallAppFormValues = z.infer<ReturnType<typeof createInstallSchema>>

type InstallAppDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  item: AppCatalogItem | null
}

/**
 * Install a catalog app. The SDK install contract requires
 * createBillingProfile on every branch, but the switch is only exposed for
 * Stripe; sandbox and external_invoicing always submit false.
 */
export function InstallAppDialog({
  open,
  onOpenChange,
  item,
}: InstallAppDialogProps) {
  const { t } = useTranslation()
  const installMutation = useInstallApp()

  const isStripe = item?.type === 'stripe'

  const form = useForm<InstallAppFormValues>({
    resolver: zodResolver(createInstallSchema(isStripe)),
    defaultValues: {
      name: item?.name ?? '',
      apiKey: '',
      createBillingProfile: true,
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        name: item?.name ?? '',
        apiKey: '',
        createBillingProfile: true,
      })
    }
  }, [open, item, form])

  const onSubmit = (values: InstallAppFormValues) => {
    if (!item) return
    installMutation.mutate(
      item.type === 'stripe'
        ? {
            type: 'stripe',
            name: values.name,
            createBillingProfile: values.createBillingProfile,
            apiKey: values.apiKey,
          }
        : item.type === 'sandbox'
          ? { type: 'sandbox', name: values.name, createBillingProfile: false }
          : {
              type: 'external_invoicing',
              name: values.name,
              createBillingProfile: false,
            },
      {
        onSuccess: () => {
          toast.success(t('config.apps.install.toast.installed'))
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
            {t('config.apps.install.title', { name: item?.name ?? '' })}
          </DialogTitle>
          <DialogDescription>
            {t('config.apps.install.description')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.apps.install.name')}</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('config.apps.install.validation.nameRequired')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            {isStripe && (
              <>
                <FormField
                  control={form.control}
                  name='apiKey'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('config.apps.install.apiKey')}</FormLabel>
                      <FormControl>
                        <PasswordInput
                          placeholder='sk_live_...'
                          autoComplete='new-password'
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('config.apps.install.apiKeyHint')}
                      </FormDescription>
                      <FormMessage>
                        {form.formState.errors.apiKey
                          ? t('config.apps.install.validation.apiKeyRequired')
                          : undefined}
                      </FormMessage>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='createBillingProfile'
                  render={({ field }) => (
                    <FormItem>
                      <div className='flex items-center justify-between gap-4'>
                        <div className='space-y-0.5'>
                          <FormLabel>
                            {t('config.apps.install.createBillingProfile')}
                          </FormLabel>
                          <FormDescription>
                            {t('config.apps.install.createBillingProfileHint')}
                          </FormDescription>
                        </div>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </div>
                    </FormItem>
                  )}
                />
              </>
            )}
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => onOpenChange(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type='submit' disabled={installMutation.isPending}>
                {installMutation.isPending
                  ? t('common.submitting')
                  : t('config.apps.install.submit')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
