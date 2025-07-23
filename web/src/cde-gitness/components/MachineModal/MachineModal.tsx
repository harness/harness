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
import { cloneDeep, get } from 'lodash-es'
import { TypesInfraProviderResource, useCreateInfraProviderResource } from 'services/cde'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { getStringDropdownOptions, HYBRID_VM_GCP, regionType } from 'cde-gitness/constants'
import { validateMachineForm } from 'cde-gitness/utils/InfraValidations.utils'
import { useStrings } from 'framework/strings'
import { getZoneByRegion, machineTypes, persistentDiskTypes } from 'cde-gitness/utils/dropdownData.utils'
import CustomSelectDropdown from '../CustomSelectDropdown/CustomSelectDropdown'
import CustomInput from '../CustomInput/CustomInput'
import css from './MachineModal.module.scss'

interface MachineModalProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  infraproviderIdentifier: string
  regionIdentifier: string
  setRegionData: (val: regionType[]) => void
  regionData: regionType[]
}

interface MachineModalForm {
  name: string
  disk_type: string
  boot_size: string
  machine_type: string
  identifier: string
  disk_size: string
  boot_type: string
  zone: string
  image_name: string
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
  const zoneOptions: string[] = getZoneByRegion(regionIdentifier)
  const { mutate, loading } = useCreateInfraProviderResource({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: infraproviderIdentifier
  })

  const onSubmitHandler = async (values: MachineModalForm) => {
    try {
      const { name, disk_type, boot_size, machine_type, identifier, disk_size, boot_type, zone, image_name } = values
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
            zone,
            vm_image_name: image_name
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
      <Formik<MachineModalForm>
        formName="edit-layout-name"
        onSubmit={values => {
          onSubmitHandler(values)
        }}
        initialValues={{} as MachineModalForm}
        validationSchema={validateMachineForm(getString)}>
        {formik => {
          const { zone, disk_type, machine_type, disk_size, boot_size, boot_type } = formik?.values
          return (
            <FormikForm>
              <Layout.Vertical spacing="normal" className={css.formContainer}>
                <FormInput.InputWithIdentifier
                  inputLabel={getString('cde.configureInfra.name')}
                  inputName="name"
                  isIdentifierEditable={true}
                />
                <CustomSelectDropdown
                  options={zoneOptions?.map(options => getStringDropdownOptions(options))}
                  value={{ value: zone, label: zone }}
                  label={getString('cde.gitspaceInfraHome.zone')}
                  onChange={(value: { label: string; value: string }) => formik.setFieldValue('zone', value?.value)}
                  error={formik?.submitCount ? get(formik?.errors, 'zone') : ''}
                  allowCustom
                />
                <FormInput.Text
                  name="image_name"
                  label={getString('cde.gitspaceInfraHome.machineImageName')}
                  placeholder={getString('cde.gitspaceInfraHome.machineImageNamePlaceholder')}
                />
                <CustomSelectDropdown
                  options={persistentDiskTypes?.map((options: string) => getStringDropdownOptions(options))}
                  value={{ value: disk_type, label: disk_type }}
                  label={getString('cde.gitspaceInfraHome.diskType')}
                  onChange={(value: { label: string; value: string }) =>
                    formik.setFieldValue('disk_type', value?.value)
                  }
                  error={formik?.submitCount ? get(formik?.errors, 'disk_type') : ''}
                  allowCustom
                />
                <CustomInput
                  label={getString('cde.gitspaceInfraHome.diskSize')}
                  name="disk_size"
                  placeholder="e.g 100"
                  type="number"
                  value={disk_size}
                  onChange={(form: { value: string }) => formik.setFieldValue('disk_size', form.value)}
                  error={formik?.submitCount ? get(formik?.errors, 'disk_size') : ''}
                />
                <CustomSelectDropdown
                  options={persistentDiskTypes?.map((options: string) => getStringDropdownOptions(options))}
                  value={{ value: boot_type, label: boot_type }}
                  label={getString('cde.gitspaceInfraHome.bootType')}
                  onChange={(value: { label: string; value: string }) =>
                    formik.setFieldValue('boot_type', value?.value)
                  }
                  error={formik?.submitCount ? get(formik?.errors, 'boot_type') : ''}
                  allowCustom
                />
                <CustomInput
                  label={getString('cde.gitspaceInfraHome.bootSize')}
                  name="boot_size"
                  placeholder="e.g 100"
                  type="number"
                  value={boot_size}
                  onChange={(form: { value: string }) => formik.setFieldValue('boot_size', form.value)}
                  error={formik?.submitCount ? get(formik?.errors, 'boot_size') : ''}
                />
                <CustomSelectDropdown
                  options={machineTypes?.map((options: string) => getStringDropdownOptions(options))}
                  value={{ value: machine_type, label: machine_type }}
                  label={getString('cde.gitspaceInfraHome.machineType')}
                  onChange={(value: { label: string; value: string }) =>
                    formik.setFieldValue('machine_type', value?.value)
                  }
                  allowCustom
                  error={formik?.submitCount ? get(formik?.errors, 'machine_type') : ''}
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
