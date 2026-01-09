/**
 * Query Key Factory for consistent React Query cache keys.
 *
 * This factory provides a type-safe way to generate query keys
 * that follow a consistent hierarchy for proper cache invalidation.
 */

// ============================================================================
// Query Key Types
// ============================================================================

export type QueryKeyBase = readonly string[];

export type QueryKeyWithId<T extends QueryKeyBase> = readonly [...T, string];

export type QueryKeyWithParams<T extends QueryKeyBase, P> = readonly [...T, P];

// ============================================================================
// Query Key Factory
// ============================================================================

/**
 * Creates a query key factory for a resource.
 *
 * @example
 * ```ts
 * const tenantKeys = createQueryKeys("tenants");
 *
 * // All tenants: ["tenants", "list"]
 * tenantKeys.list();
 *
 * // Single tenant: ["tenants", "detail", "tenant-123"]
 * tenantKeys.detail("tenant-123");
 *
 * // Tenant with nested resource: ["tenants", "detail", "tenant-123", "api-keys"]
 * tenantKeys.detail("tenant-123").nested("api-keys");
 * ```
 */
export function createQueryKeys<TResource extends string>(resource: TResource) {
  const all = [resource] as const;

  return {
    /** Base key for all queries of this resource */
    all,

    /**
     * Key for list queries.
     * Optionally accepts filter parameters.
     */
    list: <TParams extends Record<string, unknown> | undefined = undefined>(
      params?: TParams,
    ) => {
      const base = [...all, "list"] as const;
      return params ? ([...base, params] as const) : base;
    },

    /**
     * Key for list queries with specific filters.
     */
    lists: () => [...all, "list"] as const,

    /**
     * Key for a single resource detail.
     */
    detail: (id: string) => {
      const detailKey = [...all, "detail", id] as const;

      return {
        /** The detail key itself */
        key: detailKey,

        /**
         * Key for a nested resource under this detail.
         */
        nested: <TNested extends string>(nestedResource: TNested) =>
          [...detailKey, nestedResource] as const,

        /**
         * Key for a nested resource with an ID.
         */
        nestedDetail: <TNested extends string>(
          nestedResource: TNested,
          nestedId: string,
        ) => [...detailKey, nestedResource, nestedId] as const,
      };
    },

    /**
     * Key for all detail queries (for invalidation).
     */
    details: () => [...all, "detail"] as const,

    /**
     * Key for a specific action on a resource.
     */
    action: <TAction extends string>(action: TAction, id?: string) => {
      const actionKey = [...all, action] as const;
      return id ? ([...actionKey, id] as const) : actionKey;
    },
  };
}

// ============================================================================
// Nested Query Key Factory
// ============================================================================

/**
 * Creates a query key factory for nested resources.
 *
 * @example
 * ```ts
 * const apiKeyKeys = createNestedQueryKeys("tenants", "api-keys");
 *
 * // All API keys for a tenant: ["tenants", "tenant-123", "api-keys", "list"]
 * apiKeyKeys.list("tenant-123");
 *
 * // Single API key: ["tenants", "tenant-123", "api-keys", "detail", "key-456"]
 * apiKeyKeys.detail("tenant-123", "key-456");
 * ```
 */
export function createNestedQueryKeys<
  TParent extends string,
  TChild extends string,
>(parent: TParent, child: TChild) {
  const parentKey = [parent] as const;

  return {
    /** Base key for parent resource */
    parent: parentKey,

    /**
     * Key for all child resources under a parent.
     */
    all: (parentId: string) => [...parentKey, parentId, child] as const,

    /**
     * Key for list of child resources.
     */
    list: <TParams extends Record<string, unknown> | undefined = undefined>(
      parentId: string,
      params?: TParams,
    ) => {
      const base = [...parentKey, parentId, child, "list"] as const;
      return params ? ([...base, params] as const) : base;
    },

    /**
     * Key for a single child resource.
     */
    detail: (parentId: string, childId: string) =>
      [...parentKey, parentId, child, "detail", childId] as const,

    /**
     * Key for all detail queries under a parent (for invalidation).
     */
    details: (parentId: string) =>
      [...parentKey, parentId, child, "detail"] as const,

    /**
     * Key for an action on a child resource.
     */
    action: <TAction extends string>(
      parentId: string,
      action: TAction,
      childId?: string,
    ) => {
      const actionKey = [...parentKey, parentId, child, action] as const;
      return childId ? ([...actionKey, childId] as const) : actionKey;
    },
  };
}

// ============================================================================
// Pre-defined Query Keys
// ============================================================================

/**
 * Pre-defined query keys for common resources.
 * Use these for consistency across the application.
 */
export const queryKeys = {
  // Admin resources
  tenants: createQueryKeys("tenants"),
  users: createQueryKeys("users"),
  models: createQueryKeys("models"),
  providers: createQueryKeys("providers"),
  budgets: createQueryKeys("budgets"),
  rateLimits: createQueryKeys("rate-limits"),

  // Nested resources
  tenantApiKeys: createNestedQueryKeys("tenants", "api-keys"),
  tenantMemberships: createNestedQueryKeys("tenants", "memberships"),
  tenantModels: createNestedQueryKeys("tenants", "models"),
  tenantUsage: createNestedQueryKeys("tenants", "usage"),

  // User resources (prefixed to avoid conflicts)
  userTenants: createQueryKeys("user-tenants"),
  userApiKeys: createQueryKeys("user-api-keys"),
  userProfile: createQueryKeys("user-profile"),
  userUsage: createQueryKeys("user-usage"),

  // Global resources
  health: createQueryKeys("health"),
  telemetry: createQueryKeys("telemetry"),
  settings: createQueryKeys("settings"),
} as const;

// ============================================================================
// Invalidation Helpers
// ============================================================================

/**
 * Get all keys that should be invalidated when a resource changes.
 *
 * @example
 * ```ts
 * // When a tenant is updated, invalidate tenant list and detail
 * const keys = getInvalidationKeys(queryKeys.tenants, "tenant-123");
 * // Returns: [["tenants"], ["tenants", "detail", "tenant-123"]]
 * ```
 */
export function getInvalidationKeys(
  keyFactory: ReturnType<typeof createQueryKeys>,
  id?: string,
): readonly (readonly string[])[] {
  const keys: (readonly string[])[] = [keyFactory.all];

  if (id) {
    keys.push(keyFactory.detail(id).key);
  }

  return keys;
}

/**
 * Get all keys that should be invalidated when a nested resource changes.
 */
export function getNestedInvalidationKeys(
  keyFactory: ReturnType<typeof createNestedQueryKeys>,
  parentId: string,
  childId?: string,
): readonly (readonly string[])[] {
  const keys: (readonly string[])[] = [keyFactory.all(parentId)];

  if (childId) {
    keys.push(keyFactory.detail(parentId, childId));
  }

  return keys;
}
