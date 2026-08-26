import { useEffect, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { userManager, safeRedirectPath } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { AuthLayout } from '@/features/auth/auth-layout'

export const Route = createFileRoute('/auth/callback')({
  component: AuthCallback,
})

// eslint-disable-next-line react-refresh/only-export-components
function AuthCallback() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [error, setError] = useState<unknown>(null)

  useEffect(() => {
    let cancelled = false

    void (async () => {
      try {
        const user = await userManager.signinRedirectCallback()
        if (cancelled) return
        useAuthStore.getState().setOidcUser(user)

        const state = user.state as { redirect?: string } | undefined
        // Defense in depth: the value was sanitized when entering the OIDC
        // state; sanitize again on the way out. `href` accepts arbitrary
        // strings; typed `to` cannot express a user-provided redirect path.
        const redirectTo = safeRedirectPath(state?.redirect)
        navigate({ href: redirectTo, replace: true })
      } catch (err) {
        if (!cancelled) setError(err)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [navigate])

  if (error) {
    return (
      <AuthLayout>
        <Card className='max-w-sm gap-4'>
          <CardHeader>
            <CardTitle className='text-lg tracking-tight'>
              {t('auth.signIn.failedTitle')}
            </CardTitle>
            <CardDescription>
              {t('auth.signIn.failedDescription')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              variant='outline'
              className='w-full'
              onClick={() => navigate({ to: '/sign-in', replace: true })}
            >
              {t('auth.signIn.retry')}
            </Button>
          </CardContent>
        </Card>
      </AuthLayout>
    )
  }

  return (
    <div className='flex h-svh items-center justify-center gap-2 text-muted-foreground'>
      <Loader2 className='animate-spin' />
      <span>{t('auth.signIn.redirecting')}</span>
    </div>
  )
}
