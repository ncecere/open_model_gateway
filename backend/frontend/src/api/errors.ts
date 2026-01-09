import { AxiosError } from "axios";

/**
 * API error codes commonly returned by the backend.
 */
export type ApiErrorCode =
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "rate_limited"
  | "validation_error"
  | "internal_error"
  | "service_unavailable";

/**
 * Structured API error response from the backend.
 */
export interface ApiErrorResponse {
  error?: string;
  message?: string;
  code?: ApiErrorCode;
  details?: Record<string, unknown>;
}

/**
 * ApiError wraps axios errors with typed response data.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code?: ApiErrorCode;
  readonly details?: Record<string, unknown>;
  readonly originalError: AxiosError<ApiErrorResponse>;

  constructor(error: AxiosError<ApiErrorResponse>) {
    const response = error.response;
    const message =
      response?.data?.error ||
      response?.data?.message ||
      error.message ||
      "An unexpected error occurred";

    super(message);
    this.name = "ApiError";
    this.status = response?.status ?? 0;
    this.code = response?.data?.code;
    this.details = response?.data?.details;
    this.originalError = error;
  }

  /**
   * Check if this is a specific HTTP status error.
   */
  isStatus(status: number): boolean {
    return this.status === status;
  }

  /**
   * Check if this is an authentication error (401).
   */
  isUnauthorized(): boolean {
    return this.status === 401;
  }

  /**
   * Check if this is a permission error (403).
   */
  isForbidden(): boolean {
    return this.status === 403;
  }

  /**
   * Check if the resource was not found (404).
   */
  isNotFound(): boolean {
    return this.status === 404;
  }

  /**
   * Check if this is a rate limit error (429).
   */
  isRateLimited(): boolean {
    return this.status === 429;
  }

  /**
   * Check if this is a validation error (400).
   */
  isValidationError(): boolean {
    return this.status === 400;
  }

  /**
   * Check if this is a server error (5xx).
   */
  isServerError(): boolean {
    return this.status >= 500 && this.status < 600;
  }
}

/**
 * Type guard to check if an error is an AxiosError.
 */
export function isAxiosError(error: unknown): error is AxiosError<ApiErrorResponse> {
  return error instanceof Error && "isAxiosError" in error;
}

/**
 * Convert an unknown error to an ApiError.
 */
export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  if (isAxiosError(error)) {
    return new ApiError(error);
  }
  // Wrap unknown errors
  const axiosLikeError = {
    message: error instanceof Error ? error.message : String(error),
    response: undefined,
    isAxiosError: true,
    config: {},
    toJSON: () => ({}),
  } as unknown as AxiosError<ApiErrorResponse>;
  return new ApiError(axiosLikeError);
}

/**
 * Get a user-friendly error message from an error.
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (isAxiosError(error)) {
    return new ApiError(error).message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
