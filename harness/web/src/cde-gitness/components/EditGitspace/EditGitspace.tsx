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
import {
  Button,
  ButtonVariation,
  Container,
  Formik,
  FormikForm,
  Layout,
  ModalDialog,
  Text,
  useToaster
} from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { CDEIDESelect } from 'cde-gitness/components/CDEIDESelect/CDEIDESelect'
import { CDESSHSelect } from 'cde-gitness/components/CDESSHSelect/CDESSHSelect'
import { SelectEditMachine } from 'cde-gitness/components/SelectEditMachine/SelectEditMachine'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { TypesGitspaceSettingsResponse, useListInfraProviderResources, useUpdateGitspace } from 'services/cde'
import { getIDEOption, getIDETypeOptions } from 'cde-gitness/constants'
import SolidInfoIcon from 'cde-gitness/assests/solidInfo.svg?url'
import { useFilteredIdeOptions } from 'cde-gitness/hooks/useFilteredIdeOptions'
import type { EnumIDEType, TypesInfraProviderResource } from '../../../services/cde'
import css from './EditGitspace.module.scss'

interface EditGitspaceProps {
  isOpen: boolean
  onClose: () => void
  gitspaceId: string
  gitspaceData: {
    name: string
    ide: EnumIDEType
    branch: string
    devcontainer_path: string
    ssh_token_identifier: string
    resource?: {
      identifier: string
      config_identifier: string
      name: string
      region: string
      disk: string
      cpu: string
      memory: string
      persistent_disk_type: string
    }
  }
  gitspaceSettings: TypesGitspaceSettingsResponse | null
  onGitspaceUpdated?: () => void
  isFromUsageDashboard?: boolean
  gitspacePath?: string
}

export const EditGitspace: React.FC<EditGitspaceProps> = ({
  isOpen,
  onClose,
  gitspaceId,
  gitspaceData,
  gitspaceSettings,
  onGitspaceUpdated,
  isFromUsageDashboard = false,
  gitspacePath
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  const space = useGetSpaceParam()
  const {
    accountIdentifier = '',
    orgIdentifier = '',
    projectIdentifier = ''
  } = useGetCDEAPIParams({
    gitspacePath,
    fromUsageDashboard: isFromUsageDashboard
  })

  const { showSuccess, showError } = useToaster()
  const [isLoading, setIsLoading] = useState(false)
  const ideOptions = getIDETypeOptions(getString) ?? []

  const { data: infraResources, loading: loadingResources } = useListInfraProviderResources({
    accountIdentifier: accountIdentifier,
    infraprovider_identifier: gitspaceData?.resource?.config_identifier || '',
    queryParams: {
      current_resource_identifier: gitspaceData?.resource?.identifier || '',
      acl_filter: 'true'
    }
  })

  const { mutate: updateGitspace } = useUpdateGitspace({
    accountIdentifier: accountIdentifier,
    orgIdentifier: orgIdentifier,
    projectIdentifier: projectIdentifier,
    gitspace_identifier: gitspaceId
  })
  const isResourceInDenyList = (resource: any): boolean => {
    if (!resource || !resource.identifier) {
      return false
    }

    let infraProviderType = resource?.metadata?.infra_provider_type

    if (!infraProviderType) {
      if (resource?.identifier?.includes('aws')) {
        infraProviderType = 'hybrid_vm_aws'
      } else if (resource?.identifier?.includes('gcp')) {
        infraProviderType = 'hybrid_vm_gcp'
      } else {
        infraProviderType = 'harness_gcp'
      }
    }

    const accessList = gitspaceSettings?.settings?.infra_provider?.[infraProviderType]?.access_list

    if (!accessList || accessList.mode !== 'deny' || !Array.isArray(accessList.list)) {
      return false
    }

    return accessList.list.includes(resource.identifier)
  }
  const filteredIdeOptions = useFilteredIdeOptions(ideOptions, gitspaceSettings, getString)

  const defaultIdeType = useMemo(() => {
    if (gitspaceData.ide && filteredIdeOptions.some(option => option.value === gitspaceData.ide)) {
      return gitspaceData.ide
    }

    return filteredIdeOptions.length > 0 ? filteredIdeOptions[0].value : undefined
  }, [filteredIdeOptions, gitspaceData.ide])

  const transformedInitialValues = useMemo(() => {
    return {
      ...gitspaceData,
      resource_identifier: gitspaceData.resource?.identifier,
      resource_space_ref: gitspaceData.resource?.config_identifier,
      ssh_token_identifier: gitspaceData.ssh_token_identifier || ''
    }
  }, [gitspaceData])

  const machineTypes = useMemo(() => {
    if (!infraResources || !Array.isArray(infraResources)) return []

    const availableMachines = infraResources.filter(resource => !resource.is_deleted)

    if (
      gitspaceData?.resource?.identifier &&
      !availableMachines.some(machine => machine.identifier === gitspaceData.resource?.identifier) &&
      !isResourceInDenyList(gitspaceData?.resource)
    ) {
      return [
        ...availableMachines,
        {
          identifier: gitspaceData.resource.identifier,
          name: gitspaceData.resource.name || '',
          region: gitspaceData.resource.region,
          config_identifier: gitspaceData.resource.config_identifier,
          cpu: gitspaceData.resource.cpu || '',
          memory: gitspaceData.resource.memory || '',
          disk: gitspaceData.resource.disk || '',
          metadata: {
            persistent_disk_type: gitspaceData.resource.persistent_disk_type || ''
          }
        } as TypesInfraProviderResource
      ]
    }

    return availableMachines
  }, [infraResources, gitspaceData?.resource])

  const defaultMachineType = useMemo(() => {
    if (!machineTypes.length) return undefined

    const currentMachine = machineTypes.find(m => m.identifier === gitspaceData?.resource?.identifier)

    return currentMachine || machineTypes[0]
  }, [machineTypes, gitspaceData?.resource?.identifier])

  const isIdeInDenyList = (ide: EnumIDEType): boolean => {
    if (!gitspaceSettings?.settings?.gitspace_config?.ide?.access_list) {
      return false
    }

    const { mode, list } = gitspaceSettings.settings.gitspace_config.ide.access_list

    if (mode === 'deny' && list && Array.isArray(list)) {
      return list.includes(ide)
    }

    return false
  }

  const validateSubmission = (values: typeof gitspaceData) => {
    if (isIdeInDenyList(values.ide)) {
      if (filteredIdeOptions.length > 0) {
        throw new Error(getString('cde.update.errorChosenIDE'))
      } else {
        throw new Error(getString('cde.update.errorAllIDE'))
      }
    }
    if (machineTypes.length === 0) {
      throw new Error(getString('cde.update.errorAllResources'))
    }
  }

  const handleSubmit = async (values: typeof gitspaceData) => {
    setIsLoading(true)
    try {
      validateSubmission(values)
      const ideObject = getIDEOption(values.ide, getString)

      const payload = {
        name: values.name,
        ide: values.ide,
        resource_identifier: values.resource?.identifier,
        // resource_space_ref: values.resource?.config_identifier,
        ssh_token_identifier: ideObject?.allowSSH ? values.ssh_token_identifier || '' : ''
      }

      const updatedGitspace = await updateGitspace(payload)
      showSuccess(getString('cde.update.gitspaceUpdateSuccess'))
      onClose()
      if (onGitspaceUpdated) {
        onGitspaceUpdated()
      }

      const newGitspaceId = updatedGitspace?.identifier || gitspaceId

      const spacePath = isFromUsageDashboard && gitspacePath ? gitspacePath : space

      history.push(
        `${routes.toCDEGitspaceDetail({
          space: spacePath,
          gitspaceId: newGitspaceId
        })}`
      )
    } catch (error) {
      showError(getString('cde.update.gitspaceUpdateFailed'))
      showError(getErrorMessage(error))
    } finally {
      setIsLoading(false)
    }
  }
  return (
    <ModalDialog isOpen={isOpen} onClose={onClose} title={getString('cde.update.editGitspace')} width={920}>
      <Formik
        formName="editGitspaces"
        initialValues={transformedInitialValues}
        onSubmit={handleSubmit}
        enableReinitialize>
        {formik => {
          const selectedIDE = formik.values.ide ? getIDEOption(formik.values.ide, getString) : null

          return (
            <FormikForm>
              <Container
                className={css.formOuterContainer}
                onClick={e => {
                  e.stopPropagation()
                }}>
                <CDEIDESelect
                  onChange={formik.setFieldValue}
                  selectedIde={formik.values.ide || defaultIdeType}
                  filteredIdeOptions={filteredIdeOptions}
                  isEditMode={true}
                />
                {selectedIDE?.allowSSH ? (
                  <CDESSHSelect
                    isEditMode={true}
                    isFromUsageDashboard={isFromUsageDashboard}
                    gitspacePath={gitspacePath}
                  />
                ) : (
                  <></>
                )}

                <SelectEditMachine
                  options={machineTypes}
                  defaultValue={defaultMachineType || machineTypes[0] || {}}
                  isDisabled={isLoading || machineTypes.length === 0}
                  isEditMode={true}
                  loading={loadingResources}
                />

                <Layout.Horizontal className={css.buttonContainer}>
                  <Button
                    text={getString('cde.update.updateGitspace')}
                    variation={ButtonVariation.PRIMARY}
                    type="submit"
                    loading={isLoading}
                    className={css.button}
                  />
                  <Button
                    text={getString('cancel')}
                    variation={ButtonVariation.TERTIARY}
                    onClick={e => {
                      e.stopPropagation()
                      onClose()
                    }}
                    disabled={isLoading}
                    className={css.button}
                  />
                </Layout.Horizontal>

                <Container className={css.infoNoteContainer}>
                  <Layout.Horizontal spacing="small" className={css.infoNote}>
                    <img src={SolidInfoIcon} alt="Info" className={css.infoIcon} />
                    <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                      {getString('cde.update.gitspaceUpdateNote')}
                    </Text>
                  </Layout.Horizontal>
                </Container>
              </Container>
            </FormikForm>
          )
        }}
      </Formik>
    </ModalDialog>
  )
}
