/* eslint-disable react-hooks/incompatible-library --
 * TanStack Table's `useReactTable` returns functions the React Compiler
 * cannot memoize safely; this advisory is expected for the table wrapper.
 */
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { DataTablePagination } from './pagination'

type ServerTableProps<TData> = {
  columns: ColumnDef<TData, unknown>[]
  data: TData[]
  /** 1-based page number from server-side pagination. */
  page: number
  pageSize: number
  total: number | undefined
  /** Receives the resolved pagination after a page/size change. */
  onPageChange: (pagination: { pageIndex: number; pageSize: number }) => void
  isLoading?: boolean
  isFetching?: boolean
  toolbar?: React.ReactNode
  /** Shown when the query resolved with an empty page. */
  emptyMessage?: React.ReactNode
  className?: string
}

/**
 * Table for server-side paginated lists: TanStack Table drives rendering
 * (columns, row model) while page state is owned by the caller's query.
 */
export function ServerTable<TData>({
  columns,
  data,
  page,
  pageSize,
  total,
  onPageChange,
  isLoading,
  isFetching,
  toolbar,
  emptyMessage,
  className,
}: ServerTableProps<TData>) {
  const { t } = useTranslation()
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    pageCount:
      total === undefined ? 1 : Math.max(1, Math.ceil(total / pageSize)),
    state: {
      pagination: { pageIndex: Math.max(0, page - 1), pageSize },
    },
    onPaginationChange: (updater) => {
      const next =
        typeof updater === 'function'
          ? updater({ pageIndex: page - 1, pageSize })
          : updater
      onPageChange(next)
    },
  })

  return (
    <div className={cn('space-y-4', isFetching && 'opacity-70', className)}>
      {toolbar}
      <Card className='overflow-hidden py-0'>
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className='bg-hover/50'>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, row) => (
                <TableRow key={row}>
                  {columns.map((_column, col) => (
                    <TableCell key={col}>
                      <Skeleton className='h-5 w-full' />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : table.getRowModel().rows.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className='h-24 text-center text-muted-foreground'
                >
                  {emptyMessage ?? t('common.table.empty')}
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>
      {total !== undefined && (
        <DataTablePagination table={table} className='justify-end' />
      )}
    </div>
  )
}
