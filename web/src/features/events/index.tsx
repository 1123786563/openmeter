import { useState } from 'react'
import { subDays } from 'date-fns'
import { ChevronRight, Play } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useEvents, useSubjects } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'

function toLocalInputValue(date: Date): string {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

/**
 * Ingested-event browser (v3 events, cursor paginated). "Load more" advances
 * the cursor; filters reset it.
 */
export function EventsPage() {
  const { t } = useTranslation()
  const { data: subjects } = useSubjects()

  const [fromInput, setFromInput] = useState(
    toLocalInputValue(subDays(new Date(), 7))
  )
  const [toInput, setToInput] = useState('')
  const [subject, setSubject] = useState('__all__')
  const [filters, setFilters] = useState<{
    subject?: string
    from?: Date
    to?: Date
  }>({})
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [pages, setPages] = useState<
    Awaited<ReturnType<typeof useEvents>['data']>[]
  >([])

  const { data, isLoading, isFetching } = useEvents({
    ...filters,
    after: cursor,
  })

  // Accumulate loaded pages for the "load more" experience: `pages` holds the
  // data of cursors we already advanced past; `data` is the current cursor's.
  const allEvents = [
    ...pages.flatMap((page) => page?.data ?? []),
    ...(data?.data ?? []),
  ]

  const applyFilters = () => {
    setPages([])
    setCursor(undefined)
    setFilters({
      subject: subject === '__all__' ? undefined : subject,
      from: fromInput ? new Date(fromInput) : undefined,
      to: toInput ? new Date(toInput) : undefined,
    })
  }

  const loadMore = () => {
    if (data?.meta.page.next) {
      setPages((prev) => [...prev, data])
      setCursor(data.meta.page.next ?? undefined)
    }
  }

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('events.title')}
          description={t('events.description')}
        />
        <Card className='mt-6 py-0'>
          <CardHeader className='flex flex-wrap items-end gap-3 space-y-0'>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('events.filter.from')}
              </CardTitle>
              <Input
                type='datetime-local'
                className='h-8 w-56'
                value={fromInput}
                onChange={(event) => setFromInput(event.target.value)}
              />
            </div>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('events.filter.to')}
              </CardTitle>
              <Input
                type='datetime-local'
                className='h-8 w-56'
                value={toInput}
                onChange={(event) => setToInput(event.target.value)}
              />
            </div>
            <div className='space-y-1'>
              <CardTitle className='text-xs font-normal text-muted-foreground'>
                {t('events.filter.subject')}
              </CardTitle>
              <Select value={subject} onValueChange={setSubject}>
                <SelectTrigger className='h-8 w-52'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='__all__'>
                    {t('events.filter.allSubjects')}
                  </SelectItem>
                  {(subjects ?? []).map((item) => (
                    <SelectItem key={item.id} value={item.key}>
                      {item.key}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button size='sm' className='h-8' onClick={applyFilters}>
              <Play className='size-3.5' />
              {t('events.filter.apply')}
            </Button>
          </CardHeader>
          <CardContent className='pt-0'>
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead>{t('events.fields.id')}</TableHead>
                  <TableHead>{t('events.fields.type')}</TableHead>
                  <TableHead>{t('events.fields.subject')}</TableHead>
                  <TableHead>{t('events.fields.time')}</TableHead>
                  <TableHead>{t('events.fields.ingestedAt')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading && pages.length === 0 ? (
                  Array.from({ length: 6 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell colSpan={5}>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  ))
                ) : allEvents.length === 0 && !isFetching ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('events.empty')}
                    </TableCell>
                  </TableRow>
                ) : (
                  allEvents.map((item) => (
                    <TableRow key={item.event.id}>
                      <TableCell className='font-mono text-xs'>
                        {item.event.id}
                      </TableCell>
                      <TableCell>{item.event.type}</TableCell>
                      <TableCell>{item.event.subject}</TableCell>
                      <TableCell className='text-muted-foreground'>
                        {formatDateTime(item.event.time ?? undefined)}
                      </TableCell>
                      <TableCell className='text-muted-foreground'>
                        {formatDateTime(item.ingestedAt)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
            {data?.meta.page.next && (
              <div className='flex justify-center pt-4 pb-2'>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={loadMore}
                  disabled={isFetching}
                >
                  {t('events.loadMore')}
                  <ChevronRight className='size-3.5' />
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      </Main>
    </>
  )
}
