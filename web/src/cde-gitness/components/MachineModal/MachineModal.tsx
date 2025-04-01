import React from 'react'
import {
  Button,
  ButtonVariation,
  Formik,
  FormikForm,
  FormInput,
  Layout,
  ModalDialog,
  useToaster
} from '@harnessio/uicore'
import { cloneDeep } from 'lodash-es'
import { TypesInfraProviderResource, useCreateInfraProviderResource } from 'services/cde'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { HYBRID_VM_GCP, regionType } from 'cde-gitness/constants'
import { validateMachineForm } from 'cde-gitness/utils/InfraValidations.utils'
import { useStrings } from 'framework/strings'
import css from './MachineModal.module.scss'

interface MachineModalProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  infraproviderIdentifier: string
  regionIdentifier: string
  setRegionData: (val: regionType[]) => void
  regionData: regionType[]
}

function MachineModal({
  isOpen,
  setIsOpen,
  infraproviderIdentifier,
  regionIdentifier,
  setRegionData,
  regionData
}: MachineModalProps) {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const { showSuccess, showError } = useToaster()
  const { mutate, loading } = useCreateInfraProviderResource({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: infraproviderIdentifier
  })

  const onSubmitHandler = async (values: any) => {
    try {
      const { name, disk_type, boot_size, machine_type, identifier, disk_size, boot_type, zone } = values
      const payload: any = [
        {
          identifier,
          name,
          infra_provider_type: HYBRID_VM_GCP,
          region: regionIdentifier,
          disk: disk_type,
          metadata: {
            persistent_disk_type: disk_type,
            boot_disk_size: boot_size,
            persistent_disk_size: disk_size,
            boot_disk_type: boot_type,
            region_name: regionIdentifier,
            machine_type,
            zone
          }
        }
      ]
      const data: TypesInfraProviderResource[] = await mutate(payload)
      showSuccess(getString('cde.create.machineCreateSuccess'))
      const cloneData = cloneDeep(regionData)
      const updatedData: regionType[] = []
      cloneData?.forEach((region: regionType) => {
        if (region?.region_name === regionIdentifier && data?.length === 1) {
          region.machines.push(data?.[0])
        }
        updatedData.push(region)
      })
      setRegionData(updatedData)
      setIsOpen(false)
    } catch (err) {
      showError(getString('cde.create.machineCreateFailed'))
      showError(getErrorMessage(err))
    }
  }

  return (
    <ModalDialog
      className={css.machineModal}
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
      title={getString('cde.gitspaceInfraHome.createNewMachine')}
      width={700}>
      <Formik
        formName="edit-layout-name"
        onSubmit={(values: any) => {
          onSubmitHandler(values)
        }}
        initialValues={{}}
        validationSchema={validateMachineForm(getString)}>
        {() => {
          return (
            <FormikForm>
              <Layout.Vertical spacing="normal" className={css.formContainer}>
                <FormInput.InputWithIdentifier
                  inputLabel={getString('cde.configureInfra.name')}
                  inputName="name"
                  isIdentifierEditable={true}
                />
                <FormInput.Text label={getString('cde.gitspaceInfraHome.zone')} name="zone" />
                <FormInput.Text
                  label={getString('cde.gitspaceInfraHome.diskType')}
                  name="disk_type"
                  placeholder="e.g Balanced"
                />
                <FormInput.Text
                  label={getString('cde.gitspaceInfraHome.diskSize')}
                  name="disk_size"
                  placeholder="e.g 100"
                />
                <FormInput.Text
                  label={getString('cde.gitspaceInfraHome.bootType')}
                  name="boot_type"
                  placeholder="e.g standard"
                />
                <FormInput.Text
                  label={getString('cde.gitspaceInfraHome.bootSize')}
                  name="boot_size"
                  placeholder="e.g 100"
                />
                <FormInput.Text
                  label={getString('cde.gitspaceInfraHome.machineType')}
                  name="machine_type"
                  placeholder="e.g standard"
                />
              </Layout.Vertical>
              <Layout.Horizontal spacing="small" className={css.modalFooter}>
                <Button
                  variation={ButtonVariation.PRIMARY}
                  text={getString('cde.gitspaceInfraHome.create')}
                  type="submit"
                  loading={loading}
                />
                <Button
                  variation={ButtonVariation.TERTIARY}
                  text={getString('cde.configureInfra.cancel')}
                  onClick={() => setIsOpen(false)}
                />
              </Layout.Horizontal>
            </FormikForm>
          )
        }}
      </Formik>
    </ModalDialog>
  )
}

export default MachineModal
