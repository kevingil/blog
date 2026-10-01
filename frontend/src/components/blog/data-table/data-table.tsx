import * as React from "react";
import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
  OnChangeFn,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { DataTablePagination } from "./data-table-pagination";
import { DataTableToolbar } from "./data-table-toolbar";
import { cn } from "@/lib/utils";

const COMPACT_HIDDEN_COLUMNS = new Set([
  "tags",
  "article.created_at",
  "article.published_at",
  "status",
]);

function compactColumnClass(id: string) {
  if (COMPACT_HIDDEN_COLUMNS.has(id)) {
    return "hidden w-0 max-w-0 overflow-hidden border-0 p-0 md:table-cell md:w-auto md:max-w-none md:overflow-visible md:p-2";
  }
  if (id === "article.draft_title") {
    return "max-w-0 whitespace-normal md:max-w-none";
  }
  if (id === "actions") {
    return "w-10 px-1";
  }
  return undefined;
}

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  totalPages: number;
  currentPage: number;
  onPageChange: (page: number) => void;
  onSortingChange: OnChangeFn<SortingState>;
  sorting: SortingState;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  statusFilter: "all" | "published" | "drafts";
  onStatusFilterChange: (value: "all" | "published" | "drafts") => void;
  isLoading?: boolean;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  totalPages,
  currentPage,
  onPageChange,
  onSortingChange,
  sorting,
  searchQuery,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  isLoading = false,
}: DataTableProps<TData, TValue>) {
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    []
  );
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
    },
    enableRowSelection: true,
    onSortingChange,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualPagination: true,
    manualSorting: true,
    pageCount: totalPages,
  });

  return (
    <div className="flex flex-col h-full min-h-0 min-w-0">
      <DataTableToolbar
        table={table}
        searchQuery={searchQuery}
        onSearchChange={onSearchChange}
        statusFilter={statusFilter}
        onStatusFilterChange={onStatusFilterChange}
      />
      <div className="relative min-h-0 min-w-0 flex-1 overflow-y-auto overflow-x-hidden rounded-md border border-border bg-card md:overflow-x-auto">
        <Table noWrapper className="table-fixed md:table-auto">
          <TableHeader className="sticky top-0 bg-muted/60 z-10 border-b border-border">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id} className={cn("bg-muted/60 text-foreground", compactColumnClass(header.column.id))}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  Loading...
                </TableCell>
              </TableRow>
            ) : table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && "selected"}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className={compactColumnClass(cell.column.id)}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <DataTablePagination
        table={table}
        totalPages={totalPages}
        currentPage={currentPage}
        onPageChange={onPageChange}
      />
    </div>
  );
}
