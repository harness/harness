import React, { useMemo } from 'react'
import { Card, Text, Layout, Button, ButtonVariation, Container, TableV2, FormInput } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext, FieldArray } from 'formik'
import type { Column } from 'react-table'
import { useStrings } from 'framework/strings'
import type { AdminSettingsFormValues } from 'cde-gitness/pages/AdminSettings/utils/adminSettingsUtils'
import css from './AllowedImagePaths.module.scss'

interface ConnectorRowData {
  imagePath: string
}

interface CellRenderProps {
  row: { index: number }
  remove: (index: number) => void
}

type CellRenderFunction = (props: CellRenderProps) => React.ReactNode

type ColumnType = Column<ConnectorRowData> & {
  Cell?: CellRenderFunction
}

const ImagePathCell = React.memo(({ row }: Pick<CellRenderProps, 'row'>) => {
  return (
    <Container flex={{ alignItems: 'flex-start', justifyContent: 'flex-start' }} className={css.imagePathCellContainer}>
      <FormInput.Text
        name={`gitspaceImages.access_list.list[${row.index}]`}
        placeholder="mcr.microsoft.com/devcontainers/java"
        className={css.imagePathCard}
      />
    </Container>
  )
})

ImagePathCell.displayName = 'ImagePathCell'

const DeleteCell = React.memo(({ row, remove }: CellRenderProps) => {
  return (
    <Container flex={{ alignItems: 'flex-start', justifyContent: 'flex-end' }} className={css.imagePathCellContainer}>
      <Button
        variation={ButtonVariation.ICON}
        icon="main-trash"
        iconProps={{ size: 24 }}
        onClick={() => remove(row.index)}
      />
    </Container>
  )
})

DeleteCell.displayName = 'DeleteCell'

export const AllowedImagePaths: React.FC = () => {
  const { values } = useFormikContext<AdminSettingsFormValues>()
  const { getString } = useStrings()

  const paths = values.gitspaceImages?.access_list?.list || []
  const showTable = paths.length > 0

  const baseColumns = useMemo<ColumnType[]>(
    () => [
      {
        id: 'imagePath',
        Header: '',
        accessor: 'imagePath',
        width: '100%',
        Cell: ImagePathCell
      },
      {
        id: 'delete',
        Header: '',
        width: '10%',
        Cell: DeleteCell
      }
    ],
    []
  )

  return (
    <Card className={css.mainContentCard}>
      <Layout.Vertical spacing="medium">
        <Text font={{ variation: FontVariation.H5 }}>
          {getString('cde.settings.images.allowedImagePathsAndRegistries')}
        </Text>
        <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
          {getString('cde.settings.images.allowedImagePathsDescription')}
        </Text>

        <FieldArray name="gitspaceImages.access_list.list">
          {({ push, remove }) => (
            <>
              {showTable && (
                <Container margin={{ top: 'medium' }}>
                  <TableV2<ConnectorRowData>
                    getRowClassName={() => css.defaultImageTableRow}
                    className={css.defaultImageTable}
                    columns={baseColumns.map(col => {
                      if (col.id === 'delete') {
                        return {
                          ...col,
                          Cell: ({ row }: Pick<CellRenderProps, 'row'>) => {
                            return React.createElement(DeleteCell, { row, remove })
                          }
                        }
                      }
                      return col
                    })}
                    data={paths.map(imagePath => ({ imagePath }))}
                    minimal
                    hideHeaders
                  />
                </Container>
              )}

              <Button
                variation={ButtonVariation.SECONDARY}
                width="220px"
                margin={{ top: 'medium' }}
                onClick={() => push('')}>
                {getString('cde.settings.images.newImagePath')}
              </Button>
            </>
          )}
        </FieldArray>
      </Layout.Vertical>
    </Card>
  )
}
