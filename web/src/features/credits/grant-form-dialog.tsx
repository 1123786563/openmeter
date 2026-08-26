import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreateCreditGrant } from '@/api/hooks'
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

const grantSchema = z.object({
  name: z.string().min(1).max(256),
  amount: z
    .string()
    .min(1)
    .refine(
      (value) => Number.isFinite(Number(value)) && Number(value) > 0,
      'credits.grantForm.validation.amount'
    ),
  currency: z.string().min(3).max(8),
  fundingMethod: z.enum(['none', 'invoice', 'external']),
  priority: z
    .string()
    .refine(
      (value) => Number.isInteger(Number(value)) && Number(value) >= 1,
      'credits.grantForm.validation.priority'
    ),
  effectiveAt: z.string().optional(),
  expiresAfter: z.string().optional(),
  description: z.string().max(1024).optional(),
})

type GrantFormValues = z.infer<typeof grantSchema>

export function GrantFormDialog({
  open,
  onOpenChange,
  customerId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  customerId: string
}) {
  const { t } = useTranslation()
  const createMutation = useCreateCreditGrant()

  const form = useForm<GrantFormValues>({
    resolver: zodResolver(grantSchema),
    defaultValues: {
      name: '',
      amount: '',
      currency: 'USD',
      fundingMethod: 'none',
      priority: '1',
      effectiveAt: '',
      expiresAfter: '',
      description: '',
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        name: '',
        amount: '',
        currency: 'USD',
        fundingMethod: 'none',
        priority: '1',
        effectiveAt: '',
        expiresAfter: '',
        description: '',
      })
    }
  }, [open, form])

  const onSubmit = (values: GrantFormValues) => {
    createMutation.mutate(
      {
        customerId,
        body: {
          name: values.name,
          amount: values.amount,
          currency: values.currency,
          fundingMethod: values.fundingMethod,
          priority: Number(values.priority),
          description: values.description || undefined,
          effectiveAt: values.effectiveAt
            ? new Date(values.effectiveAt)
            : undefined,
          expiresAfter: values.expiresAfter || undefined,
        },
      },
      {
        onSuccess: () => {
          toast.success(t('credits.toast.grantCreated'))
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
          <DialogTitle>{t('credits.grantForm.title')}</DialogTitle>
          <DialogDescription>
            {t('credits.grantForm.description')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('credits.grantForm.name')}</FormLabel>
                  <FormControl>
                    <Input placeholder='Q3 marketing credits' {...field} />
                  </FormControl>
                  <FormMessage>
                    {form.formState.errors.name
                      ? t('credits.grantForm.validation.name')
                      : undefined}
                  </FormMessage>
                </FormItem>
              )}
            />
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='amount'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('credits.grantForm.amount')}</FormLabel>
                    <FormControl>
                      <Input inputMode='decimal' placeholder='100' {...field} />
                    </FormControl>
                    <FormMessage>
                      {form.formState.errors.amount
                        ? t('credits.grantForm.validation.amount')
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
                    <FormLabel>{t('credits.grantForm.currency')}</FormLabel>
                    <FormControl>
                      <Input placeholder='USD' {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='fundingMethod'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('credits.grantForm.fundingMethod')}
                    </FormLabel>
                    <FormControl>
                      <select
                        className='h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs'
                        {...field}
                      >
                        {(['none', 'invoice', 'external'] as const).map(
                          (option) => (
                            <option key={option} value={option}>
                              {t(`credits.grantForm.fundingMethod_${option}`)}
                            </option>
                          )
                        )}
                      </select>
                    </FormControl>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='priority'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('credits.grantForm.priority')}</FormLabel>
                    <FormControl>
                      <Input
                        inputMode='numeric'
                        type='number'
                        min={1}
                        {...field}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <FormField
                control={form.control}
                name='effectiveAt'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('credits.grantForm.effectiveAt')}</FormLabel>
                    <FormControl>
                      <Input type='datetime-local' {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='expiresAfter'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('credits.grantForm.expiresAfter')}</FormLabel>
                    <FormControl>
                      <Input placeholder='P30D' {...field} />
                    </FormControl>
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
                    {t('credits.grantForm.descriptionLabel')}
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
                  : t('credits.grantForm.submit')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
