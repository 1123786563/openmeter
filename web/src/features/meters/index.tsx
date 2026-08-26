import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useMeters } from '@/api/hooks'
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

export function MetersPage() {
  const { t } = useTranslation()
  const { data, isLoading } = useMeters()

  return (
    <>
      <Header />
      <Main fixed>
        <PageHeader
          title={t('meters.title')}
          description={t('meters.description')}
        />
        <div className='mt-6'>
          {isLoading ? (
            <div className='space-y-2'>
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className='bg-hover/50'>
                  <TableHead className='pl-6'>
                    {t('meters.fields.slug')}
                  </TableHead>
                  <TableHead>{t('meters.fields.name')}</TableHead>
                  <TableHead>{t('meters.fields.aggregation')}</TableHead>
                  <TableHead>{t('meters.fields.eventType')}</TableHead>
                  <TableHead className='pr-6'>
                    {t('meters.fields.description')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data?.data ?? []).map((meter) => (
                  <TableRow key={meter.id}>
                    <TableCell className='pl-6 font-medium'>
                      <Link
                        to='/meters/$meterId'
                        params={{ meterId: meter.id }}
                        className='hover:underline'
                      >
                        {meter.key}
                      </Link>
                    </TableCell>
                    <TableCell>{meter.name}</TableCell>
                    <TableCell>
                      {t(`meters.aggregation.${meter.aggregation}`, {
                        defaultValue: meter.aggregation,
                      })}
                    </TableCell>
                    <TableCell className='font-mono text-xs'>
                      {meter.eventType}
                    </TableCell>
                    <TableCell className='pr-6 text-muted-foreground'>
                      {meter.description ?? '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {(data?.data ?? []).length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='h-24 text-center text-muted-foreground'
                    >
                      {t('meters.empty')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </Main>
    </>
  )
}
