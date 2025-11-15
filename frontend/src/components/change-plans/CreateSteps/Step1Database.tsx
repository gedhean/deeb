import React, { useState } from 'react';
import { Database as DatabaseIcon, ExternalLink, Check, ChevronsUpDown } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Field,
  FieldLabel,
  FieldDescription,
  FieldError,
  FieldContent,
} from '@/components/ui/field';
import { cn } from '@/lib/utils';
import type { Database } from '@/lib/api/types';

export interface Step1DatabaseProps {
  databases: Database[];
  isLoadingDatabases: boolean;
  selectedDatabaseId: string;
  onDatabaseChange: (databaseId: string) => void;
  onNext: () => void;
  onCancel: () => void;
  error?: string;
}

export const Step1Database: React.FC<Step1DatabaseProps> = ({
  databases,
  isLoadingDatabases,
  selectedDatabaseId,
  onDatabaseChange,
  onNext,
  onCancel,
  error,
}) => {
  const [open, setOpen] = useState(false);
  const selectedDatabase = databases.find((db) => db.id === selectedDatabaseId);
  const isValid = !!selectedDatabaseId;

  return (
    <div className="space-y-6">
      {/* Database Selection */}
      <Field data-invalid={!!error}>
        <FieldContent>
          <FieldLabel>Target Database</FieldLabel>
          <FieldDescription>
            Choose the database where you want to apply the changes
          </FieldDescription>
          {isLoadingDatabases ? (
            <Skeleton className="h-10 w-full" />
          ) : (
            <Popover open={open} onOpenChange={setOpen}>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  role="combobox"
                  aria-expanded={open}
                  aria-invalid={!!error}
                  className={cn(
                    'w-full justify-between',
                    !selectedDatabaseId && 'text-muted-foreground',
                    error && 'border-destructive',
                  )}
                >
                  {selectedDatabaseId ? (
                    <div className="flex items-center gap-2 truncate">
                      <DatabaseIcon className="h-4 w-4 shrink-0" />
                      <span className="truncate">{selectedDatabase?.database_name}</span>
                    </div>
                  ) : databases.length === 0 ? (
                    'No databases available'
                  ) : (
                    'Select a database...'
                  )}
                  <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent
                className="p-0"
                align="start"
                style={{ width: 'var(--radix-popover-trigger-width)' }}
              >
                <Command>
                  <CommandInput placeholder="Search databases..." />
                  <CommandList>
                    <CommandEmpty>No database found.</CommandEmpty>
                    <CommandGroup>
                      {databases.map((database) => {
                        const instanceName = database.database_instance?.name || 'Unknown Instance';
                        const envName = database.environment?.name || 'Unknown Environment';
                        return (
                          <CommandItem
                            key={database.id}
                            value={`${database.database_name} ${instanceName} ${envName}`}
                            onSelect={() => {
                              onDatabaseChange(database.id);
                              setOpen(false);
                            }}
                          >
                            <Check
                              className={cn(
                                'h-4 w-4',
                                selectedDatabaseId === database.id ? 'opacity-100' : 'opacity-0',
                              )}
                            />
                            <div className="flex items-center gap-2 flex-1 min-w-0 font-mono">
                              <DatabaseIcon className="h-4 w-4 text-muted-foreground shrink-0" />
                              <span className="text-muted-foreground">{envName}</span>
                              <span className="text-muted-foreground">{'>'}</span>
                              <span className="text-muted-foreground">{instanceName}</span>
                              <span className="text-muted-foreground">{'>'}</span>
                              <span className="font-medium">{database.database_name}</span>
                            </div>
                          </CommandItem>
                        );
                      })}
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          )}
          <FieldError id="database-error">{error}</FieldError>
        </FieldContent>
      </Field>

      {/* Selected Database Details */}
      {selectedDatabase && (
        <Card>
          <CardContent className="pt-4 pb-4">
            <div className="divide-y">
              {/* Database Name with link */}
              <div className="grid grid-cols-3 gap-4 py-2 items-center">
                <span className="text-sm font-medium text-muted-foreground">Database</span>
                <div className="col-span-2">
                  <Link
                    to={`/databases/${selectedDatabase.id}`}
                    target="_blank"
                    className="text-sm text-primary hover:underline inline-flex items-center gap-1"
                  >
                    {selectedDatabase.database_name}
                    <ExternalLink className="h-3 w-3" />
                  </Link>
                </div>
              </div>

              {/* Database Instance with link */}
              <div className="grid grid-cols-3 gap-4 py-2 items-center">
                <span className="text-sm font-medium text-muted-foreground">Instance</span>
                <div className="col-span-2">
                  {selectedDatabase.database_instance?.id ? (
                    <Link
                      to={`/database-instances/${selectedDatabase.database_instance.id}`}
                      target="_blank"
                      className="text-sm text-primary hover:underline inline-flex items-center gap-1"
                    >
                      {selectedDatabase.database_instance.name || 'Unknown'}
                      <ExternalLink className="h-3 w-3" />
                    </Link>
                  ) : (
                    <span className="text-sm text-foreground">
                      {selectedDatabase.database_instance?.name || 'Unknown'}
                    </span>
                  )}
                </div>
              </div>

              {/* Environment (no detail page, so no link) */}
              <div className="grid grid-cols-3 gap-4 py-2 items-center">
                <span className="text-sm font-medium text-muted-foreground">Environment</span>
                <div className="col-span-2">
                  <span className="text-sm text-foreground">
                    {selectedDatabase.environment?.name || 'Unknown'}
                  </span>
                </div>
              </div>

              {/* Host:Port */}
              {selectedDatabase.database_instance?.host && (
                <div className="grid grid-cols-3 gap-4 py-2 items-center">
                  <span className="text-sm font-medium text-muted-foreground">Host</span>
                  <div className="col-span-2">
                    <span className="text-sm text-foreground font-mono">
                      {selectedDatabase.database_instance.host}:
                      {selectedDatabase.database_instance.port}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-6 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="button" onClick={onNext} disabled={!isValid}>
          Next: SQL Content
        </Button>
      </div>
    </div>
  );
};
