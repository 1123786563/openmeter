import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface SignOutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SignOutDialog({ open, onOpenChange }: SignOutDialogProps) {
  const { t } = useTranslation()
  const signout = useAuthStore((state) => state.signout)

  const handleSignOut = () => {
    onOpenChange(false)
    // Redirects the browser to Casdoor's end-session endpoint; on failure
    // the local session is still cleared, leaving the user signed out.
    void signout().catch((error) => {
      // eslint-disable-next-line no-console
      console.error('[auth] signout redirect failed', error)
    })
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('common.signOut.title')}
      desc={t('common.signOut.description')}
      confirmText={t('common.signOut.confirm')}
      destructive
      handleConfirm={handleSignOut}
      className='sm:max-w-sm'
    />
  )
}
