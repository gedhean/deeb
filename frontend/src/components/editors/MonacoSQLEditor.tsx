import React, { useRef, useEffect } from 'react';
import * as monaco from 'monaco-editor';

// IMPORTANT: Import setup BEFORE any other monaco imports to ensure proper initialization
import {
  updateDatabaseSchema,
  configureMonacoLanguageSettings,
  getLanguageId,
  type SQLDialect,
  type DatabaseSchema,
} from '@/lib/sql-language-setup';

export type { SQLDialect, DatabaseSchema };
export type EditorTheme = 'light' | 'dark';

export interface MonacoSQLEditorProps {
  value: string;
  onChange: (value: string) => void;
  theme?: EditorTheme;
  dialect?: SQLDialect;
  hasError?: boolean;
  readonly?: boolean;
  height?: string | number;
  /** Database schema for enhanced autocompletion */
  schema?: DatabaseSchema;
}

export const MonacoSQLEditor: React.FC<MonacoSQLEditorProps> = ({
  value,
  onChange,
  theme = 'dark',
  dialect = 'mysql',
  hasError = false,
  readonly = false,
  height = 400,
  schema,
}) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const isInitialized = useRef(false);

  // Update database schema when it changes
  useEffect(() => {
    updateDatabaseSchema(dialect, schema);
  }, [dialect, schema]);

  // Initialize Monaco Editor
  useEffect(() => {
    if (!containerRef.current || isInitialized.current) {
      return;
    }

    isInitialized.current = true;

    // Get the correct language ID for the dialect
    const languageId = getLanguageId(dialect);

    // Determine Monaco theme based on light/dark preference
    const monacoTheme = theme === 'dark' ? 'vs-dark' : 'vs';

    // Create Monaco Editor instance directly
    const editor = monaco.editor.create(containerRef.current, {
      value: value,
      language: languageId,
      theme: monacoTheme,
      // Learn more: https://github.com/microsoft/monaco-editor/issues/311
      renderValidationDecorations: 'on',
      // Learn more: https://github.com/microsoft/monaco-editor/issues/4270
      accessibilitySupport: 'off',
      autoClosingQuotes: 'never',
      detectIndentation: false,
      readOnly: readonly,
      minimap: { enabled: false },
      fontSize: 14,
      lineNumbers: 'on',
      roundedSelection: true,
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fixedOverflowWidgets: true,
      tabSize: 2,
      insertSpaces: true,
      wordWrap: 'on',
      wrappingIndent: 'indent',
      suggest: {
        showKeywords: true,
        showSnippets: true,
        showFunctions: true,
        snippetsPreventQuickSuggestions: false,
      },
      quickSuggestions: {
        other: true,
        comments: false,
        strings: false,
      },
      suggestOnTriggerCharacters: true,
      acceptSuggestionOnEnter: 'on',
      tabCompletion: 'on',
      parameterHints: {
        enabled: true,
      },
      formatOnPaste: true,
      formatOnType: true,
      folding: false,
      foldingStrategy: 'indentation',
      showFoldingControls: 'always',
      bracketPairColorization: {
        enabled: true,
      },
      padding: {
        top: 8,
        bottom: 8,
      },
      scrollbar: {
        vertical: 'visible',
        horizontal: 'visible',
        useShadows: false,
        verticalScrollbarSize: 10,
        horizontalScrollbarSize: 10,
        alwaysConsumeMouseWheel: false,
      },
      glyphMargin: false,
      inlineSuggest: {
        showToolbar: 'never',
      },
      renderLineHighlight: 'none',
    });

    editorRef.current = editor;

    // Configure Monaco language settings (brackets, comments, etc.)
    configureMonacoLanguageSettings(dialect, monaco);

    /**
     * Allow code completion when typing in snippets.
     *
     * You can also set configurations when creating monaco-editor instance
     */
    monaco.editor.onDidCreateEditor((editor) => {
      editor.updateOptions({
        suggest: {
          snippetsPreventQuickSuggestions: false,
        },
      });
    });

    // Listen to content changes
    const disposable = editor.onDidChangeModelContent(() => {
      const newValue = editor.getValue();
      onChange(newValue);
    });

    // Cleanup on unmount
    return () => {
      disposable.dispose();
      editor.dispose();
      isInitialized.current = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update value when prop changes
  useEffect(() => {
    if (editorRef.current && editorRef.current.getValue() !== value) {
      const model = editorRef.current.getModel();
      if (model) {
        editorRef.current.pushUndoStop();
        editorRef.current.executeEdits('', [
          {
            range: model.getFullModelRange(),
            text: value,
            forceMoveMarkers: true,
          },
        ]);
        editorRef.current.pushUndoStop();
      }
    }
  }, [value]);

  // Update theme when prop changes
  useEffect(() => {
    if (editorRef.current) {
      const monacoTheme = theme === 'dark' ? 'vs-dark' : 'vs';
      monaco.editor.setTheme(monacoTheme);
    }
  }, [theme]);

  // Update language when dialect changes
  useEffect(() => {
    if (editorRef.current) {
      const languageId = getLanguageId(dialect);
      const model = editorRef.current.getModel();
      if (model) {
        monaco.editor.setModelLanguage(model, languageId);
        configureMonacoLanguageSettings(dialect, monaco);
      }
    }
  }, [dialect]);

  // Update readonly when prop changes
  useEffect(() => {
    if (editorRef.current) {
      editorRef.current.updateOptions({ readOnly: readonly });
    }
  }, [readonly]);

  return (
    <div
      className={`rounded-md border ${hasError ? 'border-destructive' : 'border-input'} overflow-hidden`}
    >
      <div
        ref={containerRef}
        style={{ height: typeof height === 'number' ? `${height}px` : height, width: '100%' }}
      />
    </div>
  );
};
