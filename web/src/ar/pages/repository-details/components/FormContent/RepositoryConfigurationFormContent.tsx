/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useContext, useEffect, useState } from 'react'
import classNames from 'classnames'
import { FormikContextType, connect } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Text } from '@harnessio/uicore'

import { useAppStore } from '@ar/hooks'
import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { Separator } from '@ar/components/Separator/Separator'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import CollapseContainer from '@ar/components/CollapseContainer/CollapseContainer'
import { RepositoryProviderContext } from '../../context/RepositoryProvider'
import RepositoryDetailsFormContent from './RepositoryDetailsFormContent'
import RepositoryIncludeExcludePatternFormContent from './RepositoryIncludeExcludePatternFormContent'
import RepositoryUpstreamProxiesFormContent from './RepositoryUpstreamProxiesFormContent'
import RepositoryCleanupPoliciesFormContent from './RepositoryCleanupPoliciesFormContent'
import css from './FormContent.module.scss'

interface RepositoryConfigurationFormContentProps {
  readonly: boolean
}

function RepositoryConfigurationFormContent(
  props: RepositoryConfigurationFormContentProps & { formik: FormikContextType<VirtualRegistryRequest> }
): JSX.Element {
  const { formik, readonly } = props
  const { getString } = useStrings()
  const { setIsDirty } = useContext(RepositoryProviderContext)
  const { parent } = useAppStore()
  const { dirty, values } = formik
  const [isCollapsedAdvancedConfig] = useState(getInitialStateOfCollapse())

  useEffect(() => {
    setIsDirty(dirty)
  }, [dirty])

  function getInitialStateOfCollapse(): boolean {
    const isUpstreamProxiesSelected = !!values.config?.upstreamProxies?.length
    const isIncludesPatternAdded = !!values.allowedPattern?.length
    const isExcludesPatternAdded = !!values.allowedPattern?.length
    const isCleanupPoliciesAdded = !!values.cleanupPolicy?.length
    return isUpstreamProxiesSelected || isIncludesPatternAdded || isExcludesPatternAdded || isCleanupPoliciesAdded
  }

  return (
    <Container padding="xxlarge">
      <Container>
        <Text className={css.cardHeading} font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('repositoryDetails.repositoryForm.title')}
        </Text>
      </Container>
      <Card className={classNames(css.cardContainer, css.marginTopLarge)}>
        <RepositoryDetailsFormContent isEdit readonly={readonly} />
      </Card>
      <CollapseContainer
        className={css.marginTopLarge}
        title={getString('repositoryDetails.repositoryForm.advancedOptionsTitle')}
        subTitle={
          parent === Parent.Enterprise
            ? getString('repositoryDetails.repositoryForm.enterpriseAdvancedOptionsSubTitle')
            : getString('repositoryDetails.repositoryForm.ossAdvancedOptionsSubTitle')
        }
        initialState={isCollapsedAdvancedConfig}>
        <Card className={classNames(css.cardContainer)}>
          <Container className={css.upstreamProxiesContainer}>
            <RepositoryUpstreamProxiesFormContent isEdit disabled={readonly} />
          </Container>
          {parent === Parent.Enterprise && (
            <>
              <Separator />
              <RepositoryIncludeExcludePatternFormContent isEdit disabled={readonly} />
              <Separator />
              <Container className={css.upstreamProxiesContainer}>
                <RepositoryCleanupPoliciesFormContent isEdit disabled={readonly} />
              </Container>
            </>
          )}
        </Card>
      </CollapseContainer>
    </Container>
  )
}

export default connect<RepositoryConfigurationFormContentProps, VirtualRegistryRequest>(
  RepositoryConfigurationFormContent
)
