/**
 * Page size used when fetching a full reference list to populate filter-option
 * dropdowns or to resolve related-entity names in a table (projects,
 * environments, database instances, users, …).
 *
 * These lists are rendered as client-side selectable options, so they must
 * contain every entity — not just the first page. 1000 is the backend's maximum
 * allowed `limit`. If a single resource can realistically exceed this, the
 * corresponding filter should move to a server-side searchable select instead.
 */
export const REFERENCE_LIST_LIMIT = 1000;
