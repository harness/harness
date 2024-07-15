import { Color } from '@harnessio/design-system'
import { Layout, Text } from '@harnessio/uicore'
import React from 'react'
import ReactTimeago from 'react-timeago'
import { Circle } from 'iconoir-react'
import type { IconName } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import { getIconByRepoType } from 'cde/components/CreateGitspace/components/SelectRepository/SelectRepository.utils'
import type { TypesGitspaceConfig } from 'cde-gitness/services'
import { GitspaceStatus } from 'cde/constants'
import { getStatusColor, getStatusText } from '../GitspaceListing/ListGitspaces'

export const DetailsCard = ({ data }: { data: TypesGitspaceConfig | null; loading?: boolean }) => {
  const { getString } = useStrings()
  const { branch, state, name, code_repo_url, code_repo_type, instance } = data || {}
  const color = getStatusColor(state)
  const customProps =
    state === GitspaceStatus.STARTING
      ? {
          icon: 'loading' as IconName,
          iconProps: { color: Color.PRIMARY_4 }
        }
      : { icon: undefined }
  return (
    <>
      <Layout.Horizontal
        width={'80%'}
        flex={{ justifyContent: 'space-between' }}
        padding={{ bottom: 'xlarge', top: 'xlarge' }}>
        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text>{getString('cde.status')}</Text>
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
            {state !== GitspaceStatus.STARTING && <Circle height={10} width={10} color={color} fill={color} />}
            <Text
              {...customProps}
              color={Color.BLACK}
              title={name}
              font={{ align: 'left', size: 'normal', weight: 'semi-bold' }}>
              {getStatusText(getString, state)}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>

        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text>{getString('cde.repository.repo')}</Text>
          <Layout.Horizontal
            spacing="small"
            flex={{ alignItems: 'center', justifyContent: 'start' }}
            onClick={e => {
              e.preventDefault()
              e.stopPropagation()
            }}>
            {getIconByRepoType({ repoType: code_repo_type, height: 20 })}
            <Text
              title={'RepoName'}
              color={Color.PRIMARY_7}
              margin={{ left: 'small' }}
              style={{ cursor: 'pointer' }}
              font={{ align: 'left', size: 'normal' }}
              onClick={() => window.open(code_repo_url, '_blank')}>
              {name}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>

        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text>{getString('branch')}</Text>
          <Text
            iconProps={{ size: 10 }}
            color={Color.PRIMARY_7}
            icon="git-branch"
            style={{ cursor: 'pointer' }}
            onClick={() => window.open(code_repo_url, '_blank')}>
            {branch}
          </Text>
        </Layout.Vertical>

        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text>{getString('cde.lastActivated')}</Text>
          {instance?.last_used ? (
            <ReactTimeago date={instance?.last_used || 0} />
          ) : (
            <Text color={Color.GREY_500}>{getString('cde.na')}</Text>
          )}
        </Layout.Vertical>
      </Layout.Horizontal>
    </>
  )
}
