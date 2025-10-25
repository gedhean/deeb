# Frontend Conventions & Standards

This document outlines the naming conventions, code standards, and best practices for the Deeb frontend application.

## File Naming Conventions

### Components Directory: `frontend/src/components/`

**Use kebab-case for component files**

```
✅ Correct:
frontend/src/components/user-profile.tsx
frontend/src/components/navigation-bar.tsx
frontend/src/components/theme-provider.tsx
frontend/src/components/mode-toggle.tsx
frontend/src/components/protected-route.tsx
frontend/src/components/health-check.tsx

❌ Incorrect:
frontend/src/components/UserProfile.tsx
frontend/src/components/NavigationBar.tsx
frontend/src/components/ThemeProvider.tsx
```

**Examples from our project:**
```
frontend/src/components/
├── ui/                        # Shadcn UI components (kebab-case)
│   ├── button.tsx
│   ├── card.tsx
│   ├── input.tsx
│   ├── dropdown-menu.tsx
│   └── ...
├── theme-provider.tsx         # Custom components (kebab-case)
├── mode-toggle.tsx
├── protected-route.tsx
└── health-check.tsx
```

### Non-Components Directory: `frontend/src/` (pages, contexts, etc.)

**Use PascalCase for component files outside the components directory**

```
✅ Correct:
frontend/src/pages/LoginPage.tsx
frontend/src/pages/DashboardPage.tsx
frontend/src/pages/HealthCheckPage.tsx
frontend/src/contexts/AuthContext.tsx
frontend/src/hooks/UseAuth.tsx

❌ Incorrect:
frontend/src/pages/login-page.tsx
frontend/src/contexts/auth-context.tsx
frontend/src/hooks/use-auth.tsx
```

**Examples from our project:**
```
frontend/src/
├── pages/                     # Pages use PascalCase
│   ├── Home.tsx
│   ├── LoginPage.tsx
│   ├── DashboardPage.tsx
│   └── HealthCheckPage.tsx
├── contexts/                  # Contexts use PascalCase
│   └── AuthContext.tsx
├── hooks/                     # Hooks use PascalCase
│   └── UseAuth.tsx
├── lib/                       # Utilities use kebab-case
│   ├── utils.ts
│   ├── api/
│   │   ├── client.ts
│   │   ├── types.ts
│   │   └── services/
│   │       ├── auth.ts
│   │       └── health.ts
└── App.tsx                    # Root component uses PascalCase
```

## Component Naming Convention Summary

| Location | File Naming | Export Name | Example File | Example Export |
|----------|-------------|-------------|--------------|----------------|
| `src/components/` | kebab-case | PascalCase | `user-profile.tsx` | `export const UserProfile` |
| `src/components/ui/` | kebab-case | PascalCase | `button.tsx` | `export const Button` |
| `src/pages/` | PascalCase | PascalCase | `LoginPage.tsx` | `export const LoginPage` |
| `src/contexts/` | PascalCase | PascalCase | `AuthContext.tsx` | `export const AuthContext` |
| `src/hooks/` | PascalCase | PascalCase | `UseAuth.tsx` | `export const useAuth` |
| `src/lib/` | kebab-case | camelCase | `api-client.ts` | `export const apiClient` |

## React Component Structure

### Component Export Pattern

**Always use named exports with PascalCase:**

```tsx
// ✅ Correct
export const UserProfile: React.FC = () => {
  return <div>User Profile</div>
}

// ❌ Incorrect
export default function userProfile() {
  return <div>User Profile</div>
}
```

### Component Type Annotations

Use `React.FC` with proper TypeScript types:

```tsx
interface UserProfileProps {
  userId: string
  onUpdate?: (user: User) => void
}

export const UserProfile: React.FC<UserProfileProps> = ({ userId, onUpdate }) => {
  // Component implementation
}
```

## Directory Structure Standards

```
frontend/src/
├── components/           # Reusable UI components (kebab-case)
│   ├── ui/              # Shadcn UI components (kebab-case)
│   ├── layout/          # Layout components (kebab-case)
│   └── forms/           # Form components (kebab-case)
├── pages/               # Page components (PascalCase)
├── contexts/            # React contexts (PascalCase)
├── hooks/               # Custom React hooks (PascalCase)
├── lib/                 # Utilities and libraries (kebab-case)
│   ├── api/            # API client and services
│   ├── utils.ts        # Utility functions
│   └── constants.ts    # Constants
├── types/               # TypeScript type definitions (kebab-case)
├── styles/              # Global styles (kebab-case)
└── App.tsx              # Root application component
```

## Import Conventions

### Path Aliases

Use configured path aliases for cleaner imports:

```tsx
// ✅ Correct - using path aliases
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'
import { apiClient } from '@/lib/api/client'

// ❌ Incorrect - relative paths
import { Button } from '../../../components/ui/button'
import { useAuth } from '../../contexts/AuthContext'
```

### Import Order

1. React and external libraries
2. Internal components and hooks
3. Types
4. Styles

```tsx
// 1. React and external libraries
import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

// 2. Internal components and hooks
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'

// 3. Types
import type { User, LoginRequest } from '@/lib/api/types'

// 4. Styles
import './styles.css'
```

## TypeScript Standards

### Type Definitions

- Use `interface` for object shapes
- Use `type` for unions, intersections, or primitives
- Export types from dedicated `types.ts` files or inline with components

```tsx
// Interface for object shapes
interface UserProfileProps {
  user: User
  onEdit: () => void
}

// Type for unions
type Status = 'idle' | 'loading' | 'success' | 'error'

// Type for component props with children
type LayoutProps = {
  children: React.ReactNode
  className?: string
}
```

### File Extensions

- `.tsx` for files containing JSX
- `.ts` for files without JSX (utilities, types, etc.)

## Styling Conventions

### Tailwind CSS

- Use Tailwind utility classes directly in JSX
- Follow our configured theme (neutral base color, new-york style)
- Use CSS variables for theme customization
- Support dark mode with `dark:` prefix

```tsx
// ✅ Correct - Tailwind utilities with dark mode
<div className="bg-white dark:bg-gray-900 p-4 rounded-lg">
  <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
    Title
  </h1>
</div>
```

### CSS Modules (when needed)

Use kebab-case for CSS file names:

```
component-name.module.css
user-profile.module.css
```

## State Management

### Local State

Use `useState` for component-local state:

```tsx
const [email, setEmail] = useState('')
const [isLoading, setIsLoading] = useState(false)
```

### Context API

Use Context for global/shared state (e.g., AuthContext):

```tsx
import { createContext, useContext } from 'react'

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
```

## Error Handling

### Error Display

Use consistent error message patterns:

```tsx
const [error, setError] = useState('')

// Display errors
{error && (
  <div className="p-3 text-sm text-red-600 bg-red-50 dark:bg-red-900/20 dark:text-red-400 rounded-md">
    {error}
  </div>
)}
```

### API Error Handling

```tsx
try {
  await apiCall()
} catch (err) {
  if (err instanceof APIError) {
    setError(err.message)
  } else {
    setError('An unexpected error occurred')
  }
}
```

## Form Conventions

### Form Validation

- Client-side validation with clear error messages
- Validate on blur for better UX
- Show inline errors below fields

```tsx
const [emailError, setEmailError] = useState('')

const validateEmail = (value: string): boolean => {
  if (!value) {
    setEmailError('Email is required')
    return false
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    setEmailError('Invalid email format')
    return false
  }
  setEmailError('')
  return true
}

// In JSX
<Input
  value={email}
  onChange={(e) => setEmail(e.target.value)}
  onBlur={() => validateEmail(email)}
  className={emailError ? 'border-red-500' : ''}
/>
{emailError && (
  <p className="text-xs text-red-600">{emailError}</p>
)}
```

## Testing Conventions

### Test File Naming

Use `.test.tsx` or `.spec.tsx` suffix:

```
component-name.test.tsx
auth-context.test.tsx
use-auth.test.tsx
```

### Test Organization

```tsx
describe('ComponentName', () => {
  it('should render correctly', () => {
    // Test implementation
  })

  it('should handle user interaction', () => {
    // Test implementation
  })
})
```

## Code Quality Tools

### Linting & Formatting

- **ESLint**: Configured for React and TypeScript
- **Prettier**: Automatic code formatting
- **Commands**:
  - `pnpm run lint` - Check for issues
  - `pnpm run lint --fix` - Auto-fix issues
  - `pnpm run format` - Format code

## Documentation

### Component Documentation

Add JSDoc comments for complex components:

```tsx
/**
 * User profile component that displays user information
 * and allows editing.
 *
 * @param user - The user object to display
 * @param onEdit - Callback when edit button is clicked
 */
export const UserProfile: React.FC<UserProfileProps> = ({ user, onEdit }) => {
  // Implementation
}
```

## Git Conventions

### Commit Messages

Follow conventional commits for frontend changes:

```
feat(frontend): add login page UI
fix(frontend): correct email validation
style(frontend): update button styles
refactor(frontend): simplify auth context
```

## Summary: Quick Reference

| Type | Location | File Name | Export Name | Example |
|------|----------|-----------|-------------|---------|
| **UI Component** | `src/components/` | kebab-case | PascalCase | `user-avatar.tsx` → `UserAvatar` |
| **Shadcn Component** | `src/components/ui/` | kebab-case | PascalCase | `button.tsx` → `Button` |
| **Page Component** | `src/pages/` | PascalCase | PascalCase | `LoginPage.tsx` → `LoginPage` |
| **Context** | `src/contexts/` | PascalCase | PascalCase | `AuthContext.tsx` → `AuthContext` |
| **Hook** | `src/hooks/` | PascalCase | camelCase | `UseAuth.tsx` → `useAuth` |
| **Utility** | `src/lib/` | kebab-case | camelCase | `api-client.ts` → `apiClient` |
| **Type** | `src/types/` | kebab-case | PascalCase | `user-types.ts` → `User` |

---

**Last Updated**: 2025-01-25
**Reference for**: All frontend development in the Deeb project
