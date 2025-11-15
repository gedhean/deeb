/**
 * Custom CompletionService for Monaco SQL Languages
 * Provides enhanced autocompletion with database schema context
 */

import type {
  CompletionService,
  ICompletionItem,
} from 'monaco-sql-languages/esm/monaco.contribution';
import { languages } from 'monaco-editor';
import { EntityContextType } from 'monaco-sql-languages/esm/main';

/**
 * Database schema metadata for autocompletion
 */
export interface DatabaseSchema {
  tables: TableMetadata[];
  views?: ViewMetadata[];
}

export interface TableMetadata {
  name: string;
  schema?: string;
  columns: ColumnMetadata[];
  comment?: string;
}

export interface ViewMetadata {
  name: string;
  schema?: string;
  columns: ColumnMetadata[];
  comment?: string;
}

export interface ColumnMetadata {
  name: string;
  type: string;
  comment?: string;
  nullable?: boolean;
  defaultValue?: string;
}

/**
 * Create a CompletionService with database schema context
 */
export function createCompletionService(schema?: DatabaseSchema): CompletionService {
  return async (model, position, _completionContext, suggestions, _entities, snippets) => {
    const completionItems: ICompletionItem[] = [];

    // If no suggestions from parser, return empty
    if (!suggestions) {
      return completionItems;
    }

    const { keywords, syntax } = suggestions;
    const word = model.getWordUntilPosition(position);
    const range = {
      startLineNumber: position.lineNumber,
      endLineNumber: position.lineNumber,
      startColumn: word.startColumn,
      endColumn: word.endColumn,
    };

    // Add keyword completions
    if (keywords && keywords.length > 0) {
      const keywordItems = keywords.map((keyword) => ({
        label: keyword,
        kind: languages.CompletionItemKind.Keyword,
        insertText: keyword,
        range,
        detail: 'SQL Keyword',
        sortText: `2${keyword}`, // Sort keywords with priority 2
      }));
      completionItems.push(...keywordItems);
    }

    // Add syntax-aware completions
    if (syntax && syntax.length > 0) {
      for (const syntaxItem of syntax) {
        const { syntaxContextType } = syntaxItem;

        // Handle table suggestions
        if (
          syntaxContextType === EntityContextType.TABLE ||
          syntaxContextType === EntityContextType.TABLE_CREATE
        ) {
          if (schema?.tables) {
            const tableItems = schema.tables.map((table) => {
              const tableName = table.schema ? `${table.schema}.${table.name}` : table.name;
              return {
                label: tableName,
                kind: languages.CompletionItemKind.Class,
                insertText: table.name,
                range,
                detail: 'Table',
                documentation: table.comment || `Table: ${tableName}`,
                sortText: `1${table.name}`, // Sort tables with highest priority
              };
            });
            completionItems.push(...tableItems);
          }

          // Add views as well
          if (schema?.views) {
            const viewItems = schema.views.map((view) => {
              const viewName = view.schema ? `${view.schema}.${view.name}` : view.name;
              return {
                label: viewName,
                kind: languages.CompletionItemKind.Interface,
                insertText: view.name,
                range,
                detail: 'View',
                documentation: view.comment || `View: ${viewName}`,
                sortText: `1${view.name}`, // Sort views with highest priority
              };
            });
            completionItems.push(...viewItems);
          }
        }

        // Handle column suggestions
        if (syntaxContextType === EntityContextType.COLUMN) {
          if (schema?.tables) {
            // Collect all columns from all tables
            const columnItems: ICompletionItem[] = [];
            for (const table of schema.tables) {
              for (const column of table.columns) {
                const detail = `${column.type}${column.nullable === false ? ' NOT NULL' : ''}`;
                const documentation = column.comment
                  ? `${column.comment}\n\nTable: ${table.name}\nType: ${detail}`
                  : `Table: ${table.name}\nType: ${detail}`;

                columnItems.push({
                  label: column.name,
                  kind: languages.CompletionItemKind.Field,
                  insertText: column.name,
                  range,
                  detail,
                  documentation,
                  sortText: `1${column.name}`, // Sort columns with highest priority
                });
              }
            }

            // Remove duplicates (same column name from different tables)
            const uniqueColumns = new Map<string, ICompletionItem>();
            for (const item of columnItems) {
              if (!uniqueColumns.has(item.label as string)) {
                uniqueColumns.set(item.label as string, item);
              }
            }

            completionItems.push(...uniqueColumns.values());
          }
        }

        // Handle database suggestions
        if (syntaxContextType === EntityContextType.DATABASE) {
          // Could add database suggestions here if available
          // For now, we'll skip this
        }
      }
    }

    // Add snippets if available (following the working example pattern)
    if (snippets && snippets.length > 0) {
      const snippetItems = snippets.map((snippet) => ({
        label: snippet.label || snippet.prefix,
        kind: languages.CompletionItemKind.Snippet,
        filterText: snippet.prefix,
        insertText: snippet.insertText,
        insertTextRules: languages.CompletionItemInsertTextRule.InsertAsSnippet,
        range,
        detail: snippet.description || 'SQL Snippet',
        documentation: snippet.insertText,
        sortText: `3_${snippet.prefix}`, // Sort snippets after keywords but before other items
      }));
      completionItems.push(...snippetItems);
    }

    return completionItems;
  };
}

/**
 * Create a basic CompletionService without schema context
 * This is useful as a fallback or for testing
 */
export function createBasicCompletionService(): CompletionService {
  return createCompletionService();
}

/**
 * Preprocess SQL code before parsing
 * Can be extended to handle application-specific syntax if needed
 * @param code SQL code to preprocess
 * @returns preprocessed SQL code
 */
export function preprocessCode(code: string): string {
  // Currently no preprocessing needed
  // This function exists to match the monaco-sql-languages API
  return code;
}
