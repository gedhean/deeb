/**
 * SQL Language Setup for Monaco Editor
 * Configures monaco-sql-languages for MySQL and PostgreSQL support with proper worker routing
 *
 * Based on the working example from monaco-sql-languages/website
 */

// Import contributions FIRST before anything else
import 'monaco-sql-languages/esm/languages/mysql/mysql.contribution';
import 'monaco-sql-languages/esm/languages/pgsql/pgsql.contribution';

import { LanguageIdEnum, setupLanguageFeatures } from 'monaco-sql-languages';
import type * as Monaco from 'monaco-editor';
import {
  createCompletionService,
  preprocessCode,
  type DatabaseSchema,
} from './sql-completion-service';

// STEP 2: Import workers using Vite's worker query syntax
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import MySQLWorker from 'monaco-sql-languages/esm/languages/mysql/mysql.worker?worker';
import PgSQLWorker from 'monaco-sql-languages/esm/languages/pgsql/pgsql.worker?worker';

export type SQLDialect = 'mysql' | 'pgsql';
export type { DatabaseSchema } from './sql-completion-service';

// STEP 3: Configure MonacoEnvironment to route language workers
// This must be done before Monaco Editor is initialized
(globalThis as typeof globalThis & { MonacoEnvironment: unknown }).MonacoEnvironment = {
  getWorker(_: unknown, label: string) {
    if (label === LanguageIdEnum.MYSQL) {
      return new MySQLWorker();
    }
    if (label === LanguageIdEnum.PG) {
      return new PgSQLWorker();
    }
    // Default to the base editor worker for other languages
    return new EditorWorker();
  },
};

// Store the current schema for dynamic updates
let currentMySQLSchema: DatabaseSchema | undefined;
let currentPgSQLSchema: DatabaseSchema | undefined;

// Create a shared completion service instance (like the working example does)
const sharedCompletionService = createCompletionService();

// STEP 4: Setup language features for MySQL and PostgreSQL
setupLanguageFeatures(LanguageIdEnum.MYSQL, {
  completionItems: {
    enable: true,
    completionService: sharedCompletionService,
  },
  preprocessCode,
});

setupLanguageFeatures(LanguageIdEnum.PG, {
  completionItems: {
    enable: true,
    completionService: sharedCompletionService,
  },
  preprocessCode,
});

/**
 * Update the schema for a specific SQL dialect
 * This will reconfigure the completion service with new schema data
 * @param dialect - The SQL dialect to update
 * @param schema - The new database schema
 */
export function updateDatabaseSchema(dialect: SQLDialect, schema?: DatabaseSchema) {
  // Store the schema for the specific dialect
  if (dialect === 'mysql') {
    currentMySQLSchema = schema;
  } else {
    currentPgSQLSchema = schema;
  }

  // Get the appropriate schema based on dialect
  const dialectSchema = dialect === 'mysql' ? currentMySQLSchema : currentPgSQLSchema;

  // Create a new completion service with the updated schema
  const completionService = createCompletionService(dialectSchema);
  const languageId = getLanguageId(dialect);

  // Reconfigure language features with the new completion service
  setupLanguageFeatures(languageId, {
    completionItems: {
      enable: true,
      completionService,
    },
    diagnostics: true,
    definitions: true,
    references: true,
  });
}

/**
 * Configure additional Monaco language settings
 * This should be called in the editor mount handler
 * @param dialect - The SQL dialect to configure
 * @param monaco - Monaco editor instance
 */
export function configureMonacoLanguageSettings(dialect: SQLDialect, monaco: typeof Monaco) {
  const languageId = getLanguageId(dialect);

  // Configure additional language settings
  monaco.languages.setLanguageConfiguration(languageId, {
    comments: {
      lineComment: '--',
      blockComment: ['/*', '*/'],
    },
    brackets: [
      ['(', ')'],
      ['[', ']'],
    ],
    autoClosingPairs: [
      { open: '(', close: ')' },
      { open: '[', close: ']' },
      { open: "'", close: "'", notIn: ['string', 'comment'] },
      { open: '"', close: '"', notIn: ['string', 'comment'] },
      { open: '`', close: '`', notIn: ['string', 'comment'] },
    ],
    surroundingPairs: [
      { open: '(', close: ')' },
      { open: '[', close: ']' },
      { open: "'", close: "'" },
      { open: '"', close: '"' },
      { open: '`', close: '`' },
    ],
  });
}

/**
 * Get the Monaco language ID for a given SQL dialect
 * @param dialect - The SQL dialect
 * @returns The Monaco language ID
 */
export function getLanguageId(dialect: SQLDialect): LanguageIdEnum {
  switch (dialect) {
    case 'mysql':
      return LanguageIdEnum.MYSQL;
    case 'pgsql':
      return LanguageIdEnum.PG;
    default:
      // This should never happen due to TypeScript type checking
      // But we provide a safe fallback
      return LanguageIdEnum.MYSQL;
  }
}
