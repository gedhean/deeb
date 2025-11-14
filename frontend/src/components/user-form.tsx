import React from 'react';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export type UserFormMode = 'create' | 'edit';

export interface UserFormData {
  email?: string;
  display_name?: string;
  password?: string;
  organization_role: 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper';
  is_active?: boolean;
}

export interface UserFormErrors {
  email?: string;
  password?: string;
  organization_role?: string;
}

export interface UserFormProps {
  mode: UserFormMode;
  formData: UserFormData;
  formErrors: UserFormErrors;
  onChange: (data: Partial<UserFormData>) => void;
}

export const UserForm: React.FC<UserFormProps> = ({ mode, formData, formErrors, onChange }) => {
  const handleFieldChange = (field: keyof UserFormData, value: string | boolean) => {
    onChange({ [field]: value });
  };

  const handleRoleChange = (value: string) => {
    handleFieldChange(
      'organization_role',
      value as 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper',
    );
  };

  const handleIsActiveChange = (value: string) => {
    handleFieldChange('is_active', value === 'true');
  };

  return (
    <div className="space-y-4 py-4">
      {/* Display Name Field */}
      <div className="space-y-2">
        <label htmlFor="user-display-name" className="text-sm font-medium">
          Display Name
        </label>
        <Input
          id="user-display-name"
          type="text"
          value={formData.display_name || ''}
          onChange={(e) => handleFieldChange('display_name', e.target.value)}
          placeholder="John Doe"
        />
      </div>

      {/* Email Field */}
      <div className="space-y-2">
        <label htmlFor="user-email" className="text-sm font-medium">
          Email {mode === 'create' && '*'}
        </label>
        <Input
          id="user-email"
          type="email"
          value={formData.email || ''}
          onChange={(e) => handleFieldChange('email', e.target.value)}
          placeholder="user@example.com"
          disabled={mode === 'edit'}
          className={mode === 'edit' ? 'bg-muted' : ''}
          aria-invalid={!!formErrors.email}
        />
        {mode === 'edit' && (
          <p className="text-xs text-muted-foreground">Email cannot be changed</p>
        )}
        {formErrors.email && <p className="text-sm text-destructive">{formErrors.email}</p>}
      </div>

      {/* Password Field (only in create mode, optional in edit mode) */}
      {mode === 'create' ? (
        <div className="space-y-2">
          <label htmlFor="user-password" className="text-sm font-medium">
            Password *
          </label>
          <Input
            id="user-password"
            type="password"
            value={formData.password || ''}
            onChange={(e) => handleFieldChange('password', e.target.value)}
            placeholder="Enter password (min 8 characters)"
            aria-invalid={!!formErrors.password}
          />
          {formErrors.password && <p className="text-sm text-destructive">{formErrors.password}</p>}
        </div>
      ) : (
        <div className="space-y-2">
          <label htmlFor="user-password" className="text-sm font-medium">
            Password
          </label>
          <Input
            id="user-password"
            type="password"
            value={formData.password || ''}
            onChange={(e) => handleFieldChange('password', e.target.value)}
            placeholder="Leave blank to keep current password"
            aria-invalid={!!formErrors.password}
          />
          <p className="text-xs text-muted-foreground">
            Leave blank to keep current password. Enter new password to change it.
          </p>
          {formErrors.password && <p className="text-sm text-destructive">{formErrors.password}</p>}
        </div>
      )}

      {/* Organization Role Field */}
      <div className="space-y-2">
        <label htmlFor="user-role" className="text-sm font-medium">
          Organization Role *
        </label>
        <Select value={formData.organization_role} onValueChange={handleRoleChange}>
          <SelectTrigger id="user-role" aria-invalid={!!formErrors.organization_role}>
            <SelectValue placeholder="Select role" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="OrganizationDeveloper">Developer</SelectItem>
            <SelectItem value="OrganizationDBA">DBA</SelectItem>
            <SelectItem value="OrganizationAdmin">Admin</SelectItem>
          </SelectContent>
        </Select>
        {formErrors.organization_role && (
          <p className="text-sm text-destructive">{formErrors.organization_role}</p>
        )}
      </div>

      {/* Status Field (only in edit mode) */}
      {mode === 'edit' && formData.is_active !== undefined && (
        <div className="space-y-2">
          <label htmlFor="user-active" className="text-sm font-medium">
            Status
          </label>
          <Select
            value={formData.is_active ? 'true' : 'false'}
            onValueChange={handleIsActiveChange}
          >
            <SelectTrigger id="user-active">
              <SelectValue placeholder="Select status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="true">Active</SelectItem>
              <SelectItem value="false">Inactive</SelectItem>
            </SelectContent>
          </Select>
        </div>
      )}
    </div>
  );
};
