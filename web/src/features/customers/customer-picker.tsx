import { useState } from 'react'
import type { Customer } from '@openmeter/client'
import { ChevronsUpDown, Loader2, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCustomers } from '@/api/hooks'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'

type CustomerPickerProps = {
  value: Customer | null
  onChange: (customer: Customer | null) => void
  className?: string
}

/**
 * Searchable customer combobox backed by the server-side name filter
 * (`filter[name][contains]`).
 */
export function CustomerPicker({
  value,
  onChange,
  className,
}: CustomerPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')

  const { data, isLoading } = useCustomers({
    page: 1,
    pageSize: 20,
    search: search || undefined,
  })

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant='outline'
          role='combobox'
          className={cn('w-full justify-between font-normal', className)}
        >
          {value ? (
            <span className='flex min-w-0 items-center gap-2'>
              <UserRound className='size-4 shrink-0 text-muted-foreground' />
              <span className='truncate'>
                {value.name}
                <span className='text-muted-foreground'> ({value.key})</span>
              </span>
            </span>
          ) : (
            <span className='text-muted-foreground'>
              {t('customers.picker.placeholder')}
            </span>
          )}
          <ChevronsUpDown className='size-4 shrink-0 opacity-50' />
        </Button>
      </PopoverTrigger>
      <PopoverContent className='w-80 p-0' align='start'>
        <Command shouldFilter={false}>
          <CommandInput
            placeholder={t('customers.searchPlaceholder')}
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            {isLoading ? (
              <div className='flex items-center justify-center gap-2 py-6 text-sm text-muted-foreground'>
                <Loader2 className='size-4 animate-spin' />
                {t('common.loading')}
              </div>
            ) : (
              <>
                <CommandEmpty>{t('customers.empty')}</CommandEmpty>
                <CommandGroup>
                  {(data?.data ?? []).map((customer) => (
                    <CommandItem
                      key={customer.id}
                      value={customer.id}
                      onSelect={() => {
                        onChange(customer)
                        setOpen(false)
                      }}
                    >
                      <div className='flex min-w-0 flex-col'>
                        <span className='truncate'>{customer.name}</span>
                        <span className='text-xs text-muted-foreground'>
                          {customer.key}
                          {customer.primaryEmail
                            ? ` · ${customer.primaryEmail}`
                            : ''}
                        </span>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
                {(data?.data.length ?? 0) === 20 && (
                  <div className='px-2 py-1.5 text-xs text-muted-foreground'>
                    {t('customers.picker.keepTyping')}
                  </div>
                )}
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

export function CustomerPickerSkeleton() {
  return <Skeleton className='h-9 w-full' />
}
