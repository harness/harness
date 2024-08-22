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
import type { FormikProps } from 'formik'
import classNames from 'classnames'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Text } from '@harnessio/uicore'

import { useAppStore } from '@ar/hooks'
import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import { Separator } from '@ar/components/Separator/Separator'
import CollapseContainer from '@ar/components/CollapseContainer/CollapseContainer'
import { RepositoryProviderContext } from '@ar/pages/repository-details/context/RepositoryProvider'

import UpstreamProxyDetailsFormContent from './UpstreamProxyDetailsFormContent'
import UpstreamProxyAuthenticationFormContent from './UpstreamProxyAuthenticationFormContent'
import UpstreamProxyIncludeExcludePatternFormContent from './UpstreamProxyIncludeExcludePatternFormContent'
import UpstreamProxyCleanupPoliciesFormContent from './UpstreamProxyCleanupPoliciesFormContent'
import type { UpstreamRegistryRequest } from '../../types'

import css from './FormContent.module.scss'

interface UpstreamProxyConfigurationFormContentProps {
  formikProps: FormikProps<UpstreamRegistryRequest>
  readonly: boolean
}

export default function UpstreamProxyConfigurationFormContent(
  props: UpstreamProxyConfigurationFormContentProps
): JSX.Element {
  const { formikProps, readonly } = props
  const { parent } = useAppStore()
  const { getString } = useStrings()
  const { setIsDirty } = useContext(RepositoryProviderContext)
  const { dirty, values } = formikProps
  const [isCollapsedAdvancedConfig] = useState(getInitialStateOfCollapse())

  useEffect(() => {
    setIsDirty(dirty)
  }, [dirty])

  function getInitialStateOfCollapse(): boolean {
    const isIncludesPatternAdded = !!values.allowedPattern?.length
    const isExcludesPatternAdded = !!values.blockedPattern?.length
    const isCleanupPoliciesAdded = !!values.cleanupPolicy?.length
    return isIncludesPatternAdded || isExcludesPatternAdded || isCleanupPoliciesAdded
  }

  return (
    <Container padding="xxlarge">
      <Container>
        <Text className={css.cardHeading} font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('upstreamProxyDetails.form.title')}
        </Text>
      </Container>
      <Card className={classNames(css.cardContainer, css.marginTopLarge)}>
        <Layout.Vertical>
          <UpstreamProxyDetailsFormContent isEdit formikProps={formikProps} readonly={readonly} />
          <UpstreamProxyAuthenticationFormContent formikProps={formikProps} readonly={readonly} />
        </Layout.Vertical>
      </Card>
      {parent === Parent.Enterprise && (
        <CollapseContainer
          className={css.marginTopLarge}
          title={getString('repositoryDetails.repositoryForm.advancedOptionsTitle')}
          subTitle={getString('repositoryDetails.repositoryForm.enterpriseAdvancedOptionsSubTitle')}
          initialState={isCollapsedAdvancedConfig}>
          <Card className={classNames(css.cardContainer)}>
            <UpstreamProxyIncludeExcludePatternFormContent formikProps={formikProps} isEdit readonly={readonly} />
            <Separator />
            <Container className={css.cleanupPoliciesContainer}>
              <UpstreamProxyCleanupPoliciesFormContent formikProps={formikProps} isEdit disabled={readonly} />
            </Container>
          </Card>
        </CollapseContainer>
      )}
    </Container>
  )
}
