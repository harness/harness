import { useEffect, useMemo, useState } from 'react'
import { stringSubstitute } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { useListStatusCheckResults } from 'services/code'
import { PRCheckExecutionState } from 'components/PRCheckExecutionStatus/PRCheckExecutionStatus'

export function usePRChecksDecision({
  repoMetadata,
  pullRequestMetadata
}: Partial<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>>) {
  const { data, error, refetch } = useListStatusCheckResults({
    repo_ref: `${repoMetadata?.path as string}/+`,
    commit_sha: pullRequestMetadata?.source_sha as string,
    lazy: !repoMetadata?.path || !pullRequestMetadata?.source_sha
  })
  const [count, setCount] = useState(DEFAULT_COUNTS)
  const { getString } = useStrings()
  const [color, setColor] = useState<Color>(Color.GREEN_500)
  const [background, setBackground] = useState<Color>(Color.GREEN_50)
  const [message, setMessage] = useState('')
  const [complete, setComplete] = useState(true)
  const status = useMemo(() => {
    let _status: PRCheckExecutionState | undefined
    const _count = { ...DEFAULT_COUNTS }
    const total = data?.length

    if (total) {
      for (const check of data) {
        switch (check.status) {
          case PRCheckExecutionState.ERROR:
          case PRCheckExecutionState.FAILURE:
          case PRCheckExecutionState.RUNNING:
          case PRCheckExecutionState.PENDING:
          case PRCheckExecutionState.SUCCESS:
            _count[check.status]++
            setCount({ ..._count })
            break
          default:
            console.error('Unrecognized PR check status', check) // eslint-disable-line no-console
            break
        }
      }

      if (_count.error) {
        _status = PRCheckExecutionState.ERROR
        setColor(Color.RED_900)
        setBackground(Color.RED_50)
        setMessage(stringSubstitute(getString('prChecks.error'), { count: _count.error, total }) as string)
      } else if (_count.failure) {
        _status = PRCheckExecutionState.FAILURE
        setColor(Color.RED_900)
        setBackground(Color.RED_50)
        setMessage(stringSubstitute(getString('prChecks.failure'), { count: _count.failure, total }) as string)
      } else if (_count.running) {
        _status = PRCheckExecutionState.RUNNING
        setColor(Color.ORANGE_900)
        setBackground(Color.ORANGE_100)
        setMessage(stringSubstitute(getString('prChecks.running'), { count: _count.running, total }) as string)
      } else if (_count.pending) {
        _status = PRCheckExecutionState.PENDING
        setColor(Color.GREY_600)
        setBackground(Color.GREY_100)
        setMessage(stringSubstitute(getString('prChecks.pending'), { count: _count.pending, total }) as string)
      } else if (_count.success) {
        _status = PRCheckExecutionState.SUCCESS
        setColor(Color.GREEN_800)
        setBackground(Color.GREEN_50)
        setMessage(stringSubstitute(getString('prChecks.success'), { count: _count.success, total }) as string)
      }

      setComplete(!_count.pending && !_count.running)
    } else {
      setComplete(false)
    }

    return _status
  }, [data]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const pollingFn = () => {
      if (repoMetadata?.path && pullRequestMetadata?.source_sha && !complete) {
        refetch().then(() => {
          interval = window.setTimeout(pollingFn, POLLING_INTERVAL)
        })
      }
    }
    let interval = window.setTimeout(pollingFn, POLLING_INTERVAL)
    return () => window.clearTimeout(interval)
  }, [repoMetadata?.path, pullRequestMetadata?.source_sha, complete]) // eslint-disable-line react-hooks/exhaustive-deps

  return {
    overallStatus: status,
    count,
    error,
    data,
    color,
    background,
    message
  }
}

export type PRChecksDecisionResult = ReturnType<typeof usePRChecksDecision>

const POLLING_INTERVAL = 10000

const DEFAULT_COUNTS = {
  error: 0,
  failure: 0,
  pending: 0,
  running: 0,
  success: 0
}
