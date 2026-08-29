import { useState } from 'react'
import type { Customer, Meter } from '@openmeter/client'
import { ChevronsUpDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCreatePortalToken, useMeters } from '@/api/hooks'
import type { PortalToken } from '@/api/legacy'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { CustomerPicker } from '@/features/customers/customer-picker'
import { TokenOnceDialog } from './token-once-dialog'

/** Meter slug multi-select (Popover + Command + Checkbox); empty = all meters. */
function MeterMultiSelect({
  meters,
  value,
  onChange,
}: {
  meters: Meter[]
  value: string[]
  onChange: (next: string[]) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const toggle = (slug: string) =>
    onChange(
      value.includes(slug) ? value.filter((s) => s !== slug) : [...value, slug]
    )

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant='outline'
          role='combobox'
          className='w-full justify-between font-normal'
        >
          <span className='truncate'>
            {value.length === 0 ? (
              <span className='text-muted-foreground'>
                {t('config.portalTokens.form.allMeters')}
              </span>
            ) : (
              value.join(', ')
            )}
          </span>
          <ChevronsUpDown className='size-4 shrink-0 opacity-50' />
        </Button>
      </PopoverTrigger>
      <PopoverContent className='w-80 p-0' align='start'>
        <Command>
          <CommandInput
            placeholder={t('config.portalTokens.form.meterSearch')}
          />
          <CommandList>
            <CommandEmpty>
              {t('config.portalTokens.form.noMeters')}
            </CommandEmpty>
            <CommandGroup>
              {meters.map((meter) => (
                <CommandItem
                  key={meter.id}
                  value={meter.key}
                  onSelect={() => toggle(meter.key)}
                >
                  <Checkbox
                    checked={value.includes(meter.key)}
                    className='me-2'
                    onCheckedChange={() => toggle(meter.key)}
                  />
                  <span className='truncate'>{meter.name}</span>
                  <span className='ms-2 text-xs text-muted-foreground'>
                    {meter.key}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

export function IssueTokenDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const { data: metersData } = useMeters()
  const createMutation = useCreatePortalToken()

  const [customer, setCustomer] = useState<Customer | null>(null)
  const [meterSlugs, setMeterSlugs] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)
  const [issuedToken, setIssuedToken] = useState<string | null>(null)

  // 表单字段在关闭时于事件处理器中复位（挂载即干净），避免 effect 内 setState。
  // issuedToken 不在此复位：由一次性明文弹窗自身的 onClose 清空。
  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setCustomer(null)
      setMeterSlugs([])
      setError(null)
    }
    onOpenChange(next)
  }

  const submit = () => {
    if (!customer) {
      setError(t('config.portalTokens.form.customerRequired'))
      return
    }
    createMutation.mutate(
      {
        // v1 portal token 的 subject 即客户的 usage subject（= customer.key）
        subject: customer.key,
        allowedMeterSlugs:
          meterSlugs.length > 0 ? meterSlugs : undefined,
      },
      {
        onSuccess: (portalToken: PortalToken) => {
          toast.success(t('config.portalTokens.toast.issued'))
          handleOpenChange(false)
          if (portalToken.token) {
            setIssuedToken(portalToken.token)
          } else {
            // spec 承诺创建响应必含 token；缺失属异常，如实提示
            toast.error(t('config.portalTokens.toast.noPlaintext'))
          }
        },
        onError: handleServerError,
      }
    )
  }

  return (
    <>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('config.portalTokens.form.title')}</DialogTitle>
            <DialogDescription>
              {t('config.portalTokens.form.description')}
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-4'>
            <div className='space-y-2'>
              <Label>{t('config.portalTokens.form.customer')}</Label>
              <CustomerPicker value={customer} onChange={setCustomer} />
              {error && <p className='text-sm text-destructive'>{error}</p>}
            </div>
            <div className='space-y-2'>
              <Label>{t('config.portalTokens.form.allowedMeters')}</Label>
              <MeterMultiSelect
                meters={metersData?.data ?? []}
                value={meterSlugs}
                onChange={setMeterSlugs}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => handleOpenChange(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type='button'
              onClick={submit}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending
                ? t('common.submitting')
                : t('config.portalTokens.issue')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <TokenOnceDialog token={issuedToken} onClose={() => setIssuedToken(null)} />
    </>
  )
}
