// usePaymentOrderPolling
//
// 按 out_trade_no 轮询 verifyOrder 来获取最新订单状态。
// 终止条件：订单进入终止态（PAID/RECHARGING/COMPLETED/CANCELLED/EXPIRED/FAILED/REFUNDED/REFUND_FAILED）
// 或者超过最大轮询次数（默认 300 次 × 2s = 10 分钟 —— 与后端默认 order_timeout 30 分钟留足余量）。

import { onBeforeUnmount, ref } from 'vue'
import { paymentApi, type OrderStatus, type PaymentOrder } from '../api/payment'
import { getErrorMessage } from '../core/api'

export type PollingOutcome = 'paid' | 'terminal-nonpaid' | 'timeout' | 'error'

export interface UsePaymentOrderPollingOptions {
  // 单次轮询间隔（ms），默认 2000
  intervalMs?: number
  // 最大轮询次数，默认 300
  maxAttempts?: number
  // 是否在终止时自动停止
  autoStopOnTerminal?: boolean
  // 业务级回调
  onPaid?: (order: PaymentOrder) => void
  onTerminal?: (order: PaymentOrder) => void
  onError?: (message: string) => void
}

const TERMINAL_STATUSES: OrderStatus[] = [
  'PAID',
  'RECHARGING',
  'COMPLETED',
  'EXPIRED',
  'CANCELLED',
  'FAILED',
  'REFUNDED',
  'REFUND_FAILED',
  'PARTIALLY_REFUNDED',
  'REFUND_REQUESTED',
  'REFUNDING',
  'REFUND_PENDING',
]

export function isTerminal(status: OrderStatus): boolean {
  return TERMINAL_STATUSES.includes(status)
}

export function isPaidLike(status: OrderStatus): boolean {
  return status === 'PAID' || status === 'RECHARGING' || status === 'COMPLETED' || status === 'PARTIALLY_REFUNDED'
}

export function usePaymentOrderPolling(outTradeNo: () => string, options: UsePaymentOrderPollingOptions = {}) {
  const order = ref<PaymentOrder | null>(null)
  const status = ref<OrderStatus | ''>('')
  const attempts = ref(0)
  const polling = ref(false)
  const outcome = ref<PollingOutcome | ''>('')
  const lastError = ref('')

  let timer: ReturnType<typeof setTimeout> | null = null
  let stopped = false

  function clearTimer() {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  function stop(reason: PollingOutcome) {
    stopped = true
    polling.value = false
    outcome.value = reason
    clearTimer()
  }

  async function tick() {
    if (stopped) return
    const key = outTradeNo()
    if (!key) {
      stop('error')
      lastError.value = '缺少 out_trade_no'
      options.onError?.(lastError.value)
      return
    }
    attempts.value += 1
    try {
      const latest = await paymentApi.verifyOrder({ out_trade_no: key })
      if (latest) {
        order.value = latest
        status.value = latest.status
      }
      if (latest && isTerminal(latest.status)) {
        if (isPaidLike(latest.status)) {
          options.onPaid?.(latest)
          stop('paid')
        } else {
          options.onTerminal?.(latest)
          stop('terminal-nonpaid')
        }
        return
      }
    } catch (e) {
      lastError.value = getErrorMessage(e)
      // 网络错误不立刻停止，继续轮询
      options.onError?.(lastError.value)
    }
    const limit = options.maxAttempts ?? 300
    if (attempts.value >= limit) {
      stop('timeout')
      return
    }
    timer = setTimeout(tick, options.intervalMs ?? 2000)
  }

  function start() {
    if (polling.value) return
    stopped = false
    polling.value = true
    outcome.value = ''
    attempts.value = 0
    lastError.value = ''
    tick()
  }

  function pause() {
    stopped = true
    polling.value = false
    clearTimer()
  }

  function resume() {
    if (polling.value) return
    start()
  }

  onBeforeUnmount(() => {
    stopped = true
    clearTimer()
  })

  return { order, status, attempts, polling, outcome, lastError, start, pause, resume }
}
