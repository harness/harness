import React, { useState, useMemo, useCallback } from 'react'
import { Card, Text, Layout, Button, ButtonVariation, Container, TableV2, FormInput } from '@harnessio/uicore'
import { Spinner } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext } from 'formik'
import { useStrings } from 'framework/strings'
import { ProvideDefaultImageModal } from 'components/ProvideDefaultImage/ProvideDefaultImage'
import type { AdminSettingsFormValues } from 'cde-gitness/pages/AdminSettings/utils/adminSettingsUtils'
import { useAppContext } from 'AppContext'
import type { TypesGitspaceSettingsResponse } from 'services/cde'
import { getConnectorIcon } from 'cde-gitness/pages/AdminSettings/utils/connectorUtils'
import css from './DefaultGitspaceImage.module.scss'

interface DefaultGitspaceImageProps {
  settings: TypesGitspaceSettingsResponse | null
}

interface ConnectorRowData {
  name: string
  identifier: string
  type: string
  id: string
  imagePath: string
}

interface TableRowProps {
  row: {
    original: ConnectorRowData
  }
}

export const DefaultGitspaceImage: React.FC<DefaultGitspaceImageProps> = ({ settings }: DefaultGitspaceImageProps) => {
  const { values, setFieldValue } = useFormikContext<AdminSettingsFormValues>()
  const { getString } = useStrings()
  const { accountInfo, hooks } = useAppContext()
  const { useGetConnector } = hooks

  const hadInitialImage = Boolean(
    settings?.settings?.gitspace_config?.devcontainer?.devcontainer_image?.image_name ||
      values.gitspaceImages?.image_name
  )

  const [showRow, setShowRow] = useState(hadInitialImage)
  const [isModalOpen, setIsModalOpen] = useState(false)

  const connectorRef = values.gitspaceImages?.image_connector_ref
  const connectorId = connectorRef?.includes('account.') ? connectorRef.split('account.')[1] : connectorRef

  const { loading: loadingConnector, data: connectorData } = useGetConnector({
    identifier: connectorId || '',
    queryParams: { accountIdentifier: accountInfo?.identifier },
    lazy: !connectorId
  })

  const DeleteCell = useCallback(
    () => (
      <Container flex={{ alignItems: 'flex-start', justifyContent: 'flex-end' }} className={css.imagePathCellContainer}>
        <Button
          variation={ButtonVariation.ICON}
          icon="main-trash"
          iconProps={{ size: 24 }}
          onClick={() => {
            setFieldValue('gitspaceImages', {
              ...values.gitspaceImages,
              image_name: undefined,
              image_connector_ref: undefined,
              default_image_added: false
            })
            setShowRow(false)
          }}
        />
      </Container>
    ),
    [setFieldValue, values.gitspaceImages]
  )

  const ImagePathCell = useCallback(
    () => (
      <Container
        flex={{ alignItems: 'flex-start', justifyContent: 'flex-start' }}
        className={css.imagePathCellContainer}>
        <FormInput.Text name="gitspaceImages.image_name" className={css.imagePathCard} />
      </Container>
    ),
    []
  )

  const privateColumns = useMemo(
    () => [
      {
        Header: getString('cde.settings.images.artifactRegistryConnector'),
        accessor: 'name' as const,
        width: '50%',
        Cell: ({ row }: TableRowProps) => {
          const { name, identifier, type } = row.original
          return (
            <Layout.Horizontal
              spacing="medium"
              flex={{ alignItems: 'flex-start', justifyContent: 'flex-start' }}
              className={css.imagePathCellContainer}>
              <img src={getConnectorIcon(type)} height={28} width={28} />
              <Layout.Vertical spacing="xsmall">
                <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.GREY_800}>
                  {name}
                </Text>
                <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
                  ID: {identifier}
                </Text>
              </Layout.Vertical>
            </Layout.Horizontal>
          )
        }
      },
      {
        Header: getString('cde.settings.images.defaultPathToPrivateGitspaceImagePath'),
        accessor: 'imagePath' as const,
        width: '50%',
        Cell: ImagePathCell
      },
      {
        Header: '',
        accessor: 'id' as const,
        width: '10%',
        Cell: DeleteCell
      }
    ],
    [getString, DeleteCell, ImagePathCell]
  )

  const publicColumns = useMemo(
    () => [
      {
        Header: getString('cde.settings.images.defaultPathToPublicGitspaceImagePath'),
        accessor: 'imagePath' as const,
        width: '100%',
        Cell: ImagePathCell
      },
      {
        Header: '',
        accessor: 'id' as const,
        width: '10%',
        Cell: DeleteCell
      }
    ],
    [getString, DeleteCell, ImagePathCell]
  )

  return (
    <>
      <Card className={css.mainContentCard}>
        <Layout.Vertical spacing="medium">
          <Text font={{ variation: FontVariation.H5 }}>{getString('cde.settings.images.defaultGitspaceImage')}</Text>
          <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
            {getString('cde.settings.images.defaultImageDescription')}
          </Text>

          {showRow ? (
            <>
              {loadingConnector ? (
                <Spinner size={Spinner.SIZE_STANDARD} />
              ) : (
                <Container className={css.tableContainer} margin={{ top: 'medium', bottom: 'medium' }}>
                  <TableV2<ConnectorRowData>
                    getRowClassName={() => css.defaultImageTableRow}
                    className={css.defaultImageTable}
                    columns={values.gitspaceImages?.image_connector_ref ? privateColumns : publicColumns}
                    data={
                      values.gitspaceImages?.image_connector_ref
                        ? connectorData?.data?.connector
                          ? [connectorData.data.connector]
                          : []
                        : [{}]
                    }
                    minimal
                  />
                </Container>
              )}
            </>
          ) : null}

          {showRow ? null : (
            <Button
              variation={ButtonVariation.SECONDARY}
              width={'220px'}
              margin={{ top: 'medium' }}
              onClick={() => setIsModalOpen(true)}>
              {getString('cde.settings.images.provideDefaultImage')}
            </Button>
          )}
        </Layout.Vertical>
      </Card>
      <ProvideDefaultImageModal
        isOpen={isModalOpen}
        onClose={formValues => {
          setIsModalOpen(false)
          if (formValues?.imagePath) {
            setShowRow(true)
          }
        }}
      />
    </>
  )
}
