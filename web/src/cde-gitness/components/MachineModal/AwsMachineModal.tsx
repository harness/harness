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
import { getStringDropdownOptions, HYBRID_VM_AWS, regionType } from 'cde-gitness/constants'
import { validateAwsMachineForm } from 'cde-gitness/utils/InfraValidations.utils'
import { InfraDetails } from 'cde-gitness/pages/InfraConfigure/AWS/InfraDetails/InfraDetails.constants'
import { useStrings } from 'framework/strings'
import CustomSelectDropdown from '../CustomSelectDropdown/CustomSelectDropdown'
import CustomInput from '../CustomInput/CustomInput'
import css from './MachineModal.module.scss'

interface AwsMachineModalProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  infraproviderIdentifier: string
  regionIdentifier: string
  setRegionData: (val: regionType[]) => void
  regionData: regionType[]
  refetch: () => void
}

interface AwsMachineModalForm {
  name: string
  disk_type: string
  boot_size: string
  machine_type: string
  disk_size: string
  boot_type: string
  zone: string
  vm_image_name: string
}

function AwsMachineModal({
  isOpen,
  setIsOpen,
  infraproviderIdentifier,
  regionIdentifier,
  setRegionData,
  regionData,
  refetch
}: AwsMachineModalProps) {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const { showSuccess, showError } = useToaster()

  const getAwsZonesByRegion = (region: string): string[] => {
    return (InfraDetails.regions as Record<string, string[]>)[region] || []
  }

  const getAwsInstanceTypes = (): string[] => {
    return InfraDetails.instance_types.map(instance => instance.name)
  }

  const getAwsVolumeTypes = (): string[] => {
    return InfraDetails.volume_types.map(volume => volume.name)
  }

  const zoneOptions: string[] = getAwsZonesByRegion(regionIdentifier)
  const instanceTypeOptions: string[] = getAwsInstanceTypes()
  const volumeTypeOptions: string[] = getAwsVolumeTypes()
  const { mutate, loading } = useCreateInfraProviderResource({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: infraproviderIdentifier
  })

  const onSubmitHandler = async (values: AwsMachineModalForm) => {
    try {
      const { name, disk_type, boot_size, machine_type, disk_size, boot_type, zone, vm_image_name } = values
      const payload: any = [
        {
          name,
          infra_provider_type: HYBRID_VM_AWS,
          region: regionIdentifier,
          disk: disk_size,
          metadata: {
            persistent_disk_type: disk_type,
            boot_disk_size: boot_size,
            persistent_disk_size: disk_size,
            boot_disk_type: boot_type,
            region_name: regionIdentifier,
            machine_type,
            zone,
            vm_image_name
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
      refetch?.()
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
      <Formik<AwsMachineModalForm>
        formName="edit-layout-name"
        onSubmit={values => {
          onSubmitHandler(values)
        }}
        initialValues={{} as AwsMachineModalForm}
        validationSchema={validateAwsMachineForm(getString)}>
        {formik => {
          const { zone, disk_type, machine_type, disk_size, boot_size, boot_type } = formik?.values
          return (
            <FormikForm>
              <Layout.Vertical spacing="normal" className={css.formContainer}>
                <FormInput.Text label={getString('cde.configureInfra.name')} name="name" />
                <CustomSelectDropdown
                  options={zoneOptions?.map(options => getStringDropdownOptions(options))}
                  value={{ value: zone, label: zone }}
                  label={getString('cde.Aws.availabilityZone')}
                  onChange={(value: { label: string; value: string }) => formik.setFieldValue('zone', value?.value)}
                  error={formik?.submitCount ? get(formik?.errors, 'zone') : ''}
                  allowCustom
                />
                <FormInput.Text
                  name="vm_image_name"
                  label={getString('cde.Aws.machineAmiId')}
                  placeholder={getString('cde.Aws.machineAmiIdPlaceholder')}
                />
                <CustomSelectDropdown
                  options={volumeTypeOptions?.map((options: string) => getStringDropdownOptions(options))}
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
                  type="text"
                  value={disk_size}
                  autoComplete="off"
                  onChange={(form: { value: string }) => {
                    if (form.value === '' || (/\d+/.test(form.value) && parseInt(form.value, 10) >= 0)) {
                      formik.setFieldValue('disk_size', form.value)
                    }
                  }}
                  error={formik?.submitCount ? get(formik?.errors, 'disk_size') : ''}
                />
                <CustomSelectDropdown
                  options={volumeTypeOptions?.map((options: string) => getStringDropdownOptions(options))}
                  value={{ value: boot_type, label: boot_type }}
                  label={getString('cde.gitspaceInfraHome.bootDiskType')}
                  onChange={(value: { label: string; value: string }) =>
                    formik.setFieldValue('boot_type', value?.value)
                  }
                  error={formik?.submitCount ? get(formik?.errors, 'boot_type') : ''}
                  allowCustom
                />
                <CustomInput
                  label={getString('cde.gitspaceInfraHome.bootDiskSize')}
                  name="boot_size"
                  placeholder="e.g 100"
                  type="text"
                  autoComplete="off"
                  value={boot_size}
                  onChange={(form: { value: string }) => {
                    if (form.value === '' || (/\d+/.test(form.value) && parseInt(form.value, 10) >= 0)) {
                      formik.setFieldValue('boot_size', form.value)
                    }
                  }}
                  error={formik?.submitCount ? get(formik?.errors, 'boot_size') : ''}
                />
                <CustomSelectDropdown
                  options={instanceTypeOptions?.map((options: string) => getStringDropdownOptions(options))}
                  value={{ value: machine_type, label: machine_type }}
                  label={getString('cde.Aws.instanceType')}
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

export default AwsMachineModal
