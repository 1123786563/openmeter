import { useQueryClient } from '@tanstack/react-query'
import { Check, ChevronsUpDown, Layers } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useNamespaces } from '@/api/hooks'
import { useNamespaceStore } from '@/stores/namespace-store'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'

/**
 * Namespace picker mounted in the shared header.
 *
 * Selecting a namespace updates the persisted store (which drives the
 * `X-Namespace` header on every request) and invalidates the whole query
 * cache so every visible list refetches against the new namespace.
 */
export function NamespaceSwitcher() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data, isLoading, isError } = useNamespaces()
  const { currentNamespace, setNamespace } = useNamespaceStore()

  if (isError) return null

  if (isLoading || !data) {
    return <Skeleton className='h-8 w-32 rounded-md' />
  }

  // The store keeps `null` for "ride on the server default".
  const selected = currentNamespace ?? data.default
  const namespaces = data.namespaces

  const select = (namespace: string) => {
    if (namespace === selected) return
    const isDefault = namespace === data.default
    setNamespace(isDefault ? null : namespace)
    void queryClient.invalidateQueries()
    toast.success(
      t('common.namespace.switched', {
        namespace: isDefault ? data.default : namespace,
      })
    )
  }

  // Single namespace: nothing to switch between.
  if (namespaces.length <= 1) {
    return (
      <Badge variant='outline' className='h-8 gap-1.5 px-2.5 font-normal'>
        <Layers className='size-3.5 text-muted-foreground' />
        {selected}
      </Badge>
    )
  }

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button variant='outline' className='h-8 max-w-52 gap-1.5 px-2.5'>
          <Layers className='size-3.5 text-muted-foreground' />
          <span className='truncate font-normal'>{selected}</span>
          <ChevronsUpDown className='size-3.5 text-muted-foreground' />
          <span className='sr-only'>{t('common.namespace.label')}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-52'>
        <DropdownMenuLabel>{t('common.namespace.label')}</DropdownMenuLabel>
        {namespaces.map((namespace) => (
          <DropdownMenuItem
            key={namespace}
            onClick={() => select(namespace)}
            className='gap-2'
          >
            <span className='truncate'>{namespace}</span>
            {namespace === data.default && (
              <Badge variant='secondary' className='ms-auto px-1 py-0 text-xs'>
                {t('common.namespace.default')}
              </Badge>
            )}
            <Check
              size={14}
              className={cn(
                namespace !== selected && 'ms-auto hidden',
                namespace === data.default && 'ms-0'
              )}
            />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
