import { useEffect, useRef, useState } from 'react'
import { AlertTriangle, Check, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

/**
 * One-time portal token reveal. The plaintext (om_portal_...) is returned by
 * the create endpoint only; once this dialog closes it can never be fetched
 * again (the list endpoint never includes it). Clipboard write falls back to
 * a hidden textarea + execCommand for non-secure (http dev) contexts.
 */
export function TokenOnceDialog({
  token,
  onClose,
}: {
  token: string | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  // 归属于当前 token 值的状态：换 token 即自动复位，避免 effect 内 setState。
  const [copiedToken, setCopiedToken] = useState<string | null>(null)
  const [failedToken, setFailedToken] = useState<string | null>(null)
  const copied = token !== null && copiedToken === token
  const copyFailed = token !== null && failedToken === token
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (token) inputRef.current?.select()
  }, [token])

  const copyToken = async () => {
    if (!token) return
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(token)
        setCopiedToken(token)
        return
      }
      // 降级：非安全上下文（http dev server）无 navigator.clipboard
      const textarea = document.createElement('textarea')
      textarea.value = token
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(textarea)
      if (!ok) throw new Error('execCommand copy failed')
      setCopiedToken(token)
    } catch {
      setFailedToken(token)
    }
  }

  return (
    <Dialog open={Boolean(token)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className='sm:max-w-lg' onPointerDownOutside={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>{t('config.portalTokens.onceTitle')}</DialogTitle>
          <DialogDescription>
            {t('config.portalTokens.onceDescription')}
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-3'>
          <div className='flex items-center gap-2'>
            <Input
              ref={inputRef}
              readOnly
              value={token ?? ''}
              className='font-mono text-xs'
              onFocus={(e) => e.target.select()}
            />
            <Button
              type='button'
              variant='outline'
              size='sm'
              className='h-9 shrink-0'
              onClick={copyToken}
            >
              {copied ? (
                <Check className='size-3.5' />
              ) : (
                <Copy className='size-3.5' />
              )}
              {copied
                ? t('config.portalTokens.copied')
                : t('config.portalTokens.copy')}
            </Button>
          </div>
          {copyFailed && (
            <p className='text-sm text-destructive'>
              {t('config.portalTokens.copyFailed')}
            </p>
          )}
          <p className='flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-2 text-sm text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300'>
            <AlertTriangle className='mt-0.5 size-4 shrink-0' />
            {t('config.portalTokens.onceWarning')}
          </p>
        </div>
        <DialogFooter>
          <Button type='button' onClick={onClose}>
            {t('config.portalTokens.copiedClose')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
