import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { Customer } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateCustomer, useUpdateCustomer } from '@/api/hooks'
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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

/**
 * Shared form shape. `key` is only validated (and sent) when creating; it is
 * immutable after creation. The schema keeps a single stable type so the zod
 * resolver's input and output line up with the form values.
 */
const customerSchema = z.object({
  key: z.string(),
  name: z.string().min(1).max(256),
  primaryEmail: z.string().email().max(256).optional().or(z.literal('')),
  description: z.string().max(1024).optional(),
  currency: z.string().max(8).optional(),
})

type CustomerFormValues = z.infer<typeof customerSchema>

type CustomerFormDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Present when editing an existing customer. */
  customer?: Customer
}

export function CustomerFormDialog({
  open,
  onOpenChange,
  customer,
}: CustomerFormDialogProps) {
  const { t } = useTranslation()
  const isCreate = !customer

  const createMutation = useCreateCustomer()
  const updateMutation = useUpdateCustomer()

  const form = useForm<CustomerFormValues>({
    resolver: zodResolver(customerSchema),
    defaultValues: {
      key: customer?.key ?? '',
      name: customer?.name ?? '',
      primaryEmail: customer?.primaryEmail ?? '',
      description: customer?.description ?? '',
      currency: customer?.currency ?? '',
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        key: customer?.key ?? '',
        name: customer?.name ?? '',
        primaryEmail: customer?.primaryEmail ?? '',
        description: customer?.description ?? '',
        currency: customer?.currency ?? '',
      })
    }
  }, [open, customer, form])

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  const onSubmit = (values: CustomerFormValues) => {
    if (isCreate) {
      const key = values.key.trim()
      if (!/^[a-zA-Z0-9_-]{1,256}$/.test(key)) {
        form.setError('key', {
          message: t('customers.form.validation.keyPattern'),
        })
        return
      }
      createMutation.mutate(
        {
          name: values.name,
          key,
          primaryEmail: values.primaryEmail || undefined,
          description: values.description || undefined,
          currency: values.currency || undefined,
        },
        {
          onSuccess: () => {
            toast.success(t('customers.toast.created'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    } else if (customer) {
      updateMutation.mutate(
        {
          customerId: customer.id,
          body: {
            name: values.name,
            primaryEmail: values.primaryEmail || undefined,
            description: values.description || undefined,
            currency: values.currency || undefined,
          },
        },
        {
          onSuccess: () => {
            toast.success(t('customers.toast.updated'))
            onOpenChange(false)
          },
          onError: handleServerError,
        }
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>
            {isCreate
              ? t('customers.form.createTitle')
              : t('customers.form.editTitle')}
          </DialogTitle>
          <DialogDescription>
            {isCreate
              ? t('customers.form.createDescription')
              : t('customers.form.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            {isCreate && (
              <FormField
                control={form.control}
                name='key'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('customers.fields.key')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='customer-001'
                        autoComplete='off'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage>
                      {form.formState.errors.key
                        ? t('customers.form.validation.keyPattern')
                        : undefined}
                    </FormMessage>
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('customers.fields.name')}</FormLabel>
                  <FormControl>
                    <Input placeholder='Acme Inc.' {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('customers.form.validation.name')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='primaryEmail'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('customers.fields.primaryEmail')}</FormLabel>
                  <FormControl>
                    <Input
                      type='email'
                      placeholder='billing@acme.com'
                      autoComplete='off'
                      {...field}
                    />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.primaryEmail
                      ? t('customers.form.validation.email')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='currency'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('customers.fields.currency')}</FormLabel>
                  <FormControl>
                    <Input placeholder='USD' maxLength={8} {...field} />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='description'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('customers.fields.description')}</FormLabel>
                  <FormControl>
                    <Textarea rows={3} {...field} />
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
