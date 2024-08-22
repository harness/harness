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

import React, { useMemo, useState } from 'react'
import { Icon } from '@harnessio/icons'
import classNames from 'classnames'
import type { FormikProps } from 'formik'
import { get } from 'lodash-es'
import { FontVariation } from '@harnessio/design-system'
import { Button, Container, Layout, Text } from '@harnessio/uicore'
import { useGetAllRegistriesQuery } from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import { RepositoryConfigType } from '@ar/common/types'
import ReorderSelect from '@ar/components/ReorderSelect/ReorderSelect'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import type { UpstreamProxyPackageType } from '@ar/pages/upstream-proxy-details/types'
import useCreateUpstreamProxyModal from '@ar/pages/upstream-proxy-details/hooks/useCreateUpstreamProxyModal/useCreateUpstreamProxyModal'

import css from './UpstreamProxiesSelect.module.scss'

interface UpstreamProxiesSelectProps {
  formikProps: FormikProps<VirtualRegistryRequest>
  isEdit: boolean
  className: string
  name: string
  disabled: boolean
  packageType: UpstreamProxyPackageType
}

function UpstreamProxiesSelect(props: UpstreamProxiesSelectProps): JSX.Element {
  const { formikProps, className, disabled, name, packageType } = props
  const { getString } = useStrings()
  const selectedProxies = (get(formikProps.values, name) as string[]) || []
  const [showList, setShowList] = useState(!!selectedProxies?.length)
  const spaceRef = useGetSpaceRef('')

  const {
    data,
    isFetching: loading,
    error
  } = useGetAllRegistriesQuery(
    {
      space_ref: spaceRef,
      queryParams: {
        page: 0,
        size: 100,
        package_type: [packageType],
        type: RepositoryConfigType.UPSTREAM
      },
      stringifyQueryParamsOptions: {
        arrayFormat: 'comma'
      }
    },
    {
      enabled: showList
    }
  )

  const renderSelectedListNote = (): JSX.Element => {
    return (
      <Layout.Horizontal
        flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
        className={classNames(css.message, css.infoBackground)}
        spacing="xsmall">
        <Icon name="info-messaging" size={16} />
        <Text lineClamp={1} font={{ variation: FontVariation.SMALL_BOLD }}>
          {getString('repositoryDetails.upstreamProxiesSelectList.selectedList.note.label')}:
        </Text>
        <Text lineClamp={1} font={{ variation: FontVariation.SMALL }}>
          {getString('repositoryDetails.upstreamProxiesSelectList.selectedList.note.message')}
        </Text>
      </Layout.Horizontal>
    )
  }

  const items = useMemo(() => {
    if (loading) return [{ label: 'Loading...', value: '', disabled: true }]
    if (error) return [{ label: error.message, value: '', disabled: true }]
    if (data && Array.isArray(data?.content?.data?.registries)) {
      return data?.content?.data?.registries.map(each => ({
        label: each.identifier,
        value: each.identifier
      }))
    }
    return []
  }, [loading, error, data])

  const handleOnCreateUpstreamProxy = (): void => {
    queryClient.invalidateQueries(['GetAllRegistries'])
  }

  const [showCreateUpstreamProxyModal] = useCreateUpstreamProxyModal({
    onSuccess: handleOnCreateUpstreamProxy,
    defaultPackageType: packageType,
    isPackageTypeReadonly: true
  })

  if (!showList) {
    return (
      <Container flex={{ justifyContent: 'flex-start' }}>
        <Button
          className={css.addButton}
          minimal
          rightIcon="chevron-down"
          iconProps={{ size: 12 }}
          font={{ variation: FontVariation.FORM_LABEL }}
          text={getString('repositoryDetails.upstreamProxiesSelectList.addUpstreamProxies')}
          intent="primary"
          data-testid="configure-upstream-proxies"
          onClick={() => {
            setShowList(true)
          }}
          disabled={disabled}
        />
      </Container>
    )
  }

  return (
    <Container>
      <ReorderSelect
        className={className}
        items={items}
        name={name}
        formikProps={formikProps}
        selectListProps={{
          title: getString('repositoryDetails.upstreamProxiesSelectList.selectList.Title', {
            count: data?.content?.data?.registries?.length
          }),
          withSearch: true
        }}
        selectedListProps={{
          title: getString('repositoryDetails.upstreamProxiesSelectList.selectedList.Title', {
            count: selectedProxies?.length
          }),
          note: renderSelectedListNote()
        }}
        disabled={disabled}
      />
      <Button
        className={css.addButton}
        font={{ variation: FontVariation.FORM_LABEL }}
        icon="plus"
        iconProps={{ size: 12 }}
        minimal
        intent="primary"
        data-testid="add-patter"
        onClick={() => showCreateUpstreamProxyModal()}
        text={getString('repositoryDetails.upstreamProxiesSelectList.newUpstreamProxyLabel')}
        disabled={disabled}
      />
    </Container>
  )
}

export default UpstreamProxiesSelect
