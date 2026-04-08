import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type Row,
} from "@tanstack/react-table";

import { cn } from "@/lib/utils.ts";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";

type DataTableColumnMeta = {
  headClassName?: string;
  cellClassName?: string;
};

type DataTableProps<TData, TValue> = {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  emptyMessage?: string;
  stickyHeader?: boolean;
  wrapperClassName?: string;
  tableWrapperClassName?: string;
  tableHeaderClassName?: string;
  tableClassName?: string;
  rowClassName?: (row: Row<TData>) => string | undefined;
};

export function DataTable<TData, TValue>({
  columns,
  data,
  emptyMessage = "No results.",
  stickyHeader = false,
  wrapperClassName,
  tableWrapperClassName,
  tableHeaderClassName,
  tableClassName,
  rowClassName,
}: DataTableProps<TData, TValue>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
		<div className={cn("overflow-hidden", wrapperClassName)}>
			<Table
				wrapperClassName={tableWrapperClassName}
				className={tableClassName}
			>
				<TableHeader
					className={cn(
						stickyHeader &&
							"sticky top-0 z-10 bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/80",
						tableHeaderClassName,
					)}
				>
					{table.getHeaderGroups().map((headerGroup) => (
						<TableRow key={headerGroup.id}>
							{headerGroup.headers.map((header) => {
								const meta = header.column.columnDef.meta as
									| DataTableColumnMeta
									| undefined;

								return (
									<TableHead
										key={header.id}
										className={cn(
											stickyHeader &&
												"sticky top-0 z-10 bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/80",
											meta?.headClassName,
										)}
									>
										{header.isPlaceholder
											? null
											: flexRender(
													header.column.columnDef.header,
													header.getContext(),
												)}
									</TableHead>
								);
							})}
						</TableRow>
					))}
				</TableHeader>
				<TableBody>
					{table.getRowModel().rows.length > 0 ? (
						table.getRowModel().rows.map((row) => (
							<TableRow
								key={row.id}
								className={cn(
									rowClassName?.(row),
									"border-b border-b-muted/30",
								)}
							>
								{row.getVisibleCells().map((cell) => {
									const meta = cell.column.columnDef.meta as
										| DataTableColumnMeta
										| undefined;

									return (
										<TableCell
											key={cell.id}
											className={cn(meta?.cellClassName)}
										>
											{flexRender(
												cell.column.columnDef.cell,
												cell.getContext(),
											)}
										</TableCell>
									);
								})}
							</TableRow>
						))
					) : (
						<TableRow>
							<TableCell
								colSpan={columns.length}
								className="h-24 text-center text-muted-foreground"
							>
								{emptyMessage}
							</TableCell>
						</TableRow>
					)}
				</TableBody>
			</Table>
		</div>
	);
}
