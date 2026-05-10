import { onScopeDispose, ref } from 'vue'

export interface UseAutoRefreshOptions {
  intervalMs: number
  immediate?: boolean
}

export function useAutoRefresh(
  task: () => void | Promise<void>,
  options: UseAutoRefreshOptions,
) {
  const running = ref(false)
  let timer: number | null = null

  const tick = async () => {
    try {
      await task()
    } catch {
    }
  }

  const start = () => {
    if (timer !== null) return
    running.value = true
    if (options.immediate !== false) void tick()
    timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') void tick()
    }, options.intervalMs)
  }

  const stop = () => {
    running.value = false
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  const handleVisibility = () => {
    if (document.visibilityState === 'visible' && running.value) void tick()
  }
  document.addEventListener('visibilitychange', handleVisibility)

  onScopeDispose(() => {
    stop()
    document.removeEventListener('visibilitychange', handleVisibility)
  })

  return { start, stop, running }
}
