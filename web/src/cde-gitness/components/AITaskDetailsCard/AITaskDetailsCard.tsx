import React from 'react'
import { Layout, Text, Utils } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { IconName } from '@harnessio/icons'
import { Circle, Cloud } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import { getIconByRepoType } from 'cde-gitness/utils/SelectRepository.utils'
import getProviderIcon from 'cde-gitness/utils/InfraProvider.utils'
import { getIDEOption, TaskStatus } from 'cde-gitness/constants'
import type { TypesGitspaceConfig, EnumAITaskState, TypesAITask } from 'services/cde'
import { getRepoNameFromURL } from '../../utils/SelectRepository.utils'
import ResourceDetails from '../ResourceDetails/ResourceDetails'
import css from './AITaskDetailsCard.module.scss'

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

export const AITaskDetailsCard = ({
  data,
  taskState,
  standalone
}: {
  data: TypesGitspaceConfig | null
  taskState?: EnumAITaskState
  standalone?: boolean
}) => {
  const { getString } = useStrings()
  const repoName = getRepoNameFromURL(data?.code_repo_url || '') || ''
  const branch = data?.branch || ''
  const {
    infra_provider_type: providerType = '',
    config_identifier: providerConfigId = '',
    config_name,
    cpu = '',
    memory = '',
    disk = ''
  } = data?.resource || {}
  const providerIcon = getProviderIcon(providerType)
  const providerDisplayName = config_name || providerConfigId
  const md = (data?.resource?.metadata as Record<string, string> | null | undefined) || undefined
  const cores = md?.['cores']?.toString().trim() || md?.['cpu_cores']?.toString().trim()

  const specParts: string[] = []
  if (cpu) {
    const cpuLabel = /cpu/i.test(cpu) ? cpu : `${cpu} CPU`
    specParts.push(cpuLabel)
  }
  if (cores) {
    const coresLabel = /core/i.test(cores) ? cores : `${cores} cores`
    specParts.push(coresLabel)
  }
  if (memory) {
    const memHasUnit = /[a-zA-Z]/.test(memory)
    const memoryValue = memHasUnit ? memory : `${memory}GB`
    specParts.push(`${memoryValue} memory`)
  }
  if (disk) {
    const diskHasUnit = /[a-zA-Z]/.test(disk)
    const diskValue = diskHasUnit ? disk : `${disk}GB`
    specParts.push(`${diskValue} disk size`)
  }
  const specs = specParts.length ? `${specParts.join(', ')}` : undefined

  const ideOption = getIDEOption(data?.ide, getString)

  const statusColor = getStatusColor(taskState)
  const isRunning = taskState === 'running'
  const customProps = isRunning
    ? { icon: 'loading' as IconName, iconProps: { color: Color.PRIMARY_4 } }
    : { icon: undefined }

  return (
    <>
      <Layout.Horizontal width={'90%'} className={css.detailsContainer} padding={{ bottom: 'xlarge', top: 'xlarge' }}>
        <Layout.Vertical
          spacing="small"
          flex={{ justifyContent: 'center', alignItems: 'flex-start' }}
          className={css.marginLeftContainer}>
          <Text className={css.rowHeaders}>{getString('cde.status')}</Text>
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
            {!isRunning && <Circle height={10} width={10} color={statusColor} fill={statusColor} />}
            <Text {...customProps} className={css.statusText}>
              {getStatusText(taskState)}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>

        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text className={css.rowHeaders}>Gitspace</Text>
          <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
            {ideOption?.icon && <img src={ideOption.icon} className={css.standardIcon} alt={'ide'} />}
            <Text lineClamp={1} className={css.providerText} title={data?.identifier}>
              {data?.identifier || data?.name || '—'}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>

        <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
          <Text className={css.rowHeaders}>{getString('cde.repositoryAndBranch')}</Text>
          <Layout.Horizontal
            spacing="small"
            flex={{ alignItems: 'center', justifyContent: 'start' }}
            onClick={e => {
              e.preventDefault()
              e.stopPropagation()
            }}>
            {getIconByRepoType({ repoType: data?.code_repo_type, height: 20 })}
            <Text
              title={'RepoName'}
              className={css.clickableText}
              onClick={() => window.open(data?.code_repo_url || '', '_blank')}>
              {repoName}
            </Text>
            <Text color={Color.PRIMARY_7}>:</Text>
            <Text
              iconProps={{ size: 10 }}
              icon="git-branch"
              className={css.clickableText}
              onClick={() => window.open(data?.branch_url || '', '_blank')}>
              {branch}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>

        {!standalone && (
          <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
            <Text className={css.rowHeaders}>{getString('cde.infraProvider')}</Text>
            <Layout.Horizontal spacing={'small'} flex={{ alignItems: 'center', justifyContent: 'start' }}>
              {providerIcon ? (
                <img src={providerIcon} className={css.standardIcon} alt={'provider icon'} />
              ) : (
                <Cloud className={css.standardIcon} />
              )}
              <Text lineClamp={1} className={css.providerText} title={providerDisplayName}>
                {providerDisplayName || '—'}
              </Text>
            </Layout.Horizontal>
          </Layout.Vertical>
        )}

        {!standalone && (
          <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
            <Text className={css.rowHeaders}>{getString('cde.regionMachineType')}</Text>
            <ResourceDetails resource={data?.resource} />
            {specs && (
              <Text className={css.metricsText} title={specs}>
                {specs}
              </Text>
            )}
          </Layout.Vertical>
        )}
      </Layout.Horizontal>
    </>
  )
}

export default AITaskDetailsCard
