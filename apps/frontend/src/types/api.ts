export type ErrorCode =
  | "INVALID_PAYLOAD"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "RATE_LIMITED"
  | "INTERNAL_ERROR";

export interface ApiErrorInfo {
  readonly code: ErrorCode;
  readonly message: string;
  readonly details?: Record<string, unknown>;
}

export interface ApiPagination {
  readonly page: number;
  readonly perPage: number;
  readonly total: number;
}

export interface ApiMeta {
  readonly traceId?: string;
  readonly pagination?: ApiPagination;
  readonly warnings?: readonly string[];
}

export interface ApiEnvelope<T> {
  readonly success: boolean;
  readonly data: T | null;
  readonly error: ApiErrorInfo | null;
  readonly meta: ApiMeta;
}

export type ApiResponse<T> = ApiEnvelope<T>;
export type ApiListResponse<T> = ApiEnvelope<readonly T[]>;

export function isApiError<T>(
  response: ApiEnvelope<T>,
): response is ApiEnvelope<T> & { readonly error: ApiErrorInfo } {
  return !response.success && response.error !== null;
}

export function isApiSuccess<T>(
  response: ApiEnvelope<T>,
): response is ApiEnvelope<T> & { readonly data: T } {
  return response.success && response.data !== null;
}

export class ApiError extends Error {
  constructor(
    public readonly code: ErrorCode,
    message: string,
    public readonly status: number,
    public readonly traceId?: string,
    public readonly details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ApiError";
  }

  static fromResponse<T>(response: ApiEnvelope<T>, status: number): ApiError {
    const error = response.error;
    if (!error) {
      return new ApiError(
        "INTERNAL_ERROR",
        "Unknown error",
        status,
        response.meta.traceId,
      );
    }
    return new ApiError(
      error.code,
      error.message,
      status,
      response.meta.traceId,
      error.details as Record<string, unknown>,
    );
  }
}
