import { useState } from 'react'
import type { TaxCode } from '@openmeter/client'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useOrgDefaultTaxCodes, useUpdateOrgDefaultTaxCodes } from '@/api/hooks'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

/**
 * Organization default tax codes for the two billing contexts (invoicing,
 * credit grants). Selects are seeded from the server defaults; options only
 * constrain new choices, so a default referencing a deleted code is
 * resubmitted unchanged unless the user actively picks another one.
 */
export function OrgDefaultsCard({ taxCodes }: { taxCodes: TaxCode[] }) {
  const { t } = useTranslation()
  const { data: defaults, isLoading } = useOrgDefaultTaxCodes()
  const updateMutation = useUpdateOrgDefaultTaxCodes()

  const [pickedInvoicingId, setPickedInvoicingId] = useState<string | null>(
    null
  )
  const [pickedCreditGrantId, setPickedCreditGrantId] = useState<string | null>(
    null
  )

  // Derived state instead of effect-synced state: the server defaults are the
  // source of truth until the user actively picks a code for that slot.
  const invoicingId = pickedInvoicingId ?? defaults?.invoicingTaxCode.id ?? null
  const creditGrantId =
    pickedCreditGrantId ?? defaults?.creditGrantTaxCode.id ?? null

  const activeTaxCodes = taxCodes.filter((taxCode) => !taxCode.deletedAt)

  const save = () => {
    if (!invoicingId || !creditGrantId) return
    updateMutation.mutate(
      {
        invoicingTaxCode: { id: invoicingId },
        creditGrantTaxCode: { id: creditGrantId },
      },
      {
        onSuccess: () => {
          toast.success(t('config.taxCodes.defaults.toast.updated'))
        },
        onError: handleServerError,
      }
    )
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className='h-24 animate-pulse bg-muted/50' />
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('config.taxCodes.defaults.title')}</CardTitle>
        <CardDescription>
          {t('config.taxCodes.defaults.description')}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-wrap items-end gap-4'>
        <div className='space-y-2'>
          <Label>{t('config.taxCodes.defaults.invoicingTaxCode')}</Label>
          <Select
            value={invoicingId ?? undefined}
            onValueChange={setPickedInvoicingId}
          >
            <SelectTrigger className='w-56'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {activeTaxCodes.map((taxCode) => (
                <SelectItem key={taxCode.id} value={taxCode.id}>
                  {taxCode.name}（{taxCode.key}）
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className='space-y-2'>
          <Label>{t('config.taxCodes.defaults.creditGrantTaxCode')}</Label>
          <Select
            value={creditGrantId ?? undefined}
            onValueChange={setPickedCreditGrantId}
          >
            <SelectTrigger className='w-56'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {activeTaxCodes.map((taxCode) => (
                <SelectItem key={taxCode.id} value={taxCode.id}>
                  {taxCode.name}（{taxCode.key}）
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button onClick={save} disabled={updateMutation.isPending}>
          {t('config.taxCodes.defaults.save')}
        </Button>
      </CardContent>
    </Card>
  )
}
