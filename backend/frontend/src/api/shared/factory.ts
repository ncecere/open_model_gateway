/**
 * API Factory for creating typed resource endpoints.
 *
 * This factory provides a consistent way to create CRUD operations
 * for API resources with proper typing and error handling.
 */

import type { AxiosInstance, AxiosRequestConfig } from "axios";
import type { PaginationParams, PaginatedResponse } from "./types";

// ============================================================================
// Factory Types
// ============================================================================

export type HttpMethod = "get" | "post" | "put" | "patch" | "delete";

export interface ResourceConfig<T> {
  /** Base path for the resource (e.g., "/tenants") */
  basePath: string;
  /** Axios instance to use for requests */
  client: AxiosInstance;
  /** Optional transform for response data */
  transformResponse?: (data: unknown) => T;
  /** Optional transform for list response data */
  transformListResponse?: (data: unknown) => T[];
}

export interface ListOptions extends PaginationParams {
  /** Additional query parameters */
  params?: Record<string, unknown>;
}

export interface RequestOptions {
  /** Allow specific status codes without throwing */
  allowedStatuses?: number[];
  /** Additional axios config */
  config?: AxiosRequestConfig;
}

// ============================================================================
// Resource Factory
// ============================================================================

/**
 * Creates a set of typed CRUD operations for a resource.
 *
 * @example
 * ```ts
 * const tenantsApi = createResource<Tenant, CreateTenantPayload>({
 *   basePath: "/tenants",
 *   client: api,
 * });
 *
 * // List all tenants
 * const tenants = await tenantsApi.list();
 *
 * // Get a single tenant
 * const tenant = await tenantsApi.get("tenant-id");
 *
 * // Create a tenant
 * const newTenant = await tenantsApi.create({ name: "My Tenant" });
 * ```
 */
export function createResource<
  T,
  TCreate = Partial<T>,
  TUpdate = Partial<T>,
  TList = T,
>(config: ResourceConfig<T>) {
  const { basePath, client, transformResponse, transformListResponse } = config;

  /**
   * List all resources with optional pagination.
   */
  async function list(options?: ListOptions): Promise<TList[]> {
    const { data } = await client.get(basePath, {
      params: {
        limit: options?.limit,
        offset: options?.offset,
        ...options?.params,
      },
    });

    // Handle common response shapes
    if (transformListResponse) {
      return transformListResponse(data) as unknown as TList[];
    }

    // Try common list response patterns
    if (Array.isArray(data)) {
      return data as unknown as TList[];
    }

    // Look for common list property names
    const listKeys = ["data", "items", "records", "results"];
    for (const key of listKeys) {
      if (Array.isArray(data[key])) {
        return data[key] as unknown as TList[];
      }
    }

    // Try to find any array property
    for (const key of Object.keys(data)) {
      if (Array.isArray(data[key])) {
        return data[key] as unknown as TList[];
      }
    }

    return [] as unknown as TList[];
  }

  /**
   * List resources with full pagination metadata.
   */
  async function listPaginated(
    options?: ListOptions,
  ): Promise<PaginatedResponse<TList>> {
    const { data } = await client.get(basePath, {
      params: {
        limit: options?.limit,
        offset: options?.offset,
        ...options?.params,
      },
    });

    const items = transformListResponse
      ? (transformListResponse(data) as unknown as TList[])
      : findArrayInResponse<TList>(data);

    return {
      data: items,
      limit: data.limit ?? options?.limit ?? 100,
      offset: data.offset ?? options?.offset ?? 0,
      total: data.total,
    };
  }

  /**
   * Get a single resource by ID.
   */
  async function get(id: string, options?: RequestOptions): Promise<T | null> {
    const response = await client.get(`${basePath}/${id}`, {
      ...options?.config,
      validateStatus: (status) => {
        if (options?.allowedStatuses?.includes(status)) return true;
        return status >= 200 && status < 300;
      },
    });

    if (options?.allowedStatuses?.includes(response.status)) {
      if (response.status === 404) return null;
    }

    return transformResponse
      ? transformResponse(response.data)
      : (response.data as T);
  }

  /**
   * Create a new resource.
   */
  async function create(payload: TCreate): Promise<T> {
    const { data } = await client.post(basePath, payload);
    return transformResponse ? transformResponse(data) : (data as T);
  }

  /**
   * Update a resource by ID (full replacement).
   */
  async function update(id: string, payload: TUpdate): Promise<T> {
    const { data } = await client.put(`${basePath}/${id}`, payload);
    return transformResponse ? transformResponse(data) : (data as T);
  }

  /**
   * Partially update a resource by ID.
   */
  async function patch(id: string, payload: Partial<TUpdate>): Promise<T> {
    const { data } = await client.patch(`${basePath}/${id}`, payload);
    return transformResponse ? transformResponse(data) : (data as T);
  }

  /**
   * Delete a resource by ID.
   */
  async function remove(id: string): Promise<void> {
    await client.delete(`${basePath}/${id}`);
  }

  return {
    list,
    listPaginated,
    get,
    create,
    update,
    patch,
    remove,
    /** The base path for this resource */
    basePath,
    /** The axios client for custom requests */
    client,
  };
}

// ============================================================================
// Nested Resource Factory
// ============================================================================

export interface NestedResourceConfig<T> {
  /** Parent resource path (e.g., "/tenants") */
  parentPath: string;
  /** Child resource path segment (e.g., "api-keys") */
  childPath: string;
  /** Axios instance to use for requests */
  client: AxiosInstance;
  /** Optional transform for response data */
  transformResponse?: (data: unknown) => T;
  /** Optional transform for list response data */
  transformListResponse?: (data: unknown) => T[];
}

/**
 * Creates a set of typed CRUD operations for a nested resource.
 *
 * @example
 * ```ts
 * const tenantApiKeysApi = createNestedResource<ApiKey, CreateKeyPayload>({
 *   parentPath: "/tenants",
 *   childPath: "api-keys",
 *   client: api,
 * });
 *
 * // List all API keys for a tenant
 * const keys = await tenantApiKeysApi.list("tenant-id");
 *
 * // Create an API key for a tenant
 * const newKey = await tenantApiKeysApi.create("tenant-id", { name: "My Key" });
 * ```
 */
export function createNestedResource<
  T,
  TCreate = Partial<T>,
  TUpdate = Partial<T>,
  TList = T,
>(config: NestedResourceConfig<T>) {
  const { parentPath, childPath, client, transformResponse, transformListResponse } =
    config;

  function buildPath(parentId: string, childId?: string): string {
    const base = `${parentPath}/${parentId}/${childPath}`;
    return childId ? `${base}/${childId}` : base;
  }

  /**
   * List all child resources for a parent.
   */
  async function list(parentId: string, options?: ListOptions): Promise<TList[]> {
    const { data } = await client.get(buildPath(parentId), {
      params: {
        limit: options?.limit,
        offset: options?.offset,
        ...options?.params,
      },
    });

    if (transformListResponse) {
      return transformListResponse(data) as unknown as TList[];
    }

    return findArrayInResponse<TList>(data);
  }

  /**
   * Get a single child resource.
   */
  async function get(
    parentId: string,
    childId: string,
    options?: RequestOptions,
  ): Promise<T | null> {
    const response = await client.get(buildPath(parentId, childId), {
      ...options?.config,
      validateStatus: (status) => {
        if (options?.allowedStatuses?.includes(status)) return true;
        return status >= 200 && status < 300;
      },
    });

    if (options?.allowedStatuses?.includes(response.status)) {
      if (response.status === 404) return null;
    }

    return transformResponse
      ? transformResponse(response.data)
      : (response.data as T);
  }

  /**
   * Create a new child resource.
   */
  async function create(parentId: string, payload: TCreate): Promise<T> {
    const { data } = await client.post(buildPath(parentId), payload);
    return transformResponse ? transformResponse(data) : (data as T);
  }

  /**
   * Update a child resource.
   */
  async function update(
    parentId: string,
    childId: string,
    payload: TUpdate,
  ): Promise<T> {
    const { data } = await client.put(buildPath(parentId, childId), payload);
    return transformResponse ? transformResponse(data) : (data as T);
  }

  /**
   * Partially update a child resource.
   */
  async function patch(
    parentId: string,
    childId: string,
    payload: Partial<TUpdate>,
  ): Promise<T> {
    const { data } = await client.patch(buildPath(parentId, childId), payload);
    return transformResponse ? transformResponse(data) : (data as T);
  }

  /**
   * Delete a child resource.
   */
  async function remove(parentId: string, childId: string): Promise<void> {
    await client.delete(buildPath(parentId, childId));
  }

  return {
    list,
    get,
    create,
    update,
    patch,
    remove,
    buildPath,
    client,
  };
}

// ============================================================================
// Custom Endpoint Factory
// ============================================================================

export interface EndpointConfig {
  /** Axios instance to use for requests */
  client: AxiosInstance;
}

/**
 * Creates a typed endpoint function for custom API calls.
 *
 * @example
 * ```ts
 * const updateStatus = createEndpoint<Tenant, { status: string }>({
 *   client: api,
 * }).patch("/tenants/:id/status");
 *
 * const updated = await updateStatus({ id: "123" }, { status: "active" });
 * ```
 */
export function createEndpoint(config: EndpointConfig) {
  const { client } = config;

  function buildUrl(
    template: string,
    params: Record<string, string>,
  ): string {
    let url = template;
    for (const [key, value] of Object.entries(params)) {
      url = url.replace(`:${key}`, encodeURIComponent(value));
    }
    return url;
  }

  return {
    get<TResponse, TParams extends Record<string, string> = Record<string, string>>(
      path: string,
    ) {
      return async (
        pathParams: TParams,
        queryParams?: Record<string, unknown>,
      ): Promise<TResponse> => {
        const { data } = await client.get(buildUrl(path, pathParams), {
          params: queryParams,
        });
        return data as TResponse;
      };
    },

    post<
      TResponse,
      TBody = unknown,
      TParams extends Record<string, string> = Record<string, string>,
    >(path: string) {
      return async (pathParams: TParams, body?: TBody): Promise<TResponse> => {
        const { data } = await client.post(buildUrl(path, pathParams), body);
        return data as TResponse;
      };
    },

    put<
      TResponse,
      TBody = unknown,
      TParams extends Record<string, string> = Record<string, string>,
    >(path: string) {
      return async (pathParams: TParams, body?: TBody): Promise<TResponse> => {
        const { data } = await client.put(buildUrl(path, pathParams), body);
        return data as TResponse;
      };
    },

    patch<
      TResponse,
      TBody = unknown,
      TParams extends Record<string, string> = Record<string, string>,
    >(path: string) {
      return async (pathParams: TParams, body?: TBody): Promise<TResponse> => {
        const { data } = await client.patch(buildUrl(path, pathParams), body);
        return data as TResponse;
      };
    },

    delete<TParams extends Record<string, string> = Record<string, string>>(
      path: string,
    ) {
      return async (pathParams: TParams): Promise<void> => {
        await client.delete(buildUrl(path, pathParams));
      };
    },
  };
}

// ============================================================================
// Helpers
// ============================================================================

/**
 * Find an array in a response object.
 */
function findArrayInResponse<T>(data: unknown): T[] {
  if (Array.isArray(data)) {
    return data as T[];
  }

  if (typeof data === "object" && data !== null) {
    const obj = data as Record<string, unknown>;

    // Try common list property names first
    const listKeys = ["data", "items", "records", "results"];
    for (const key of listKeys) {
      if (Array.isArray(obj[key])) {
        return obj[key] as T[];
      }
    }

    // Try to find any array property
    for (const key of Object.keys(obj)) {
      if (Array.isArray(obj[key])) {
        return obj[key] as T[];
      }
    }
  }

  return [] as T[];
}
