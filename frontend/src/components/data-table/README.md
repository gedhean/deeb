# DataTable Component

A reusable data table component built with [TanStack Table](https://tanstack.com/table/latest) that provides pagination, sorting, filtering, column visibility, and row selection out of the box.

## Features

- ✅ **Pagination**: Client-side and server-side pagination support
- ✅ **Sorting**: Click column headers to sort
- ✅ **Filtering**: Built-in search/filter functionality
- ✅ **Column Visibility**: Toggle which columns to show
- ✅ **Row Selection**: Optional checkbox selection
- ✅ **Loading State**: Skeleton loaders during data fetch
- ✅ **Empty State**: Customizable empty message
- ✅ **Responsive**: Works on all screen sizes
- ✅ **Type-safe**: Full TypeScript support

## Installation

The DataTable component uses `@tanstack/react-table` which should already be installed:

```bash
pnpm add @tanstack/react-table
```

## Basic Usage

### 1. Define Your Columns

Create a column definition file (e.g., `columns.tsx`):

```tsx
import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';

export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  created_at: string;
};

export const columns: ColumnDef<User>[] = [
  {
    accessorKey: 'email',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,
  },
  {
    accessorKey: 'name',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
  },
  {
    accessorKey: 'role',
    header: 'Role',
    enableSorting: false,
  },
  {
    accessorKey: 'created_at',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      return new Date(row.getValue('created_at')).toLocaleDateString();
    },
  },
];
```

### 2. Use the DataTable Component

```tsx
import { DataTable } from '@/components/data-table';
import { columns } from './columns';

function UsersPage() {
  const { users, isLoading } = useUsers();

  return (
    <DataTable
      columns={columns}
      data={users}
      isLoading={isLoading}
      searchable
      searchKey="email"
      searchPlaceholder="Search by email..."
    />
  );
}
```

## Advanced Usage

### Server-Side Pagination

```tsx
function UsersPage() {
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);

  const { users, totalCount, isLoading } = useUsers({
    limit: pageSize,
    offset: pageIndex * pageSize,
  });

  return (
    <DataTable
      columns={columns}
      data={users}
      isLoading={isLoading}
      pagination={{
        pageIndex,
        pageSize,
        totalCount,
        onPageChange: setPageIndex,
        onPageSizeChange: setPageSize,
        pageSizeOptions: [10, 20, 50, 100],
      }}
    />
  );
}
```

### Row Selection

```tsx
function UsersPage() {
  const { users } = useUsers();
  const [selectedUsers, setSelectedUsers] = useState<User[]>([]);

  return (
    <DataTable
      columns={columns}
      data={users}
      enableRowSelection
      onRowSelectionChange={setSelectedUsers}
    />
  );
}
```

### Custom Actions Column

```tsx
import { MoreHorizontal, Pencil, Trash } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export const columns: ColumnDef<User>[] = [
  // ... other columns
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const user = row.original;

      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => handleEdit(user)}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => handleDelete(user)}>
              <Trash className="mr-2 h-4 w-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    },
  },
];
```

### Custom Cell Rendering

```tsx
{
  accessorKey: 'status',
  header: 'Status',
  cell: ({ row }) => {
    const status = row.getValue('status') as string;
    return (
      <Badge variant={status === 'active' ? 'success' : 'secondary'}>
        {status}
      </Badge>
    );
  },
}
```

## API Reference

### DataTableProps

| Prop                   | Type                              | Default         | Description                     |
| ---------------------- | --------------------------------- | --------------- | ------------------------------- |
| `columns`              | `ColumnDef<TData, TValue>[]`      | **required**    | Column definitions              |
| `data`                 | `TData[]`                         | **required**    | Table data                      |
| `isLoading`            | `boolean`                         | `false`         | Show loading skeletons          |
| `pagination`           | `DataTablePaginationOptions`      | `undefined`     | Pagination configuration        |
| `searchable`           | `boolean`                         | `false`         | Enable search/filter            |
| `searchPlaceholder`    | `string`                          | `"Search..."`   | Placeholder for search input    |
| `searchKey`            | `string`                          | `undefined`     | Column key to search on         |
| `showColumnVisibility` | `boolean`                         | `true`          | Show column visibility dropdown |
| `emptyMessage`         | `string`                          | `"No results."` | Message when no data            |
| `enableRowSelection`   | `boolean`                         | `false`         | Enable checkbox row selection   |
| `onRowSelectionChange` | `(selectedRows: TData[]) => void` | `undefined`     | Callback when selection changes |

### DataTablePaginationOptions

| Prop               | Type                          | Default                | Description                              |
| ------------------ | ----------------------------- | ---------------------- | ---------------------------------------- |
| `pageIndex`        | `number`                      | **required**           | Current page (0-based)                   |
| `pageSize`         | `number`                      | **required**           | Items per page                           |
| `totalCount`       | `number`                      | `undefined`            | Total items (for server-side pagination) |
| `onPageChange`     | `(pageIndex: number) => void` | `undefined`            | Callback when page changes               |
| `onPageSizeChange` | `(pageSize: number) => void`  | `undefined`            | Callback when page size changes          |
| `pageSizeOptions`  | `number[]`                    | `[10, 20, 30, 40, 50]` | Available page sizes                     |

## Column Definition

Columns are defined using TanStack Table's `ColumnDef` type:

```tsx
{
  // Accessor key (matches data property)
  accessorKey: 'email',

  // Header (static or function)
  header: 'Email',
  // or
  header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,

  // Cell renderer
  cell: ({ row }) => row.getValue('email'),

  // Enable/disable sorting
  enableSorting: true,

  // Enable/disable hiding
  enableHiding: true,
}
```

## Best Practices

1. **Extract column definitions** into separate files for reusability
2. **Use `DataTableColumnHeader`** for sortable columns
3. **Disable sorting/hiding** for action columns
4. **Use `row.original`** to access the full row data in cells
5. **Implement server-side pagination** for large datasets (>1000 rows)
6. **Show loading states** during data fetching
7. **Provide meaningful empty states** with clear next actions

## Examples

See the following pages for complete examples:

- Users Page
- Database Instances Page
- Change Plans Page
- Audit Log Page

## Troubleshooting

### Columns not sorting

Make sure `enableSorting: true` in column definition and use `DataTableColumnHeader`.

### Search not working

Ensure `searchable={true}` and `searchKey` prop is set to a valid column accessor key.

### Pagination not updating

Check that `onPageChange` callback is properly updating the page index state.

### TypeScript errors

Ensure your data type `TData` matches the column accessorKey values.
