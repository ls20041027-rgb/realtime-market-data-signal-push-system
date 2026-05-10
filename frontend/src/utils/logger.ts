const isDev = import.meta.env.DEV

type Level = 'debug' | 'info' | 'warn' | 'error'

const emit = (level: Level, scope: string, msg: string, extra?: unknown) => {
  if (!isDev && level === 'debug') return
  const prefix = `[${scope}]`
  const fn =
    level === 'error' ? console.error : level === 'warn' ? console.warn : console.log
  extra === undefined ? fn(prefix, msg) : fn(prefix, msg, extra)
}

export const createLogger = (scope: string) => ({
  debug: (msg: string, extra?: unknown) => emit('debug', scope, msg, extra),
  info: (msg: string, extra?: unknown) => emit('info', scope, msg, extra),
  warn: (msg: string, extra?: unknown) => emit('warn', scope, msg, extra),
  error: (msg: string, extra?: unknown) => emit('error', scope, msg, extra),
})
