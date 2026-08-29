import { useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { App } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useApps, useCreateBillingProfile } from '@/api/hooks'
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

const RESOURCE_KEY = /^[a-z0-9]+(?:_[a-z0-9]+)*$/
const COUNTRY_CODE = /^[A-Za-z]{2}$/

/** 扁平表单，提交时映射为 v3 CreateBillingProfileRequest（snake→camel 由 SDK 承担）。 */
const schema = z.object({
  name: z.string().min(1).max(256),
  description: z.string().max(1024).optional(),
  supplierName: z.string().min(1).max(256),
  supplierKey: z
    .string()
    .refine((v) => v === '' || RESOURCE_KEY.test(v), 'invalid'),
  supplierTaxId: z.string().max(32),
  addrCountry: z
    .string()
    .refine((v) => v === '' || COUNTRY_CODE.test(v), 'invalid'),
  addrLine1: z.string(),
  addrLine2: z.string(),
  addrCity: z.string(),
  addrState: z.string(),
  addrPostalCode: z.string(),
  addrPhoneNumber: z.string(),
  appTax: z.string().min(1),
  appInvoicing: z.string().min(1),
  appPayment: z.string().min(1),
  default: z.boolean(),
})

type FormValues = z.infer<typeof schema>

const EMPTY_VALUES: FormValues = {
  name: '',
  description: '',
  supplierName: '',
  supplierKey: '',
  supplierTaxId: '',
  addrCountry: '',
  addrLine1: '',
  addrLine2: '',
  addrCity: '',
  addrState: '',
  addrPostalCode: '',
  addrPhoneNumber: '',
  appTax: '',
  appInvoicing: '',
  appPayment: '',
  default: false,
}

/** App slot select: lists every installed app; which type fits which slot is a backend concern. */
function AppSlotSelect({
  apps,
  value,
  onChange,
}: {
  apps: App[]
  value: string
  onChange: (next: string) => void
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className='w-full'>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {apps.map((app) => (
          <SelectItem key={app.id} value={app.id}>
            {app.name}（{app.type}）
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function BillingProfileFormDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const { data: appsData } = useApps()
  const createMutation = useCreateBillingProfile()

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: EMPTY_VALUES,
  })

  useEffect(() => {
    if (open) form.reset(EMPTY_VALUES)
  }, [open, form])

  const apps = appsData?.data ?? []

  const onSubmit = (values: FormValues) => {
    createMutation.mutate(
      {
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        supplier: {
          name: values.supplierName.trim(),
          key: values.supplierKey.trim() || undefined,
          taxId: values.supplierTaxId.trim()
            ? { code: values.supplierTaxId.trim() }
            : undefined,
          addresses: {
            billingAddress: {
              country: values.addrCountry.trim().toUpperCase() || undefined,
              postalCode: values.addrPostalCode.trim() || undefined,
              state: values.addrState.trim() || undefined,
              city: values.addrCity.trim() || undefined,
              line1: values.addrLine1.trim() || undefined,
              line2: values.addrLine2.trim() || undefined,
              phoneNumber: values.addrPhoneNumber.trim() || undefined,
            },
          },
        },
        // workflow 必传但 UI 只读：空对象让服务端落各 workflow 默认设置
        workflow: {},
        apps: {
          tax: { id: values.appTax },
          invoicing: { id: values.appInvoicing },
          payment: { id: values.appPayment },
        },
        default: values.default,
      },
      {
        onSuccess: () => {
          toast.success(t('config.billingProfiles.toast.created'))
          onOpenChange(false)
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[85vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {t('config.billingProfiles.form.createTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('config.billingProfiles.form.createDescription')}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.billingProfiles.fields.name')}
                    </FormLabel>
                    <FormControl>
                      <Input placeholder='WeKnora 默认开票' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='description'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('config.billingProfiles.fields.description')}
                    </FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='space-y-3 rounded-lg border p-3'>
              <p className='text-sm font-medium'>
                {t('config.billingProfiles.fields.supplier')}
              </p>
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='supplierName'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.supplierName')}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder='Acme Inc.' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='supplierTaxId'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.supplierTaxId')}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder='911234567890' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='supplierKey'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.supplierKey')}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder='acme_supplier' {...field} />
                      </FormControl>
                      <FormDescription>
                        {t('config.billingProfiles.form.keyHint')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrCountry'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.country')}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder='CN' maxLength={2} {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrLine1'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.addressLine1')}
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrLine2'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.addressLine2')}
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrCity'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.city')}
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrState'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.state')}
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrPostalCode'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.postalCode')}
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='addrPhoneNumber'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.phoneNumber')}
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <div className='space-y-3 rounded-lg border p-3'>
              <p className='text-sm font-medium'>
                {t('config.billingProfiles.fields.apps')}
              </p>
              <FormDescription>
                {t('config.billingProfiles.form.appsImmutableHint')}
              </FormDescription>
              <div className='grid gap-4 sm:grid-cols-3'>
                <FormField
                  control={form.control}
                  name='appTax'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.appTax')}
                      </FormLabel>
                      <FormControl>
                        <AppSlotSelect
                          apps={apps}
                          value={field.value}
                          onChange={field.onChange}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='appInvoicing'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.appInvoicing')}
                      </FormLabel>
                      <FormControl>
                        <AppSlotSelect
                          apps={apps}
                          value={field.value}
                          onChange={field.onChange}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='appPayment'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('config.billingProfiles.fields.appPayment')}
                      </FormLabel>
                      <FormControl>
                        <AppSlotSelect
                          apps={apps}
                          value={field.value}
                          onChange={field.onChange}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <FormField
              control={form.control}
              name='default'
              render={({ field }) => (
                <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                  <div className='space-y-0.5'>
                    <FormLabel>
                      {t('config.billingProfiles.fields.default')}
                    </FormLabel>
                    <FormDescription>
                      {t('config.billingProfiles.form.defaultHint')}
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
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
