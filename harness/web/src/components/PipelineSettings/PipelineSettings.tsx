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

import { Container, PageBody } from '@harnessio/uicore'
import React from 'react'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import cx from 'classnames'
import PipelineSettingsPageHeader from 'components/PipelineSettingsPageHeader/PipelineSettingsPageHeader'
import { useStrings } from 'framework/strings'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { routes, type CODEProps } from 'RouteDefinitions'
import { getErrorMessage, voidFn } from 'utils/Utils'
import type { TypesPipeline } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import PipelineSettingsTab from 'components/PipelineSettingsTab/PipelineSettingsTab'
import PipelineTriggersTabs from 'components/PipelineTriggersTab/PipelineTriggersTab'
import css from './PipelineSettings.module.scss'

export enum TabOptions {
  SETTINGS = 'Settings',
  TRIGGERS = 'Triggers'
}

const PipelineSettings = () => {
  const { getString } = useStrings()

  const { pipeline } = useParams<CODEProps>()
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()

  const {
    data: pipelineData,
    error: pipelineError,
    loading: pipelineLoading
  } = useGet<TypesPipeline>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pipelines/${pipeline}`,
    lazy: !repoMetadata
  })

  const [selectedTab, setSelectedTab] = React.useState<TabOptions>(TabOptions.SETTINGS)

  return (
    <Container className={css.main}>
      <PipelineSettingsPageHeader
        repoMetadata={repoMetadata}
        title={`${pipeline} settings`}
        dataTooltipId="pipelineSettings"
        selectedTab={selectedTab}
        setSelectedTab={setSelectedTab}
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('pageTitle.pipelines'),
              url: routes.toCODEPipelines({ repoPath: repoMetadata.path as string })
            },
            {
              label: pipeline as string,
              url: routes.toCODEExecutions({ repoPath: repoMetadata.path as string, pipeline: pipeline as string })
            }
          ]
        }
      />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error || pipelineError) : null}
        retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || pipelineLoading} withBorder={!!pipeline} />
        {selectedTab === TabOptions.SETTINGS && (
          <PipelineSettingsTab
            pipeline={pipeline as string}
            repoPath={repoMetadata?.path as string}
            yamlPath={pipelineData?.config_path as string}
          />
        )}
        {selectedTab === TabOptions.TRIGGERS && (
          <PipelineTriggersTabs pipeline={pipeline as string} repoPath={repoMetadata?.path as string} />
        )}
      </PageBody>
    </Container>
  )
}

export default PipelineSettings
