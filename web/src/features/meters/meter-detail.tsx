import { useState } from 'react'
import { subDays } from 'date-fns'
import { Link, useParams } from '@tanstack/react-router'
import { Play } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useMeter, useMeterQuery, useSubjects } from '@/api/hooks'
import { formatDateTime, formatNumber } from '@/lib/format'
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

function toLocalInputValue(date: Date): string {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

export function MeterDetail() {
  const { t } = useTranslation()
  const { meterId } = useParams({ from: '/_authenticated/meters/$meterId' })
  const { data: meter, isLoading } = useMeter(meterId)
  const { data: subjects } = useSubjects()

  const [fromInput, setFromInput] = useState(
    toLocalInputValue(subDays(new Date(), 30))
  )
  const [toInput, setToInput] = useState(toLocalInputValue(new Date()))
  const [subject, setSubject] = useState<string>('__all__')
  const [submitted, setSubmitted] = useState<{
    from?: Date
    to?: Date
    subject?: string
  }>({ from: subDays(new Date(), 30), to: new Date() })

  const query = useMeterQuery({
    meterId,
    from: submitted.from,
    to: submitted.to,
    subject: submitted.subject,
  })

  if (isLoading) {
    return (
      <>
        <Header />
        <Main fixed>
          <Skeleton className='h-64 w-full' />
        </Main>
      </>
    )
  }
  if (!meter) return null

  const runQuery = () => {
    setSubmitted({
      from: fromInput ? new Date(fromInput) : undefined,
      to: toInput ? new Date(toInput) : undefined,
      subject: subject === '__all__' ? undefined : subject,
    })
  }

  return (
    <>
      <Header />
      <Main fixed>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Link to='/meters' className='hover:text-foreground'>
            {t('sidebar.meters')}
          </Link>
          <span>/</span>
          <span className='text-foreground'>{meter.key}</span>
        </div>
        <h1 className='mt-2 text-2xl font-bold tracking-tight md:text-3xl'>
          {meter.name}
        </h1>
        <p className='mt-1 text-sm text-muted-foreground'>
          {t('meters.aggregation.' + meter.aggregation, {
            defaultValue: meter.aggregation,
          })}
          {meter.description ? ` · ${meter.description}` : ''}
        </p>

        <Card className='mt-6 py-0'>
          <CardHeader>
            <CardTitle className='text-base'>
              {t('meters.detail.usageQuery')}
            </CardTitle>
          </CardHeader>
          <CardContent className='space-y-4 pt-0'>
            <div className='flex flex-wrap items-end gap-3'>
              <div className='space-y-1'>
                <label className='text-xs text-muted-foreground'>
                  {t('meters.detail.from')}
                </label>
                <Input
                  type='datetime-local'
                  className='h-8 w-56'
                  value={fromInput}
                  onChange={(event) => setFromInput(event.target.value)}
                />
              </div>
              <div className='space-y-1'>
                <label className='text-xs text-muted-foreground'>
                  {t('meters.detail.to')}
                </label>
                <Input
                  type='datetime-local'
                  className='h-8 w-56'
                  value={toInput}
                  onChange={(event) => setToInput(event.target.value)}
                />
              </div>
              <div className='space-y-1'>
                <label className='text-xs text-muted-foreground'>
                  {t('meters.detail.subject')}
                </label>
                <Select value={subject} onValueChange={setSubject}>
                  <SelectTrigger className='h-8 w-52'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='__all__'>
                      {t('meters.detail.allSubjects')}
                    </SelectItem>
                    {(subjects ?? []).map((item) => (
                      <SelectItem key={item.id} value={item.key}>
                        {item.key}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button size='sm' className='h-8' onClick={runQuery}>
                <Play className='size-3.5' />
                {t('meters.detail.runQuery')}
              </Button>
            </div>

            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead>{t('meters.detail.windowFrom')}</TableHead>
                  <TableHead>{t('meters.detail.windowTo')}</TableHead>
                  <TableHead>{t('meters.detail.value')}</TableHead>
                  <TableHead>{t('meters.detail.dimensions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {query.isLoading ? (
                  Array.from({ length: 4 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell colSpan={4}>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  ))
                ) : (query.data?.data.length ?? 0) === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={4}
                      className='h-20 text-center text-muted-foreground'
                    >
                      {t('meters.detail.noUsage')}
                    </TableCell>
                  </TableRow>
                ) : (
                  query.data?.data.map((row, index) => (
                    <TableRow key={index}>
                      <TableCell className='text-muted-foreground'>
                        {formatDateTime(row.from)}
                      </TableCell>
                      <TableCell className='text-muted-foreground'>
                        {formatDateTime(row.to)}
                      </TableCell>
                      <TableCell className='font-medium tabular-nums'>
                        {formatNumber(row.value)}
                      </TableCell>
                      <TableCell>
                        {Object.keys(row.dimensions).length === 0 ? (
                          <span className='text-muted-foreground'>—</span>
                        ) : (
                          Object.entries(row.dimensions).map(([key, value]) => (
                            <span
                              key={key}
                              className='me-2 inline-block rounded bg-muted px-1.5 py-0.5 font-mono text-xs'
                            >
                              {key}={value}
                            </span>
                          ))
                        )}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </Main>
    </>
  )
}
