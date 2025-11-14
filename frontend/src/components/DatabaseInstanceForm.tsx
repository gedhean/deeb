import React from 'react';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { Environment, DatabaseEngine } from '@/lib/api/types';

export type DatabaseInstanceFormMode = 'create' | 'edit';

export interface DatabaseInstanceFormData {
  name: string;
  description?: string;
  database_engine: DatabaseEngine;
  host: string;
  port: number;
  username: string;
  password: string;
  use_rds_iam_auth: boolean;
  require_ssl: boolean;
  environment_id?: string;
}

export interface DatabaseInstanceFormErrors {
  name?: string;
  host?: string;
  port?: string;
  username?: string;
  password?: string;
}

export interface DatabaseInstanceFormProps {
  mode: DatabaseInstanceFormMode;
  formData: DatabaseInstanceFormData;
  formErrors: DatabaseInstanceFormErrors;
  environments: Environment[];
  onChange: (data: Partial<DatabaseInstanceFormData>) => void;
  onDatabaseEngineChange?: (engine: DatabaseEngine) => void;
}

export const DatabaseInstanceForm: React.FC<DatabaseInstanceFormProps> = ({
  mode,
  formData,
  formErrors,
  environments,
  onChange,
  onDatabaseEngineChange,
}) => {
  const handleFieldChange = (
    field: keyof DatabaseInstanceFormData,
    value: string | number | boolean | undefined,
  ) => {
    onChange({ [field]: value });
  };

  const handleDatabaseEngineChange = (value: string) => {
    const engine = value as DatabaseEngine;
    handleFieldChange('database_engine', engine);

    // Auto-update port based on database engine (only in create mode)
    if (mode === 'create' && onDatabaseEngineChange) {
      onDatabaseEngineChange(engine);
    }
  };

  const handlePortChange = (value: string) => {
    const port = parseInt(value) || 0;
    handleFieldChange('port', port);
  };

  const fieldPrefix = mode === 'create' ? 'create' : 'edit';

  return (
    <div className="space-y-4 py-4">
      {/* Instance Name Field */}
      <div className="space-y-2">
        <label htmlFor={`${fieldPrefix}-name`} className="text-sm font-medium">
          Instance Name *
        </label>
        <Input
          id={`${fieldPrefix}-name`}
          value={formData.name}
          onChange={(e) => handleFieldChange('name', e.target.value)}
          placeholder="Enter instance name"
          aria-invalid={!!formErrors.name}
        />
        {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
      </div>

      {/* Description Field */}
      <div className="space-y-2">
        <label htmlFor={`${fieldPrefix}-description`} className="text-sm font-medium">
          Description
        </label>
        <Textarea
          id={`${fieldPrefix}-description`}
          value={formData.description || ''}
          onChange={(e) => handleFieldChange('description', e.target.value)}
          placeholder="Enter instance description"
          className="min-h-[80px]"
        />
      </div>

      {/* Database Engine Field */}
      <div className="space-y-2">
        <label htmlFor={`${fieldPrefix}-type`} className="text-sm font-medium">
          Database Engine *
        </label>
        <Select value={formData.database_engine} onValueChange={handleDatabaseEngineChange}>
          <SelectTrigger id={`${fieldPrefix}-type`}>
            <SelectValue placeholder="Select database engine" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="postgres">PostgreSQL</SelectItem>
            <SelectItem value="mysql">MySQL</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Environment Field */}
      <div className="space-y-2">
        <label htmlFor={`${fieldPrefix}-environment`} className="text-sm font-medium">
          Environment (optional)
        </label>
        <Select
          value={formData.environment_id || ''}
          onValueChange={(value) =>
            handleFieldChange('environment_id', value === '' ? undefined : value)
          }
        >
          <SelectTrigger id={`${fieldPrefix}-environment`}>
            <SelectValue placeholder="No Environment" />
          </SelectTrigger>
          <SelectContent>
            {environments.map((env) => (
              <SelectItem key={env.id} value={env.id}>
                {env.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Host and Port Fields */}
      <div className={mode === 'create' ? 'grid grid-cols-4 gap-4' : 'grid grid-cols-2 gap-4'}>
        <div className={mode === 'create' ? 'space-y-2 col-span-3' : 'space-y-2'}>
          <label htmlFor={`${fieldPrefix}-host`} className="text-sm font-medium">
            Host *
          </label>
          <Input
            id={`${fieldPrefix}-host`}
            value={formData.host}
            onChange={(e) => handleFieldChange('host', e.target.value)}
            placeholder="Enter database host"
            aria-invalid={!!formErrors.host}
          />
          {formErrors.host && <p className="text-sm text-destructive">{formErrors.host}</p>}
        </div>
        <div className={mode === 'create' ? 'space-y-2 col-span-1' : 'space-y-2'}>
          <label htmlFor={`${fieldPrefix}-port`} className="text-sm font-medium">
            Port *
          </label>
          <Input
            id={`${fieldPrefix}-port`}
            type="number"
            value={formData.port}
            onChange={(e) => handlePortChange(e.target.value)}
            placeholder="Enter database port"
            min={1}
            max={65535}
            aria-invalid={!!formErrors.port}
          />
          {formErrors.port && <p className="text-sm text-destructive">{formErrors.port}</p>}
        </div>
      </div>

      {/* Username Field */}
      <div className="space-y-2">
        <label htmlFor={`${fieldPrefix}-username`} className="text-sm font-medium">
          Username *
        </label>
        <Input
          id={`${fieldPrefix}-username`}
          value={formData.username}
          onChange={(e) => handleFieldChange('username', e.target.value)}
          placeholder="Enter database username"
          aria-invalid={!!formErrors.username}
        />
        {formErrors.username && <p className="text-sm text-destructive">{formErrors.username}</p>}
      </div>

      {/* Password Field */}
      <div className="space-y-2">
        <label htmlFor={`${fieldPrefix}-password`} className="text-sm font-medium">
          {mode === 'create'
            ? 'Password (optional)'
            : 'Password (optional, leave blank to keep current)'}
        </label>
        <Input
          id={`${fieldPrefix}-password`}
          type="password"
          value={formData.password}
          onChange={(e) => handleFieldChange('password', e.target.value)}
          placeholder={
            mode === 'create'
              ? 'Enter database password'
              : 'Enter new password (leave blank to keep current)'
          }
          disabled={formData.use_rds_iam_auth}
          aria-invalid={!!formErrors.password}
        />
        {formErrors.password && <p className="text-sm text-destructive">{formErrors.password}</p>}
      </div>

      {/* RDS IAM Authentication Checkbox */}
      <div className="space-y-2">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={formData.use_rds_iam_auth}
            onChange={(e) => handleFieldChange('use_rds_iam_auth', e.target.checked)}
            className="h-4 w-4 rounded border-gray-300"
          />
          <span className="text-sm font-medium">Use RDS IAM Authentication</span>
        </label>
      </div>

      {/* Require SSL Checkbox */}
      <div className="space-y-2">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={formData.require_ssl}
            onChange={(e) => handleFieldChange('require_ssl', e.target.checked)}
            className="h-4 w-4 rounded border-gray-300"
          />
          <span className="text-sm font-medium">Require SSL</span>
        </label>
      </div>
    </div>
  );
};
