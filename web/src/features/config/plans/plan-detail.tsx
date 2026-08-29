import { useState } from 'react'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import type { RateCard } from '@openmeter/client'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  useArchivePlan,
  useClonePlanNext,
  usePlan,
  usePublishPlan,
} from '@/api/hooks'
import { formatAmount, formatDateTime } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import { EnumBadge, StatusBadge } from '@/components/status-badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PlanAddonsTab } from './plan-addons-tab'
import { PlanFormWizard } from './plan-form-wizard'

function InfoRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className='flex items-baseline justify-between gap-4 py-1.5'>
      <span className='text-sm text-muted-foreground'>{label}</span>
      <span className='text-end text-sm font-medium break-all'>{children}</span>
    </div>
  )
}

/** Human-readable price summary per rate card; tiers collapse to a count. */
function RateCardPriceSummary({
  card,
  planCurrency,
}: {
  card: RateCard
  planCurrency: string
}) {
  const { t } = useTranslation()
  const currency = card.currency ?? planCurrency
  switch (card.price.type) {
    case 'free':
      return (
        <span className='text-muted-foreground'>{t('plan.price.free')}</span>
      )
    case 'flat':
      return (
        <span className='tabular-nums'>
          {formatAmount(card.price.amount, currency)}
        </span>
      )
    case 'unit':
      return (
        <span className='tabular-nums'>
          {formatAmount(card.price.amount, currency)} /{' '}
          {t('config.plans.detail.unit')}
        </span>
      )
    case 'graduated':
    case 'volume':
      return (
        <span className='text-muted-foreground'>
          {t('config.plans.detail.tierSummary', {
            count: card.price.tiers.length,
          })}
        </span>
      )
  }
}

export function PlanDetail() {
  const { t } = useTranslation()
  const { planId } = useParams({ from: '/_authenticated/config/plans/$planId' })
  const { data: plan, isLoading } = usePlan(planId)

  const navigate = useNavigate()
  const [confirming, setConfirming] = useState<
    'publish' | 'archive' | 'clone' | null
  >(null)

  const publishMutation = usePublishPlan()
  const archiveMutation = useArchivePlan()
  const cloneMutation = useClonePlanNext()

  const [editOpen, setEditOpen] = useState(false)

  const statusBusy =
    publishMutation.isPending ||
    archiveMutation.isPending ||
    cloneMutation.isPending

  if (isLoading) {
    return (
      <>
        <Header />
        <Main fixed>
          <Skeleton className='h-8 w-64' />
          <Skeleton className='mt-6 h-64 w-full' />
        </Main>
      </>
    )
  }
  if (!plan) {
    return (
      <>
        <Header />
        <Main fixed>
          <p className='text-sm text-muted-foreground'>
            {t('config.plans.detail.notFound')}
          </p>
        </Main>
      </>
    )
  }

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2'>
          <Button variant='ghost' size='sm' asChild>
            <Link to='/config/plans'>
              <ArrowLeft className='size-4' />
              {t('config.plans.detail.back')}
            </Link>
          </Button>
        </div>
        <div className='mt-4 flex items-center gap-3'>
          <h1 className='text-2xl font-bold tracking-tight'>{plan.name}</h1>
          <StatusBadge domain='plan' value={plan.status} />
          <span className='text-sm text-muted-foreground'>v{plan.version}</span>
          {!statusBusy && plan.status === 'draft' && (
            <Button size='sm' onClick={() => setConfirming('publish')}>
              {t('config.plans.actions.publish')}
            </Button>
          )}
          {!statusBusy && plan.status === 'draft' && (
            <Button
              size='sm'
              variant='outline'
              onClick={() => setEditOpen(true)}
            >
              {t('common.edit')}
            </Button>
          )}
          {!statusBusy && plan.status === 'active' && (
            <Button
              size='sm'
              variant='outline'
              onClick={() => setConfirming('archive')}
            >
              {t('config.plans.actions.archive')}
            </Button>
          )}
          {!statusBusy && plan.status !== 'draft' && (
            <Button
              size='sm'
              variant='outline'
              onClick={() => setConfirming('clone')}
            >
              {t('config.plans.actions.cloneNext')}
            </Button>
          )}
        </div>

        <Tabs defaultValue='overview' className='mt-6'>
          <TabsList>
            <TabsTrigger value='overview'>
              {t('config.planDetail.tabs.overview')}
            </TabsTrigger>
            <TabsTrigger value='addons'>
              {t('config.planDetail.tabs.addons')}
            </TabsTrigger>
          </TabsList>
          <TabsContent value='overview' className='mt-4'>
            <Card className='py-0'>
              <CardContent className='divide-y'>
                <InfoRow label={t('config.plans.fields.key')}>
                  <span className='font-mono text-xs'>{plan.key}</span>
                </InfoRow>
                <InfoRow label={t('config.plans.fields.currency')}>
                  {plan.currency}
                </InfoRow>
                <InfoRow label={t('config.plans.fields.billingCadence')}>
                  {plan.billingCadence}
                </InfoRow>
                <InfoRow label={t('config.plans.fields.createdAt')}>
                  {formatDateTime(plan.createdAt)}
                </InfoRow>
                <InfoRow label={t('config.plans.fields.updatedAt')}>
                  {formatDateTime(plan.updatedAt)}
                </InfoRow>
                {plan.description ? (
                  <InfoRow label={t('config.plans.fields.description')}>
                    {plan.description}
                  </InfoRow>
                ) : null}
              </CardContent>
            </Card>

            <h2 className='mt-8 text-lg font-semibold'>
              {t('config.plans.detail.phases')}
            </h2>
            {plan.phases.map((phase, phaseIndex) => (
              <Card key={phase.key ?? phaseIndex} className='mt-4 py-0'>
                <CardHeader>
                  <CardTitle className='flex items-baseline gap-2 text-base'>
                    <span>
                      {t('config.plans.detail.phaseIndex', {
                        index: phaseIndex + 1,
                      })}{' '}
                      · {phase.name}
                    </span>
                    <span className='text-xs font-normal text-muted-foreground'>
                      {phase.duration
                        ? t('config.plans.detail.duration', {
                            duration: phase.duration,
                          })
                        : t('config.plans.detail.noDuration')}
                    </span>
                  </CardTitle>
                </CardHeader>
                <Table>
                  <TableHeader>
                    <TableRow className='bg-hover/50'>
                      <TableHead className='pl-6'>
                        {t('config.plans.detail.rateCardName')}
                      </TableHead>
                      <TableHead>
                        {t('config.plans.detail.priceType')}
                      </TableHead>
                      <TableHead>{t('config.plans.detail.feature')}</TableHead>
                      <TableHead>{t('config.plans.detail.price')}</TableHead>
                      <TableHead>{t('config.plans.detail.cadence')}</TableHead>
                      <TableHead className='pr-6'>
                        {t('config.plans.detail.key')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {phase.rateCards.map((card) => (
                      <TableRow key={card.key}>
                        <TableCell className='pl-6 font-medium'>
                          {card.name}
                        </TableCell>
                        <TableCell>
                          <EnumBadge
                            domain='plan'
                            kind='priceType'
                            value={card.price.type}
                          />
                        </TableCell>
                        <TableCell className='font-mono text-xs text-muted-foreground'>
                          {card.feature?.id ?? '—'}
                        </TableCell>
                        <TableCell>
                          <RateCardPriceSummary
                            card={card}
                            planCurrency={plan.currency}
                          />
                        </TableCell>
                        <TableCell className='text-muted-foreground'>
                          {card.billingCadence ?? plan.billingCadence}
                        </TableCell>
                        <TableCell className='pr-6 font-mono text-xs text-muted-foreground'>
                          {card.key}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </Card>
            ))}
          </TabsContent>
          <TabsContent value='addons' className='mt-4'>
            <PlanAddonsTab planId={plan.id} />
          </TabsContent>
        </Tabs>
      </Main>

      <ConfirmDialog
        open={confirming === 'publish'}
        onOpenChange={(open) => !open && setConfirming(null)}
        title={t('config.plans.publishConfirm.title')}
        desc={t('config.plans.publishConfirm.description', {
          name: plan.name,
        })}
        confirmText={t('config.plans.actions.publish')}
        cancelBtnText={t('common.cancel')}
        isLoading={publishMutation.isPending}
        handleConfirm={() => {
          publishMutation.mutate(
            { planId: plan.id },
            {
              onSuccess: () => {
                toast.success(t('config.plans.toast.published'))
                setConfirming(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />

      <ConfirmDialog
        open={confirming === 'archive'}
        onOpenChange={(open) => !open && setConfirming(null)}
        title={t('config.plans.archiveConfirm.title')}
        desc={t('config.plans.archiveConfirm.description', {
          name: plan.name,
        })}
        confirmText={t('config.plans.actions.archive')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={archiveMutation.isPending}
        handleConfirm={() => {
          archiveMutation.mutate(
            { planId: plan.id },
            {
              onSuccess: () => {
                toast.success(t('config.plans.toast.archived'))
                setConfirming(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />

      <ConfirmDialog
        open={confirming === 'clone'}
        onOpenChange={(open) => !open && setConfirming(null)}
        title={t('config.plans.cloneConfirm.title')}
        desc={t('config.plans.cloneConfirm.description', {
          name: plan.name,
          version: plan.version,
        })}
        confirmText={t('config.plans.actions.cloneNext')}
        cancelBtnText={t('common.cancel')}
        isLoading={cloneMutation.isPending}
        handleConfirm={() => {
          cloneMutation.mutate(
            { planIdOrKey: plan.id },
            {
              onSuccess: (draft) => {
                toast.success(
                  t('config.plans.toast.cloned', { version: draft.version })
                )
                setConfirming(null)
                void navigate({
                  to: '/config/plans/$planId',
                  params: { planId: draft.id },
                })
              },
              onError: handleServerError,
            }
          )
        }}
      />

      <PlanFormWizard open={editOpen} onOpenChange={setEditOpen} plan={plan} />
    </>
  )
}
