import { Table, TableBody, TableCell, TableFooter, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatCurrency } from '@/utils/format'

export interface BreakdownColumn {
  label: string
  format?: (value: number) => string
}

export interface BreakdownRow {
  key: string
  label: string
  values: number[]
  deduction?: boolean
}

interface BreakdownTableProps {
  itemLabel: string
  columns: BreakdownColumn[]
  rows: BreakdownRow[]
  total?: { label: string; values: number[] }
}

function formatCell(column: BreakdownColumn, value: number, deduction?: boolean) {
  const text = (column.format ?? formatCurrency)(value)
  return deduction && value !== 0 ? `−${text}` : text
}

export function BreakdownTable({ itemLabel, columns, rows, total }: BreakdownTableProps) {
  const [primary, ...secondary] = columns

  return (
    <>
      <ul className="divide-y sm:hidden">
        {rows.map(row => (
          <li key={row.key} className="flex flex-col gap-0.5 px-4 py-2.5">
            <div className="flex items-center justify-between gap-3">
              <span className="truncate text-sm text-foreground">{row.label}</span>
              <span className="shrink-0 text-sm font-semibold tabular-nums text-foreground">
                {formatCell(primary, row.values[0], row.deduction)}
              </span>
            </div>
            {secondary.length > 0 && (
              <div className="flex flex-wrap justify-end gap-x-3 text-xs text-muted-foreground">
                {secondary.map((column, i) => (
                  <span key={column.label}>
                    {column.label}:{' '}
                    <span className="tabular-nums">{formatCell(column, row.values[i + 1], row.deduction)}</span>
                  </span>
                ))}
              </div>
            )}
          </li>
        ))}
        {total && (
          <li className="flex flex-col gap-0.5 bg-muted/50 px-4 py-3">
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm font-semibold text-foreground">{total.label}</span>
              <span className="shrink-0 font-semibold tabular-nums text-foreground">
                {formatCell(primary, total.values[0])}
              </span>
            </div>
            {secondary.length > 0 && (
              <div className="flex flex-wrap justify-end gap-x-3 text-xs text-muted-foreground">
                {secondary.map((column, i) => (
                  <span key={column.label}>
                    {column.label}:{' '}
                    <span className="font-medium tabular-nums">{formatCell(column, total.values[i + 1])}</span>
                  </span>
                ))}
              </div>
            )}
          </li>
        )}
      </ul>

      <Table className="hidden sm:table">
        <TableHeader className="bg-muted/50">
          <TableRow>
            <TableHead className="px-5 text-muted-foreground">{itemLabel}</TableHead>
            {columns.map(column => (
              <TableHead key={column.label} className="px-5 text-right text-muted-foreground">{column.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(row => (
            <TableRow key={row.key}>
              <TableCell className="px-5 font-medium text-foreground">{row.label}</TableCell>
              {columns.map((column, i) => (
                <TableCell key={column.label} className="px-5 text-right tabular-nums text-foreground">
                  {formatCell(column, row.values[i], row.deduction)}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
        {total && (
          <TableFooter className="bg-muted/50">
            <TableRow className="hover:bg-transparent">
              <TableCell className="px-5 font-semibold text-foreground">{total.label}</TableCell>
              {columns.map((column, i) => (
                <TableCell key={column.label} className="px-5 text-right font-semibold tabular-nums text-foreground">
                  {formatCell(column, total.values[i])}
                </TableCell>
              ))}
            </TableRow>
          </TableFooter>
        )}
      </Table>
    </>
  )
}
