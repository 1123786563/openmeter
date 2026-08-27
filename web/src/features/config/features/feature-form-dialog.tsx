import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateFeature } from '@/api/hooks'
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
import { Textarea } from '@/components/ui/textarea'

/** ResourceKey pattern from the v3 spec: ^[a-z0-9]+(?:_[a-z0-9]+)*$ */
const FEATURE_KEY = /^[a-z0-9]+(?:_[a-z0-9]+)*$/

type FeatureFormValues = {
  name: string
  key: string
  description?: string
}

const EMPTY_VALUES: FeatureFormValues = {
  name: '',
  key: '',
  description: '',
}

type FeatureFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function FeatureFormDialog({
  open,
  onOpenChange,
}: FeatureFormDialogProps) {
  const { t } = useTranslation()
  const createMutation = useCreateFeature()

  // shadcn's FormMessage renders error.message, so the validation copy must
  // live on the zod checks themselves; rebuilding per language keeps it in
  // sync with the active locale.
  const featureFormSchema = useMemo(
    () =>
      z.object({
        name: z
          .string()
          .min(1, t('config.features.form.validation.name'))
          .max(256, t('config.features.form.validation.name')),
        key: z
          .string()
          .min(1, t('config.features.form.validation.key'))
          .max(64, t('config.features.form.validation.key'))
          .regex(FEATURE_KEY, t('config.features.form.validation.key')),
        description: z.string().max(1024).optional(),
      }),
    [t]
  )

  const form = useForm<FeatureFormValues>({
    resolver: zodResolver(featureFormSchema),
    defaultValues: EMPTY_VALUES,
  })

  useEffect(() => {
    if (open) form.reset(EMPTY_VALUES)
  }, [open, form])

  const onSubmit = (values: FeatureFormValues) => {
    // The v3 SDK flattens CreateFeatureRequest fields at the top level
    // (no `body` wrapper), unlike update which takes { featureId, body }.
    createMutation.mutate(
      {
        name: values.name.trim(),
        key: values.key.trim(),
        description: values.description?.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t('config.features.toast.created'))
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('config.features.form.createTitle')}</DialogTitle>
          <DialogDescription>
            {t('config.features.form.createDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('config.features.fields.name')}</FormLabel>
                  <FormControl>
                    <Input placeholder='Token API' {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('config.features.form.validation.name')
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
                  <FormLabel>{t('config.features.fields.key')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder='token_api'
                      autoComplete='off'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('config.features.form.keyHint')}
                  </FormDescription>
                  <FormMessage>
                    {form.formState.errors.key
                      ? t('config.features.form.validation.key')
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
                    {t('config.features.fields.description')}
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
