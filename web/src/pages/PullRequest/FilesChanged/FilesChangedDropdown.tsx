import React from 'react'
import { Container, ButtonVariation, Layout, Text, StringSubstitute, Button, Icon, Color } from '@harness/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { waitUntil } from 'utils/Utils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import type { DiffFileEntry } from 'utils/types'
import css from './FilesChangedDropdown.module.scss'
// import { TreeExample } from 'pages/Repository/RepositoryTree/TreeExample'

const STICKY_TOP_POSITION = 64

export const FilesChangedDropdown: React.FC<{ diffs: DiffFileEntry[] }> = ({ diffs }) => {
  const { getString } = useStrings()

  return (
    <Button
      variation={ButtonVariation.LINK}
      className={css.link}
      tooltip={
        <Container padding="small" className={css.filesMenu}>
          {/* <TreeExample /> */}
          <Menu>
            {diffs?.map((diff, index) => (
              <MenuItem
                key={index}
                className={css.menuItem}
                icon={<Icon name={CodeIcon.File} padding={{ right: 'xsmall' }} />}
                labelElement={
                  <Layout.Horizontal spacing="xsmall">
                    {!!diff.addedLines && (
                      <Text color={Color.GREEN_600} style={{ fontSize: '12px' }}>
                        +{diff.addedLines}
                      </Text>
                    )}
                    {!!diff.addedLines && !!diff.deletedLines && <PipeSeparator height={8} />}
                    {!!diff.deletedLines && (
                      <Text color={Color.RED_500} style={{ fontSize: '12px' }}>
                        -{diff.deletedLines}
                      </Text>
                    )}
                  </Layout.Horizontal>
                }
                text={
                  diff.isDeleted ? diff.oldName : diff.isRename ? `${diff.oldName} -> ${diff.newName}` : diff.newName
                }
                onClick={() => {
                  // When an item is clicked, do these:
                  //  1. Scroll into the diff container of the file.
                  //     The diff content might not be rendered yet since it's off-screen
                  //  2. Wait until its content is rendered
                  //  3. Adjust scroll position as when diff content is rendered, current
                  //     window scroll position might push diff content up, we need to push
                  //     it down again to make sure first line of content is visible and not
                  //     covered by sticky headers
                  const containerDOM = document.getElementById(diff.containerId)

                  if (containerDOM) {
                    containerDOM.scrollIntoView()

                    waitUntil(
                      () => !!containerDOM.querySelector('[data-rendered="true"]'),
                      () => {
                        containerDOM.scrollIntoView()

                        if (containerDOM.getBoundingClientRect().y - STICKY_TOP_POSITION < 1) {
                          if (STICKY_TOP_POSITION) {
                            window.scroll({ top: window.scrollY - STICKY_TOP_POSITION })
                          }
                        }
                      }
                    )
                  }
                }}
              />
            ))}
          </Menu>
        </Container>
      }
      tooltipProps={{ interactionKind: 'click', hasBackdrop: true, popoverClassName: css.popover }}>
      <StringSubstitute str={getString('pr.showLink')} vars={{ count: diffs?.length || 0 }} />
    </Button>
  )
}
