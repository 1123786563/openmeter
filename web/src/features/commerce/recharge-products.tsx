import { useState } from 'react'
import type { CommerceRechargeProduct } from '@openmeter/client'
import { Pencil, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useRechargeProducts, useUpdateRechargeProduct } from '@/api/hooks'
import { formatFen, formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { RechargeProductFormDialog } from './recharge-product-form-dialog'

/**
 * Recharge product catalog with admin write access: create products, edit
 * mutable fields (display name, price), and toggle listing state. `sku`,
 * `kind`, `credits`, and `currency` are immutable after creation.
 */
export function RechargeProductsPage() {
  const { t } = useTranslation()
  const { data, isLoading } = useRechargeProducts()

  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<CommerceRechargeProduct | null>(
    null
  )
  const [toggleTarget, setToggleTarget] =
    useState<CommerceRechargeProduct | null>(null)

  const updateMutation = useUpdateRechargeProduct()

  const openCreate = () => {
    setEditTarget(null)
    setFormOpen(true)
  }

  const openEdit = (product: CommerceRechargeProduct) => {
    setEditTarget(product)
    setFormOpen(true)
  }

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('commerce.rechargeProducts.title')}
          description={t('commerce.rechargeProducts.description')}
          actions={
            <Button onClick={openCreate}>
              <Plus className='size-4' />
              {t('commerce.rechargeProducts.create')}
            </Button>
          }
        />
        <div className='mt-6'>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('commerce.rechargeProducts.name')}
                  </TableHead>
                  <TableHead>
                    {t('commerce.rechargeProducts.credits')}
                  </TableHead>
                  <TableHead>{t('commerce.rechargeProducts.price')}</TableHead>
                  <TableHead>{t('commerce.rechargeProducts.active')}</TableHead>
                  <TableHead>
                    {t('commerce.rechargeProducts.displayOrder')}
                  </TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('commerce.rechargeProducts.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data?.products ?? []).map((product) => (
                  <TableRow key={product.id}>
                    <TableCell className='pl-6 font-medium'>
                      {product.name}
                    </TableCell>
                    <TableCell className='tabular-nums'>
                      {formatNumber(product.credits)}
                    </TableCell>
                    <TableCell className='tabular-nums'>
                      {formatFen(product.priceFen, product.currency)}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant='outline'
                        className={
                          product.active
                            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300'
                            : ''
                        }
                      >
                        {product.active
                          ? t('commerce.rechargeProducts.onSale')
                          : t('commerce.rechargeProducts.offSale')}
                      </Badge>
                    </TableCell>
                    <TableCell className='text-muted-foreground tabular-nums'>
                      {product.displayOrder ?? '—'}
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end gap-1'>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2'
                          onClick={() => openEdit(product)}
                        >
                          <Pencil className='size-4' />
                          {t('common.edit')}
                        </Button>
                        <Button
                          variant='ghost'
                          size='sm'
                          className={
                            product.active
                              ? 'h-7 px-2 text-destructive hover:text-destructive'
                              : 'h-7 px-2'
                          }
                          onClick={() => setToggleTarget(product)}
                        >
                          {product.active
                            ? t('commerce.rechargeProducts.deactivate')
                            : t('commerce.rechargeProducts.activate')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {(data?.products ?? []).length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('commerce.rechargeProducts.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>

      <RechargeProductFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        product={editTarget ?? undefined}
      />

      <ConfirmDialog
        open={Boolean(toggleTarget)}
        onOpenChange={(open) => !open && setToggleTarget(null)}
        title={
          toggleTarget?.active
            ? t('commerce.rechargeProducts.toggleConfirm.deactivateTitle')
            : t('commerce.rechargeProducts.toggleConfirm.activateTitle')
        }
        desc={
          toggleTarget?.active
            ? t(
                'commerce.rechargeProducts.toggleConfirm.deactivateDescription',
                {
                  name: toggleTarget.name,
                  price: formatFen(
                    toggleTarget.priceFen,
                    toggleTarget.currency
                  ),
                }
              )
            : t('commerce.rechargeProducts.toggleConfirm.activateDescription', {
                name: toggleTarget?.name ?? '',
                price: toggleTarget
                  ? formatFen(toggleTarget.priceFen, toggleTarget.currency)
                  : '',
              })
        }
        confirmText={
          toggleTarget?.active
            ? t('commerce.rechargeProducts.deactivate')
            : t('commerce.rechargeProducts.activate')
        }
        cancelBtnText={t('common.cancel')}
        destructive={Boolean(toggleTarget?.active)}
        isLoading={updateMutation.isPending}
        handleConfirm={() => {
          if (!toggleTarget) return
          updateMutation.mutate(
            {
              productId: toggleTarget.id,
              body: { active: !toggleTarget.active },
            },
            {
              onSuccess: () => {
                toast.success(
                  toggleTarget.active
                    ? t('commerce.rechargeProducts.toast.deactivated')
                    : t('commerce.rechargeProducts.toast.activated')
                )
                setToggleTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
