import React from 'react'
import {
  Button,
  ButtonVariation,
  Formik,
  FormikForm,
  FormInput,
  ModalDialog,
  SelectOption,
  Container,
  Text,
  TableV2,
  Layout
} from '@harnessio/uicore'
import * as Yup from 'yup'
import cidrRegex from 'cidr-regex'
import { useFormikContext } from 'formik'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import type { Column } from 'react-table'
import type { ZoneConfig, regionProp } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import { AWS_AMI_ID_PATTERN } from 'cde-gitness/utils/InfraValidations.utils'
import CustomSelectDropdown from 'cde-gitness/components/CustomSelectDropdown/CustomSelectDropdown'
import SolidInfoIcon from 'cde-gitness/assests/solidInfo.svg?url'
import { InfraDetails } from './InfraDetails.constants'
import css from './NewRegionModal.module.scss'

interface NewRegionModalProps {
  isOpen: boolean
  setIsOpen: (value: boolean) => void
  onSubmit: (value: NewRegionModalForm) => void
  initialValues?: regionProp | null
  isEditMode?: boolean
  existingRegions?: string[]
}

interface NewRegionModalForm {
  location: string
  gatewayAmiId: string
  defaultSubnet: string
  proxySubnet: string
  domain: string
  zones?: ZoneConfig[]
  identifier: number
}

const validationSchema = () =>
  Yup.object().shape({
    location: Yup.string().required('Location is required'),
    gatewayAmiId: Yup.string()
      .required('Gateway AMI ID is required')
      .matches(AWS_AMI_ID_PATTERN, 'Invalid AMI ID format.'),
    domain: Yup.string().required('Domain is required'),
    zones: Yup.array()
      .of(
        Yup.object().shape({
          zone: Yup.string().required('Zone is required'),
          privateSubnet: Yup.string()
            .matches(cidrRegex({ exact: true }), 'Invalid CIDR format')
            .required('Private Subnet is required'),
          publicSubnet: Yup.string()
            .matches(cidrRegex({ exact: true }), 'Invalid CIDR format')
            .required('Public Subnet is required')
        })
      )
      .min(2, 'At least 2 zones are required')
  })

const NewRegionModal = ({
  isOpen,
  setIsOpen,
  onSubmit,
  initialValues,
  isEditMode = false,
  existingRegions = []
}: NewRegionModalProps) => {
  const { getString } = useStrings()

  const { values } = useFormikContext<{ domain: string }>()

  const regionOptions = Object.keys(InfraDetails.regions)
    .filter(region => {
      return isEditMode
        ? initialValues?.location === region || !existingRegions.includes(region)
        : !existingRegions.includes(region)
    })
    .map(item => {
      return {
        label: item,
        value: item
      }
    })

  const getDefaultZones = () => [
    {
      zone: '',
      privateSubnet: '',
      publicSubnet: '',
      id: Date.now()
    },
    {
      zone: '',
      privateSubnet: '',
      publicSubnet: '',
      id: Date.now() + 1
    }
  ]

  const getInitialValues = (): NewRegionModalForm => {
    if (initialValues) {
      const domainPrefix = initialValues.domain ? initialValues.domain.replace(`.${values?.domain}`, '') : ''

      return {
        location: initialValues.location || '',
        gatewayAmiId: initialValues.gatewayAmiId || '',
        defaultSubnet: initialValues.defaultSubnet || '',
        proxySubnet: initialValues.proxySubnet || '',
        domain: domainPrefix,
        zones: initialValues.zones && initialValues.zones.length >= 2 ? initialValues.zones : getDefaultZones(),
        identifier: initialValues.identifier || 0
      }
    }

    return {
      location: '',
      gatewayAmiId: '',
      defaultSubnet: '',
      proxySubnet: '',
      domain: '',
      zones: getDefaultZones(),
      identifier: 0
    }
  }

  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
      width={950}
      height={720}
      title={isEditMode ? 'Edit Region' : getString('cde.Aws.configureNewRegion')}>
      <Formik<NewRegionModalForm>
        validationSchema={validationSchema}
        onSubmit={formValues => {
          const fullDomain = formValues.domain ? `${formValues.domain}.${values.domain}` : values.domain
          onSubmit({
            ...formValues,
            domain: fullDomain
          })
        }}
        formName={''}
        initialValues={getInitialValues()}>
        {formikProps => {
          return (
            <FormikForm>
              <CustomSelectDropdown
                value={regionOptions.find(item => item.label === formikProps?.values?.location)}
                onChange={(data: SelectOption) => {
                  formikProps.setFieldValue('location', data?.value as string)
                }}
                label={getString('cde.Aws.selectAwsRegion')}
                options={regionOptions}
                error={formikProps.errors.location}
                disabled={isEditMode}
              />

              <div className="form-group">
                <Text className="form-group--label" font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                  {getString('cde.Aws.gatewayAmiId')}
                </Text>
                <FormInput.Text name="gatewayAmiId" placeholder="e.g. ami-12345678" />
              </div>
              <div className={`form-group ${css.marginTop20}`}>
                <Text className="form-group--label" font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                  {getString('cde.configureInfra.subdomain')}
                </Text>
                <div className={css.inputContainer}>
                  <div className={css.inputWrapper}>
                    <FormInput.Text name="domain" placeholder="us-west" disabled={isEditMode} />
                    <span className={css.domainSuffix}>.{values?.domain}</span>
                  </div>
                </div>
              </div>

              <div className={`form-group ${css.marginTop20}`}>
                <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500} className={css.zoneConfigTitle}>
                  {getString('cde.configureInfra.configureZones')}
                </Text>

                <ZonesTable formikProps={formikProps} isEditMode={isEditMode} />
              </div>

              <Container className={css.buttonContainer}>
                <Button variation={ButtonVariation.PRIMARY} type="submit" className={css.actionButton}>
                  {isEditMode ? getString('save') : getString('cde.gitspaceInfraHome.addnewRegion')}
                </Button>
                <Button
                  variation={ButtonVariation.TERTIARY}
                  text={getString('cancel')}
                  onClick={() => setIsOpen(false)}
                  className={css.cancelButton}
                />
              </Container>

              {!isEditMode && (
                <Container className={css.infoNoteContainer}>
                  <Layout.Horizontal spacing="small" className={css.infoNote}>
                    <img src={SolidInfoIcon} alt="Info" className={css.infoIcon} />
                    <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                      {getString('cde.Aws.regionConfigNote')}
                    </Text>
                  </Layout.Horizontal>
                </Container>
              )}
            </FormikForm>
          )
        }}
      </Formik>
    </ModalDialog>
  )
}

interface ZonesTableProps {
  formikProps: any
  isEditMode?: boolean
}

const ZonesTable = ({ formikProps, isEditMode = false }: ZonesTableProps) => {
  const zones = formikProps.values.zones || []

  // Create a stable reference to setFieldValue to prevent re-renders
  const setFieldValueRef = React.useRef(formikProps.setFieldValue)
  React.useEffect(() => {
    setFieldValueRef.current = formikProps.setFieldValue
  })

  const handleAddZone = React.useCallback(() => {
    if (isEditMode) {
      return
    }
    const newZone: ZoneConfig = {
      zone: '',
      privateSubnet: '',
      publicSubnet: '',
      id: Date.now()
    }
    setFieldValueRef.current('zones', [...zones, newZone])
  }, [zones, isEditMode])

  const handleDeleteZone = React.useCallback(
    (index: number) => {
      // Prevent deletion if only 2 zones remain (minimum requirement)
      if (isEditMode || zones.length <= 2) {
        return
      }
      const updatedZones = zones.filter((_: any, i: number) => i !== index)
      setFieldValueRef.current('zones', updatedZones)
    },
    [zones, isEditMode]
  )

  const ZoneCell = React.useCallback(
    ({ row }: { row: any }) => {
      const index = row.index
      const selectedRegion = formikProps.values.location

      const availableZones =
        selectedRegion && (InfraDetails.regions as Record<string, string[]>)[selectedRegion]
          ? (InfraDetails.regions as Record<string, string[]>)[selectedRegion]
          : []

      const zoneOptions = availableZones.map(zone => ({
        label: zone,
        value: zone
      }))

      return (
        <FormInput.Select
          name={`zones[${index}].zone`}
          placeholder={selectedRegion ? 'Select zone' : 'Select region first'}
          items={zoneOptions}
          disabled={isEditMode || !selectedRegion || zoneOptions.length === 0}
        />
      )
    },
    [formikProps.values.location, isEditMode]
  )

  const PrivateSubnetCell = React.useCallback(
    ({ row }: { row: any }) => {
      const index = row.index
      return <FormInput.Text name={`zones[${index}].privateSubnet`} placeholder="10.0.1.0/24" disabled={isEditMode} />
    },
    [isEditMode]
  )

  const PublicSubnetCell = React.useCallback(
    ({ row }: { row: any }) => {
      const index = row.index
      return <FormInput.Text name={`zones[${index}].publicSubnet`} placeholder="10.0.1.0/24" disabled={isEditMode} />
    },
    [isEditMode]
  )

  const ActionCell = React.useCallback(
    ({ row }: { row: any }) => {
      const index = row.index
      return (
        <Container className={css.flexCenter}>
          {zones.length > 2 && !isEditMode && (
            <Icon name="code-delete" size={24} className={css.cursorPointer} onClick={() => handleDeleteZone(index)} />
          )}
        </Container>
      )
    },
    [zones.length, handleDeleteZone, isEditMode]
  )

  // Stabilize column structure using useMemo to prevent re-creation on each render
  const zoneColumns = React.useMemo(
    (): Column<ZoneConfig>[] =>
      [
        {
          Header: 'Zone',
          accessor: 'zone',
          Cell: ZoneCell,
          width: '30%'
        },
        {
          Header: 'Private Subnet CIDR Block',
          accessor: 'privateSubnet',
          Cell: PrivateSubnetCell,
          width: '30%'
        },
        {
          Header: 'Public Subnet CIDR Block',
          accessor: 'publicSubnet',
          Cell: PublicSubnetCell,
          width: '30%'
        },
        {
          Header: '',
          accessor: 'actions',
          Cell: ActionCell,
          width: '10%'
        }
      ] as Column<ZoneConfig>[],
    [ZoneCell, PrivateSubnetCell, PublicSubnetCell, ActionCell]
  )

  const tableData = React.useMemo(() => zones, [zones])
  const { getString } = useStrings()

  return (
    <Container>
      <Container className={css.zonesContainer}>
        <div className={css.zonesTable}>
          <TableV2<ZoneConfig> columns={zoneColumns} data={tableData} className={css.zonesTable} minimal />
          {!isEditMode && (
            <div className={css.addZoneButton}>
              <Text
                icon="plus"
                iconProps={{ size: 10, color: Color.PRIMARY_7 }}
                color={Color.PRIMARY_7}
                onClick={handleAddZone}
                className={css.actionText}>
                {getString('cde.configureInfra.newZone')}
              </Text>
            </div>
          )}
        </div>
      </Container>
    </Container>
  )
}

export default NewRegionModal
