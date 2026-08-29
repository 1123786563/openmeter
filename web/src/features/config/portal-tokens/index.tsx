import { useState } from 'react'
import { KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { IssueTokenDialog } from './issue-token-dialog'

/**
 * Consumer portal token issuance (v1). The list + invalidate land in the
 * follow-up issue; this page ships the issue flow with the one-time
 * plaintext reveal.
 */
export function PortalTokensPage() {
  const { t } = useTranslation()
  const [issueOpen, setIssueOpen] = useState(false)

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
      </Main>
      <IssueTokenDialog open={issueOpen} onOpenChange={setIssueOpen} />
    </>
  )
}
