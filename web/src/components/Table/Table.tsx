import React, { useMemo, useState } from 'react'
import moment from 'moment'
import {
  Text,
  Layout,
  Color,
  TableV2,
  Button,
  ButtonVariation,
  useConfirmationDialog,
  useToaster
} from '@harness/uicore'
import type { CellProps, Renderer, Column } from 'react-table'
import { Menu, Position, Intent, Popover } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import type { Pipeline } from 'services/pm'

import styles from './Table.module.scss'

interface TableProps {
  data: Pipeline[] | null
  refetch: () => Promise<void>
  onDelete: (value: string) => Promise<void>
  onSettingsClick: (slug: string) => void
  onRowClick: (slug: string) => void
}

type CustomColumn<T extends Record<string, any>> = Column<T> & {
  refetch?: () => Promise<void>
}

const Table: React.FC<TableProps> = ({ data, refetch, onRowClick, onDelete, onSettingsClick }) => {
  const RenderColumn: Renderer<CellProps<Pipeline>> = ({
    cell: {
      column: { Header },
      row: { values }
    }
  }) => {
    let text
    switch (Header) {
      case 'ID':
        text = values.id
        break
      case 'Name':
        text = values.name
        break
      case 'Description':
        text = values.desc
        break
      case 'Slug':
        text = values.slug
        break
      case 'Created':
        text = moment(values.created).format('MM/DD/YYYY hh:mm:ss a')
        break
    }
    return (
      <Layout.Horizontal
        onClick={() => onRowClick(values.slug)}
        spacing="small"
        flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
        style={{ cursor: 'pointer' }}>
        <Layout.Vertical spacing="xsmall" padding={{ left: 'small' }} className={styles.verticalCenter}>
          <Layout.Horizontal spacing="small">
            <Text color={Color.BLACK} lineClamp={1}>
              {text}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
      </Layout.Horizontal>
    )
  }

  const RenderColumnMenu: Renderer<CellProps<Pipeline>> = ({ row: { values } }) => {
    const { showSuccess, showError } = useToaster()
    const { getString } = useStrings()
    const [menuOpen, setMenuOpen] = useState(false)
    const { openDialog } = useConfirmationDialog({
      titleText: getString('common.delete'),
      contentText: <Text color={Color.GREY_800}>Are you sure you want to delete this?</Text>,
      confirmButtonText: getString('common.delete'),
      cancelButtonText: getString('common.cancel'),
      intent: Intent.DANGER,
      buttonIntent: Intent.DANGER,
      onCloseDialog: async (isConfirmed: boolean) => {
        if (isConfirmed) {
          try {
            await onDelete(values.slug)
            showSuccess(getString('common.itemDeleted'))
            refetch()
          } catch (err) {
            showError(`Error: ${err}`)
            console.error({ err })
          }
        }
      }
    })

    return (
      <Layout.Horizontal className={styles.layout}>
        <Popover
          isOpen={menuOpen}
          onInteraction={nextOpenState => setMenuOpen(nextOpenState)}
          position={Position.BOTTOM_RIGHT}
          content={
            <Menu style={{ minWidth: 'unset' }}>
              <Menu.Item icon="trash" text={getString('common.delete')} onClick={openDialog} />
              <Menu.Item icon="settings" text={getString('settings')} onClick={() => onSettingsClick(values.slug)} />
            </Menu>
          }>
          <Button icon="Options" variation={ButtonVariation.ICON} />
        </Popover>
      </Layout.Horizontal>
    )
  }

  const columns: CustomColumn<Pipeline>[] = useMemo(
    () => [
      {
        Header: 'ID',
        id: 'id',
        accessor: row => row.id,
        width: '15%',
        Cell: RenderColumn
      },
      {
        Header: 'Name',
        id: 'name',
        accessor: row => row.name,
        width: '20%',
        Cell: RenderColumn
      },
      {
        Header: 'Description',
        id: 'desc',
        accessor: row => row.desc,
        width: '30%',
        Cell: RenderColumn,
        disableSortBy: true
      },
      {
        Header: 'Slug',
        id: 'slug',
        accessor: row => row.slug,
        width: '15%',
        Cell: RenderColumn,
        disableSortBy: true
      },
      {
        Header: 'Created',
        id: 'created',
        accessor: row => row.created,
        width: '15%',
        Cell: RenderColumn,
        disableSortBy: true
      },
      {
        Header: '',
        id: 'menu',
        accessor: row => row.slug,
        width: '5%',
        Cell: RenderColumnMenu,
        disableSortBy: true,
        refetch: refetch
      }
    ],
    [refetch]
  )
  return <TableV2<Pipeline> className={styles.table} columns={columns} name="basicTable" data={data || []} />
}

export default Table
