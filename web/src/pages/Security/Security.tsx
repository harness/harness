/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  DropDown,
  FlexExpander,
  Layout,
  PageBody,
  TableV2,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { Render } from 'react-jsx-match'
import { isEmpty, noop } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { StringKeys, useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { StatusBadge } from 'components/StatusBadge/StatusBadge'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { LIST_FETCHING_LIMIT, PageBrowserProps, getErrorMessage, voidFn } from 'utils/Utils'
import { useTaskNumbersByOriginRef } from 'utils/TaskUtils'
import {
  SHIELD_FINDING_KINDS,
  SHIELD_FINDING_STATES,
  SHIELD_MIN_RISK_OPTIONS,
  SHIELD_SCAN_STATUS_COLORS,
  SHIELD_STATE_COLORS,
  TypesShieldFinding,
  TypesShieldScan,
  getRiskColors,
  getShieldApiPath
} from 'utils/ShieldUtils'
import { FindingStateDialog } from './FindingStateDialog'
import css from './Security.module.scss'

const SCANS_LIMIT = 5

export default function Security() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { showError, showSuccess } = useToaster()
  const browserParams = useQueryParams<PageBrowserProps>()
  const pageInit = browserParams.page ? parseInt(browserParams.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [stateFilter, setStateFilter] = useState('open')
  const [kindFilter, setKindFilter] = useState('')
  const [minRisk, setMinRisk] = useState('')
  const [target, setTarget] = useState<TypesShieldFinding | undefined>()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const repoPath = repoMetadata?.path as string
  const shieldPath = getShieldApiPath(repoMetadata?.path)
  const taskNumbers = useTaskNumbersByOriginRef(repoMetadata?.path, 'shield')

  const {
    data: findings,
    error: findingsError,
    loading: findingsLoading,
    refetch: refetchFindings,
    response
  } = useGet<TypesShieldFinding[]>({
    path: `${shieldPath}/findings`,
    queryParams: {
      page,
      limit: LIST_FETCHING_LIMIT,
      ...(stateFilter && { state: stateFilter }),
      ...(kindFilter && { kind: kindFilter }),
      ...(minRisk && { min_risk: minRisk })
    },
    debounce: 500,
    lazy: !repoMetadata
  })

  const {
    data: scans,
    refetch: refetchScans,
    loading: scansLoading
  } = useGet<TypesShieldScan[]>({
    path: `${shieldPath}/scans`,
    queryParams: { page: 1, limit: SCANS_LIMIT },
    lazy: !repoMetadata
  })

  const { mutate: createScan, loading: scanning } = useMutate<TypesShieldScan>({
    verb: 'POST',
    path: `${shieldPath}/scans`
  })

  const onScan = () => {
    createScan({})
      .then(() => {
        showSuccess(getString('shield.scanStarted'))
        refetchScans()
      })
      .catch(err => showError(getErrorMessage(err)))
  }

  const columns: Column<TypesShieldFinding>[] = useMemo(
    () => [
      {
        Header: getString('shield.risk'),
        width: '70px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) => (
          <StatusBadge
            label={String(row.original.risk_score ?? 0)}
            colors={getRiskColors(row.original.risk_score)}
            title={getString('shield.riskTitle', { cvss: row.original.cvss_score ?? 0 })}
          />
        )
      },
      {
        Header: getString('shield.vulnerability'),
        width: '160px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) => (
          <Text color={Color.GREY_800} lineClamp={1} font={{ weight: 'semi-bold' }} tooltip={row.original.summary}>
            {row.original.vuln_id}
          </Text>
        )
      },
      {
        Header: getString('shield.package'),
        width: 'calc(100% - 810px)',
        Cell: ({ row }: CellProps<TypesShieldFinding>) => (
          <Text lineClamp={1} tooltip={row.original.manifest_path}>
            {row.original.package
              ? `${row.original.package}${row.original.installed_version ? `@${row.original.installed_version}` : ''}`
              : row.original.manifest_path || '-'}
          </Text>
        )
      },
      {
        Header: getString('shield.fixVersion'),
        width: '110px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) => <Text>{row.original.min_fixed_version || '-'}</Text>
      },
      {
        Header: getString('shield.kind'),
        width: '100px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) => (
          <Text>{getString(`shield.kindValue.${row.original.kind}` as StringKeys)}</Text>
        )
      },
      {
        Header: getString('shield.state'),
        width: '120px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) =>
          row.original.state ? (
            <StatusBadge
              label={getString(`shield.stateValue.${row.original.state}` as StringKeys)}
              colors={SHIELD_STATE_COLORS[row.original.state]}
              title={row.original.state_reason}
            />
          ) : null
      },
      {
        Header: getString('shield.task'),
        width: '90px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) => {
          const number = row.original.fingerprint ? taskNumbers[row.original.fingerprint] : undefined
          if (number) {
            return <Link to={routes.toCODETask({ repoPath, taskNumber: String(number) })}>TASK-{number}</Link>
          }
          return row.original.task_id ? (
            <Link to={routes.toCODETasks({ repoPath })}>{getString('shield.taskLinked')}</Link>
          ) : (
            <Text>-</Text>
          )
        }
      },
      {
        Header: '',
        width: '160px',
        Cell: ({ row }: CellProps<TypesShieldFinding>) =>
          row.original.state === 'open' ? (
            <Button
              size={ButtonSize.SMALL}
              variation={ButtonVariation.TERTIARY}
              text={getString('shield.dismiss')}
              onClick={() => setTarget(row.original)}
            />
          ) : null
      }
    ],
    [getString, routes, repoPath, taskNumbers]
  )

  const allItem = (label: string) => ({ label, value: '' })
  const stateItems = [
    allItem(getString('shield.allStates')),
    ...SHIELD_FINDING_STATES.map(s => ({ label: getString(`shield.stateValue.${s}` as StringKeys), value: s }))
  ]
  const kindItems = [
    allItem(getString('shield.allKinds')),
    ...SHIELD_FINDING_KINDS.map(k => ({ label: getString(`shield.kindValue.${k}` as StringKeys), value: k }))
  ]
  const riskItems = [
    allItem(getString('shield.anyRisk')),
    ...SHIELD_MIN_RISK_OPTIONS.map(r => ({ label: getString('shield.minRisk', { risk: r }), value: String(r) }))
  ]
  const hasFilter = !!(stateFilter || kindFilter || minRisk)

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('shield.title')}
        dataTooltipId="repositorySecurity"
      />
      <PageBody error={getErrorMessage(error || findingsError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || findingsLoading || scansLoading} withBorder />

        <Render when={repoMetadata}>
          <Layout.Vertical>
            <Container padding="xlarge">
              <Layout.Horizontal className={css.filters} spacing="medium">
                <DropDown
                  value={stateFilter}
                  items={stateItems}
                  onChange={({ value }) => {
                    setStateFilter(value as string)
                    setPage(1)
                  }}
                />
                <DropDown
                  value={kindFilter}
                  items={kindItems}
                  onChange={({ value }) => {
                    setKindFilter(value as string)
                    setPage(1)
                  }}
                />
                <DropDown
                  value={minRisk}
                  items={riskItems}
                  onChange={({ value }) => {
                    setMinRisk(value as string)
                    setPage(1)
                  }}
                />
                <FlexExpander />
                <Button
                  variation={ButtonVariation.PRIMARY}
                  text={getString('shield.scanNow')}
                  icon="play"
                  disabled={scanning}
                  onClick={onScan}
                />
              </Layout.Horizontal>
            </Container>

            <Container padding={{ left: 'xlarge', right: 'xlarge', bottom: 'xlarge' }}>
              {isEmpty(findings) ? (
                <NoResultCard
                  forSearch={false}
                  forFilter={hasFilter}
                  emptyFilterMessage={getString('shield.noFindingsForFilter')}
                  message={getString('shield.noFindings')}
                  buttonText={getString('shield.scanNow')}
                  onButtonClick={onScan}
                />
              ) : (
                <>
                  <TableV2<TypesShieldFinding>
                    className={css.table}
                    columns={columns}
                    data={findings || []}
                    getRowClassName={() => css.row}
                    onRowClick={noop}
                  />
                  <ResourceListingPagination response={response} page={page} setPage={setPage} />
                </>
              )}
            </Container>

            <Container padding={{ left: 'xlarge', right: 'xlarge', bottom: 'xlarge' }}>
              <Text font={{ variation: FontVariation.H5 }} className={css.sectionTitle}>
                {getString('shield.scans')}
              </Text>
              {isEmpty(scans) ? <Text>{getString('shield.noScans')}</Text> : <ScansTable scans={scans || []} />}
            </Container>
          </Layout.Vertical>
        </Render>
      </PageBody>

      <Render when={target}>
        <FindingStateDialog
          repoPath={repoMetadata?.path}
          finding={target as TypesShieldFinding}
          onClose={() => setTarget(undefined)}
          onUpdated={() => {
            setTarget(undefined)
            refetchFindings()
          }}
        />
      </Render>
    </Container>
  )
}

const ScansTable: React.FC<{ scans: TypesShieldScan[] }> = ({ scans }) => {
  const { getString } = useStrings()
  const columns: Column<TypesShieldScan>[] = useMemo(
    () => [
      {
        Header: getString('shield.started'),
        width: '160px',
        Cell: ({ row }: CellProps<TypesShieldScan>) => (
          <TimePopoverWithLocal time={row.original.started || row.original.created || 0} inline={false} />
        )
      },
      {
        Header: getString('shield.state'),
        width: '110px',
        Cell: ({ row }: CellProps<TypesShieldScan>) =>
          row.original.status ? (
            <StatusBadge
              label={getString(`shield.scanStatus.${row.original.status}` as StringKeys)}
              colors={SHIELD_SCAN_STATUS_COLORS[row.original.status]}
              title={row.original.error}
            />
          ) : null
      },
      {
        Header: getString('shield.trigger'),
        width: '110px',
        Cell: ({ row }: CellProps<TypesShieldScan>) => (
          <Text>{getString(`shield.triggerValue.${row.original.trigger}` as StringKeys)}</Text>
        )
      },
      {
        Header: getString('shield.commit'),
        width: '100px',
        Cell: ({ row }: CellProps<TypesShieldScan>) => <Text>{(row.original.commit_sha || '').slice(0, 8)}</Text>
      },
      {
        Header: getString('shield.packages'),
        width: '90px',
        Cell: ({ row }: CellProps<TypesShieldScan>) => <Text>{row.original.packages_count ?? 0}</Text>
      },
      {
        Header: getString('shield.findings'),
        width: '90px',
        Cell: ({ row }: CellProps<TypesShieldScan>) => <Text>{row.original.findings_count ?? 0}</Text>
      },
      {
        Header: getString('shield.error'),
        width: 'calc(100% - 660px)',
        Cell: ({ row }: CellProps<TypesShieldScan>) => (
          <Text lineClamp={1} color={Color.RED_600} tooltip={row.original.error}>
            {row.original.error || ''}
          </Text>
        )
      }
    ],
    [getString]
  )
  return (
    <TableV2<TypesShieldScan>
      className={css.table}
      columns={columns}
      data={scans}
      getRowClassName={() => css.row}
      onRowClick={noop}
    />
  )
}
