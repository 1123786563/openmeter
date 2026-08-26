import { useState } from 'react'
import { Loader2, LogIn } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

interface UserAuthFormProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Path to return to after the OIDC callback completes. */
  redirectTo?: string
}

export function UserAuthForm({
  className,
  redirectTo,
  ...props
}: UserAuthFormProps) {
  const { t } = useTranslation()
  const [isRedirecting, setIsRedirecting] = useState(false)
  const signin = useAuthStore((state) => state.signin)

  async function handleCasdoorSignIn() {
    setIsRedirecting(true)
    try {
      await signin(redirectTo)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('[auth] signin redirect failed', error)
      setIsRedirecting(false)
    }
  }

  return (
    <div className={cn('grid gap-3', className)} {...props}>
      <Button onClick={handleCasdoorSignIn} disabled={isRedirecting}>
        {isRedirecting ? <Loader2 className='animate-spin' /> : <LogIn />}
        {isRedirecting ? t('auth.signIn.signingIn') : t('auth.signIn.casdoor')}
      </Button>
    </div>
  )
}
