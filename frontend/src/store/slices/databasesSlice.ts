import type { StateCreator } from 'zustand';
import type { Database } from '@/lib/api/types';
import { databasesService } from '@/lib/api/services/databases';

export interface DatabasesSlice {
  // State
  databases: Database[];
  databasesLoading: boolean;
  databasesError: string | null;
  databasesLastFetchedAt: number | null;

  // Actions
  fetchDatabases: (filters?: { project_id?: string; environment_id?: string }) => Promise<void>;
  setDatabases: (databases: Database[]) => void;
  addDatabase: (database: Database) => void;
  updateDatabase: (databaseId: string, updates: Partial<Database>) => void;
  removeDatabase: (databaseId: string) => void;
  clearDatabases: () => void;
  getDatabaseById: (databaseId: string) => Database | undefined;
  getDatabasesByProjectId: (projectId: string) => Database[];
}

const CACHE_TTL = 5 * 60 * 1000; // 5 minutes cache TTL

export const createDatabasesSlice: StateCreator<DatabasesSlice, [], [], DatabasesSlice> = (
  set,
  get,
) => ({
  // Initial state
  databases: [],
  databasesLoading: false,
  databasesError: null,
  databasesLastFetchedAt: null,

  // Fetch databases from API with caching
  fetchDatabases: async (filters) => {
    const state = get();

    // Check if data is still fresh (only if no filters provided)
    if (
      !filters &&
      state.databasesLastFetchedAt &&
      Date.now() - state.databasesLastFetchedAt < CACHE_TTL &&
      state.databases.length > 0
    ) {
      // Data is fresh, no need to fetch
      return;
    }

    // Prevent duplicate requests
    if (state.databasesLoading) {
      return;
    }

    set({ databasesLoading: true, databasesError: null });

    try {
      const response = await databasesService.listDatabases({
        limit: 1000,
        ...filters,
      }); // Fetch all databases
      set({
        databases: response.items,
        databasesLoading: false,
        databasesError: null,
        databasesLastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        databasesLoading: false,
        databasesError: error instanceof Error ? error.message : 'Failed to fetch databases',
      });
    }
  },

  // Set databases directly (useful for initial load or refresh)
  setDatabases: (databases) =>
    set({
      databases,
      databasesLastFetchedAt: Date.now(),
    }),

  // Add a new database to the cache
  addDatabase: (database) =>
    set((state) => ({
      databases: [...state.databases, database],
      databasesLastFetchedAt: Date.now(),
    })),

  // Update an existing database in the cache
  updateDatabase: (databaseId, updates) =>
    set((state) => ({
      databases: state.databases.map((database) =>
        database.id === databaseId ? { ...database, ...updates } : database,
      ),
      databasesLastFetchedAt: Date.now(),
    })),

  // Remove a database from the cache
  removeDatabase: (databaseId) =>
    set((state) => ({
      databases: state.databases.filter((database) => database.id !== databaseId),
      databasesLastFetchedAt: Date.now(),
    })),

  // Clear all databases
  clearDatabases: () =>
    set({
      databases: [],
      databasesLoading: false,
      databasesError: null,
      databasesLastFetchedAt: null,
    }),

  // Helper to get database by ID
  getDatabaseById: (databaseId) => {
    const state = get();
    return state.databases.find((database) => database.id === databaseId);
  },

  // Helper to get databases by project ID
  getDatabasesByProjectId: (projectId) => {
    const state = get();
    return state.databases.filter(
      (database) => database.project_id && database.project_id === projectId,
    );
  },
});
