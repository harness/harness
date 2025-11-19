import React from 'react'
import { Avatar, Container, Layout, TableV2, Text, Utils } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Renderer } from 'react-table'
import type { IconName } from '@harnessio/icons'
import { useHistory } from 'react-router-dom'
import moment from 'moment'
import cx from 'classnames'
import { Circle } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { usePaginationProps } from 'cde-gitness/hooks/usePaginationProps'
import { getIDEOption, TaskStatus, getIconByAgentType } from 'cde-gitness/constants'
import { AIAgentEnum } from 'cde-gitness/constants/index'
import { getIconByRepoType, getRepoFromURL } from 'cde-gitness/utils/SelectRepository.utils'
import CopyButton from 'cde-gitness/components/CopyButton/CopyButton'
import type { TypesAITask } from 'services/cde'
import css from './ListAITasks.module.scss'

const getStatusColor = (status?: TypesAITask['state']) => {
  switch (status) {
    case TaskStatus.COMPLETED:
      return Utils.getRealCSSColor(Color.SUCCESS)
    case TaskStatus.ERROR:
      return Utils.getRealCSSColor(Color.ERROR)
    case TaskStatus.UNINITIALIZED:
      return Utils.getRealCSSColor(Color.BLACK)
    default:
      return Utils.getRealCSSColor(Color.BLACK)
  }
}

const getStatusText = (status?: TypesAITask['state']) => {
  switch (status) {
    case TaskStatus.RUNNING:
      return 'Running'
    case TaskStatus.COMPLETED:
      return 'Completed'
    case TaskStatus.ERROR:
      return 'Error'
    case TaskStatus.UNINITIALIZED:
    default:
      return 'Uninitialized'
  }
}

const RenderTaskName: Renderer<CellProps<TypesAITask>> = ({ row }) => {
  const original = row.original
  const { display_name, identifier, id } = original
  const taskId = identifier || String(id)
  const MAX_TITLE_LENGTH = 100
  const MAX_PROMPT_LENGTH = 60

  const baseTitle = display_name || original?.initial_prompt
  const titleText =
    baseTitle && baseTitle.length >= MAX_TITLE_LENGTH ? `${baseTitle.slice(0, MAX_TITLE_LENGTH)}...` : baseTitle
  const initialPrompt = original?.initial_prompt
  const initialPromptText =
    initialPrompt && initialPrompt.length >= MAX_PROMPT_LENGTH
      ? `${initialPrompt.slice(0, MAX_PROMPT_LENGTH)}...`
      : initialPrompt

  return (
    <Layout.Vertical spacing="xsmall" className={css.taskCellContainer}>
      <Text
        color={Color.BLACK}
        lineClamp={1}
        title={baseTitle}
        font={{ align: 'left', size: 'normal', weight: 'semi-bold' }}>
        {titleText}
      </Text>

      <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'start' }}>
        <Layout.Horizontal spacing="none" flex={{ alignItems: 'center', justifyContent: 'start' }}>
          <Text font={{ size: 'small' }} lineClamp={1} color={Color.GREY_800}>
            id: {taskId}
          </Text>
          <CopyButton value={taskId} className={css.copyBtn} />
        </Layout.Horizontal>
        <Container className={css.seperator} />
        <Text font={{ variation: FontVariation.SMALL }} lineClamp={1} color={Color.GREY_800} title={initialPrompt}>
          initial_prompt: {initialPromptText}
        </Text>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

const RenderAIAgent: Renderer<CellProps<TypesAITask>> = ({ row }) => {
  const { ai_agent } = row.original
  const { getString } = useStrings()
  const label = ai_agent === AIAgentEnum.CLAUDE_CODE ? getString('cde.aiTasks.create.claudeAI') : ai_agent || '—'
  const agentIcon = getIconByAgentType(ai_agent)
  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'start' }}>
      {agentIcon && <img src={agentIcon} className={css.aiAgentIcon} />}
      <Text color={Color.BLACK}>{label}</Text>
    </Layout.Horizontal>
  )
}

const RenderContextDetails: Renderer<CellProps<TypesAITask>> = ({ row }) => {
  const { getString } = useStrings()
  const original = row.original
  const {
    code_repo_type: repoType,
    code_repo_url: repoURL,
    branch: branchName = '—',
    branch_url: branchURL,
    identifier: gitspaceId,
    ide
  } = original?.gitspace_config || {}
  const repoDisplay = getRepoFromURL(repoURL) || '—'
  const ideItem = getIDEOption(ide, getString)

  return (
    <Layout.Vertical spacing="xsmall">
      {gitspaceId ? (
        <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center', justifyContent: 'start' }}>
          {ideItem?.icon && <img src={ideItem.icon} className={css.gitspaceIcon} />}
          <Text color={Color.GREY_500} lineClamp={1} font={{ variation: FontVariation.SMALL }} title={gitspaceId}>
            {gitspaceId}
          </Text>
        </Layout.Horizontal>
      ) : (
        <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
          —
        </Text>
      )}

      <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'start' }}>
        <Layout.Horizontal
          spacing="small"
          flex={{ alignItems: 'center', justifyContent: 'start' }}
          className={cx({ [css.isUrl]: !!repoURL })}
          onClick={e => {
            if (!repoURL) return
            e.preventDefault()
            e.stopPropagation()
            window.open(repoURL, '_blank')
          }}>
          <Container height={16} width={16}>
            {getIconByRepoType({ repoType, height: 16 })}
          </Container>
          <Text
            lineClamp={1}
            color={repoURL ? Color.PRIMARY_7 : Color.GREY_500}
            title={repoDisplay}
            font={{ variation: FontVariation.BODY }}>
            {repoDisplay}
          </Text>
        </Layout.Horizontal>

        <Layout.Horizontal
          spacing="small"
          flex={{ alignItems: 'center', justifyContent: 'start' }}
          className={cx({ [css.isUrl]: !!repoURL })}
          onClick={e => {
            if (!branchURL) return
            e.preventDefault()
            e.stopPropagation()
            window.open(branchURL, '_blank')
          }}>
          <Text color={branchURL ? Color.PRIMARY_7 : Color.GREY_500}>:</Text>
          <Text
            lineClamp={1}
            icon="git-branch"
            iconProps={{ size: 12 }}
            color={branchURL ? Color.PRIMARY_7 : Color.GREY_500}
            title={branchName}
            font={{ variation: FontVariation.BODY }}>
            {branchName}
          </Text>
        </Layout.Horizontal>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

const RenderAITaskStatus: Renderer<CellProps<TypesAITask>> = ({ row }) => {
  const { state } = row.original
  const color = getStatusColor(state)
  const isRunning = state === TaskStatus.RUNNING
  const customProps = isRunning
    ? { icon: 'loading' as IconName, iconProps: { color: Color.PRIMARY_4 } }
    : { icon: undefined }

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'start' }}>
      {!isRunning && <Circle height={10} width={10} color={color} fill={color} />}
      <Text {...customProps} color={Color.BLACK} font={{ weight: 'semi-bold' }}>
        {getStatusText(state)}
      </Text>
    </Layout.Horizontal>
  )
}

const OwnerAndCreatedAt: Renderer<CellProps<TypesAITask>> = ({ row }) => {
  const { created } = row.original
  const { user_display_name: displayName = '—', user_email: userEmail = '—' } = row.original?.gitspace_config || {}
  return (
    <Layout.Vertical spacing="medium" flex={{ alignItems: 'start', justifyContent: 'center' }}>
      <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'center' }}>
        <Avatar size="small" name={displayName} email={userEmail} />
        <Text lineClamp={1} font={{ variation: FontVariation.SMALL }} color={Color.GREY_800}>
          {displayName}
        </Text>
      </Layout.Horizontal>
      <Text className={css.ownerCreatedAt} font={{ variation: FontVariation.SMALL }} color={Color.GREY_800}>
        {created ? moment(created).format('DD MMM, YYYY hh:mma') : '—'}
      </Text>
    </Layout.Vertical>
  )
}

interface PageConfigProps {
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

export const ListAITasks = ({
  data,
  hasFilter,
  gotoPage,
  onPageSizeChange,
  pageConfig
}: {
  data: TypesAITask[]
  hasFilter: boolean
  gotoPage: (pageNumber: number) => void
  onPageSizeChange?: (newSize: number) => void
  pageConfig: PageConfigProps
}) => {
  const history = useHistory()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { page, pageSize, totalItems, totalPages } = pageConfig
  const safePageCount = Math.max(1, totalPages || 0)
  const safePageIndex = Math.min(Math.max(0, page - 1), safePageCount - 1)

  const paginationProps = usePaginationProps({
    itemCount: totalItems,
    pageSize: pageSize,
    pageCount: safePageCount,
    pageIndex: safePageIndex,
    gotoPage,
    onPageSizeChange
  })
  const safeData = Array.isArray(data) ? data : []

  return (
    <Container>
      {(safeData || hasFilter) && (
        <TableV2<TypesAITask>
          className={css.cdeTable}
          onRowClick={row => {
            history.push(
              routes.toCDEAITaskDetail({
                space,
                aitaskId: row?.identifier || String(row?.id || '')
              })
            )
          }}
          columns={[
            { id: 'tasks', Header: getString('cde.aiTasks.tasks'), Cell: RenderTaskName },
            { id: 'agent', Header: getString('cde.aiTasks.listing.aiAgent'), Cell: RenderAIAgent },
            { id: 'context', Header: getString('cde.aiTasks.listing.contextDetails'), Cell: RenderContextDetails },
            { id: 'status', Header: getString('cde.status'), Cell: RenderAITaskStatus },
            { id: 'ownercreated', Header: getString('cde.listing.ownerAndCreated'), Cell: OwnerAndCreatedAt }
          ]}
          data={safeData}
          pagination={paginationProps}
        />
      )}
    </Container>
  )
}

export default ListAITasks
