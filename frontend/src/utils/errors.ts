export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
    this.name = 'ApiError'
  }
}

export class NotFoundError extends ApiError {
  constructor(message = 'resource not found') {
    super(40001, message)
    this.name = 'NotFoundError'
  }
}

export class ValidationError extends ApiError {
  constructor(message = 'validation failed') {
    super(40002, message)
    this.name = 'ValidationError'
  }
}

export class ServiceDownError extends ApiError {
  constructor(code: 50001 | 50002, message = 'dependency down') {
    super(code, message)
    this.name = 'ServiceDownError'
  }
}

export class UnknownError extends ApiError {
  constructor(message = 'unknown error', code = 50003) {
    super(code, message)
    this.name = 'UnknownError'
  }
}
