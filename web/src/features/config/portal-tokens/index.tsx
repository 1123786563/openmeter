import { useState } from 'react'
import { Ban, KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useInvalidatePortalToken, usePortalTokens } from '@/api/hooks'
import type { PortalToken } from '@/api/legacy'
import { formatDateTime } from '@/lib/format'
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
import { IssueTokenDialog } from './issue-token-dialog'

/**
 * Consumer portal token administration (v1): issue flow plus the issued-token
 * list. The list response never carries the plaintext (shown once at issuance
 * only), so no column can render it. Invalidation is destructive and goes
 * through a confirm dialog, per row id.
 */
export function PortalTokensPage() {
  const { t } = useTranslation()
  const { data: tokens, isLoading } = usePortalTokens()
  const tokenList = tokens ?? []

  const invalidateMutation = useInvalidatePortalToken()

  const [issueOpen, setIssueOpen] = useState(false)
  const [invalidateTarget, setInvalidateTarget] = useState<PortalToken | null>(
    null
  )

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('config.portalTokens.title')}
          description={t('config.portalTokens.description')}
          actions={
            <Button onClick={() => setIssueOpen(true)}>
              <KeyRound className='size-4' />
              {t('config.portalTokens.issue')}
            </Button>
          }
        />
        <div className='mt-6'>
          <h2 className='mb-3 text-sm font-medium text-muted-foreground'>
            {t('config.portalTokens.list.title')}
          </h2>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('config.portalTokens.list.fields.subject')}
                  </TableHead>
                  <TableHead>
                    {t('config.portalTokens.list.fields.createdAt')}
                  </TableHead>
                  <TableHead>
                    {t('config.portalTokens.list.fields.meters')}
                  </TableHead>
                  <TableHead>
                    {t('config.portalTokens.list.fields.status')}
                  </TableHead>
                  <TableHead className='pr-6 text-right'>
                    {t('config.portalTokens.list.fields.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokenList.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className='pl-6 font-mono font-medium'>
                      {token.subject}
                    </TableCell>
                    <TableCell className='text-muted-foreground'>
                      {formatDateTime(token.createdAt)}
                    </TableCell>
                    <TableCell>
                      {!token.allowedMeterSlugs ||
                      token.allowedMeterSlugs.length === 0 ? (
                        <Badge variant='outline' className='font-normal'>
                          {t('config.portalTokens.list.unrestricted')}
                        </Badge>
                      ) : (
                        <div className='flex flex-wrap gap-1'>
                          {token.allowedMeterSlugs.map((slug) => (
                            <Badge
                              key={slug}
                              variant='secondary'
                              className='font-normal'
                            >
                              {slug}
                            </Badge>
                          ))}
                        </div>
                      )}
                    </TableCell>
                    <TableCell>
                      {token.expired ? (
                        <Badge
                          variant='outline'
                          className='font-normal text-muted-foreground'
                        >
                          {t('config.portalTokens.status.expired')}
                        </Badge>
                      ) : (
                        '—'
                      )}
                    </TableCell>
                    <TableCell className='pr-6'>
                      <div className='flex justify-end'>
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-7 px-2 text-destructive hover:text-destructive'
                          disabled={token.expired}
                          onClick={() => setInvalidateTarget(token)}
                        >
                          <Ban className='size-4' />
                          {t('config.portalTokens.invalidateConfirm.title')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {tokenList.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('config.portalTokens.list.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>
      <IssueTokenDialog open={issueOpen} onOpenChange={setIssueOpen} />

      <ConfirmDialog
        open={Boolean(invalidateTarget)}
        onOpenChange={(open) => !open && setInvalidateTarget(null)}
        title={t('config.portalTokens.invalidateConfirm.title')}
        desc={t('config.portalTokens.invalidateConfirm.description', {
          subject: invalidateTarget?.subject ?? '',
        })}
        confirmText={t('config.portalTokens.invalidateConfirm.title')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={invalidateMutation.isPending}
        handleConfirm={() => {
          if (!invalidateTarget) return
          invalidateMutation.mutate(
            { id: invalidateTarget.id },
            {
              onSuccess: () => {
                toast.success(t('config.portalTokens.toast.invalidated'))
                setInvalidateTarget(null)
              },
              onError: handleServerError,
            }
          )
        }}
      />
    </>
  )
}
