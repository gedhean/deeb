# Zustand Store

This directory contains the Zustand state management setup for the Deeb frontend application.

## Architecture

The store follows Zustand best practices:

- **Slice Pattern**: State is divided into focused, modular slices (organization, project)
- **TypeScript**: Fully typed with proper type inference
- **Flux-Inspired**: Actions are colocated with state
- **Persistence**: State is persisted to localStorage
- **DevTools**: Redux DevTools integration for debugging

## Structure

```
src/store/
├── index.ts                      # Main store combining all slices
└── slices/
    ├── organizationSlice.ts      # Organization state and actions
    └── projectSlice.ts           # Current project state and actions
```

## Usage

### Basic Usage

```typescript
import { useAppStore } from '@/store';

function MyComponent() {
  // Get entire store (not recommended - causes re-renders on any state change)
  const store = useAppStore();

  // Better: Use specific selectors for optimal performance
  const currentProject = useAppStore((state) => state.currentProject);
  const setCurrentProject = useAppStore((state) => state.setCurrentProject);
}
```

### Using Exported Selectors (Recommended)

```typescript
import {
  useCurrentProject,
  useSetCurrentProject,
  useOrganization,
  useSetOrganization,
  useClearAllState,
} from '@/store';

function ProjectSelector() {
  const currentProject = useCurrentProject();
  const setCurrentProject = useSetCurrentProject();

  const handleSelect = (project: Project) => {
    setCurrentProject(project);
  };

  return (
    <div>
      {currentProject && <p>Current: {currentProject.name}</p>}
      {/* ... */}
    </div>
  );
}

function LogoutButton() {
  const clearAllState = useClearAllState();

  const handleLogout = () => {
    clearAllState();
    // ... other logout logic
  };

  return <button onClick={handleLogout}>Logout</button>;
}
```

### Organization State

```typescript
import { useOrganization, useSetOrganization, useClearOrganization } from '@/store';

function OrganizationInfo() {
  const organization = useOrganization();
  const setOrganization = useSetOrganization();
  const clearOrganization = useClearOrganization();

  // Set organization
  useEffect(() => {
    fetchOrganization().then(setOrganization);
  }, [setOrganization]);

  return <div>{organization?.name}</div>;
}
```

### Project State

```typescript
import { useCurrentProject, useSetCurrentProject, useClearCurrentProject } from '@/store';

function ProjectInfo() {
  const currentProject = useCurrentProject();
  const setCurrentProject = useSetCurrentProject();
  const clearCurrentProject = useClearCurrentProject();

  return (
    <div>
      <h2>{currentProject?.name}</h2>
      <p>{currentProject?.description}</p>
    </div>
  );
}
```

## Performance Tips

1. **Use specific selectors**: Only select the state you need to minimize re-renders
2. **Use exported selectors**: The exported selector functions (`useCurrentProject`, etc.) are pre-optimized
3. **Avoid selecting entire store**: Don't use `useAppStore()` directly unless you need all state
4. **Split actions and state**: Select actions separately from state when you only need to call actions

## Adding New Slices

To add a new slice:

1. Create a new file in `src/store/slices/` (e.g., `mySlice.ts`)
2. Define your slice interface and create function following the pattern:

```typescript
import { StateCreator } from 'zustand';

export interface MySlice {
  myState: string;
  setMyState: (value: string) => void;
}

export const createMySlice: StateCreator<MySlice, [], [], MySlice> = (set) => ({
  myState: '',
  setMyState: (value) => set({ myState: value }),
});
```

3. Add the slice to the main store in `src/store/index.ts`:

```typescript
import { createMySlice, MySlice } from './slices/mySlice';

export type AppStore = OrganizationSlice & ProjectSlice & MySlice;

export const useAppStore = create<AppStore>()(
  devtools(
    persist(
      (...a) => ({
        ...createOrganizationSlice(...a),
        ...createProjectSlice(...a),
        ...createMySlice(...a),
      }),
      {
        name: 'deeb-app-store',
        partialize: (state) => ({
          currentProject: state.currentProject,
          organization: state.organization,
          myState: state.myState, // Add to persistence
        }),
      },
    ),
  ),
);

// Export selectors
export const useMyState = () => useAppStore((state) => state.myState);
export const useSetMyState = () => useAppStore((state) => state.setMyState);
```

## References

- [Zustand Documentation](https://github.com/pmndrs/zustand)
- [TypeScript Guide](https://github.com/pmndrs/zustand/blob/main/docs/guides/typescript.md)
- [Slices Pattern](https://github.com/pmndrs/zustand/blob/main/docs/guides/slices-pattern.md)
- [Flux-Inspired Practice](https://github.com/pmndrs/zustand/blob/main/docs/guides/flux-inspired-practice.md)
