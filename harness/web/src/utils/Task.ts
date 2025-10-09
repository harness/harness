type Task = () => void
type RunnerType = 'requestAnimationFrame' | 'requestIdleCallback' | 'setTimeout'

function createTaskPool(type: RunnerType = 'requestAnimationFrame', blockSize: number) {
  const tasks: Array<[Task, number]> = []
  const runner = type === 'requestAnimationFrame' ? requestAnimationFrame : window.requestIdleCallback

  let id = 1
  let running = false

  const scheduleTask = (task: Task, priority = tasks.length) => {
    id = id + 1 < Number.MAX_SAFE_INTEGER ? id + 1 : 1
    tasks.splice(priority, 0, [task, id])
    run()
    return id
  }

  const run = () => {
    if (!running && tasks.length) {
      running = true

      const block = tasks.splice(0, blockSize)

      const blockTracking = block.map(_ => {
        let resolve: (value: unknown) => void = noop
        new Promise(_resolve => (resolve = _resolve))
        return { resolve }
      })

      block.forEach((task, idx) => runner(() => blockTracking[idx].resolve(task[0]?.())))

      Promise.all(blockTracking).then(() => {
        running = false
        if (tasks.length) run()
      })
    }
  }

  const cancelTask = (taskId: number) => {
    const index = tasks.findIndex(([, _id]) => _id === taskId)
    if (index > -1) tasks.splice(index, 1)
  }

  return { scheduleTask, cancelTask }
}

const EXECUTION_BLOCK_SIZE = 10
const noop = () => 0

export const createRequestAnimationFrameTaskPool = (blockSize = EXECUTION_BLOCK_SIZE) =>
  createTaskPool('requestAnimationFrame', blockSize)
export const createRequestIdleCallbackTaskPool = (blockSize = EXECUTION_BLOCK_SIZE) =>
  createTaskPool('requestIdleCallback', blockSize)

// Polly fill requestIdleCallback/cancelIdleCallback for Safari
window.requestIdleCallback =
  window.requestIdleCallback ||
  function (cb) {
    const start = Date.now()
    return setTimeout(function () {
      cb({
        didTimeout: false,
        timeRemaining: function () {
          return Math.max(0, 50 - (Date.now() - start))
        }
      })
    }, 1)
  }

window.cancelIdleCallback =
  window.cancelIdleCallback ||
  function (id) {
    clearTimeout(id)
  }
