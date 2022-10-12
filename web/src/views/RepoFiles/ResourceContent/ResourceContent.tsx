import React, { useMemo, useRef, useState } from 'react'
import cx from 'classnames'
// import root from 'react-shadow'
import MonacoEditor from 'react-monaco-editor'
import type { editor as EDITOR } from 'monaco-editor/esm/vs/editor/editor.api'
import {
  Container,
  Color,
  Layout,
  Button,
  FlexExpander,
  TextInput,
  ButtonVariation,
  Tabs,
  TableV2 as Table,
  Text,
  Heading
} from '@harness/uicore'
import type { CellProps, Column } from 'react-table'
import MarkdownPreview from '@uiw/react-markdown-preview'
import { useStrings } from 'framework/strings'
// import markdownCSS from '!raw-loader!@uiw/react-markdown-preview/dist/markdown.css'
import markdown from './sampleREADME.md'
import css from './ResourceContent.module.scss'
import sampleCSs from '!raw-loader!./ResourceContent.module.scss'

// import('!raw-loader!./sampleREADME.md').then(foo => console.log('dynamic data:', foo.default))

// TODO: USE FROM SERVICE (DOES NOT EXIST YET)
interface Folder {
  name: string
  lastCommitMessage: string
  isFile?: boolean
  updated: string
}

const repodata: Folder[] = [
  {
    name: '.github',
    lastCommitMessage: 'feat: [PIE-5298]: cypress retry 0 in open mode for easy debugging',
    isFile: false,
    updated: '14 days ago'
  },
  {
    name: 'src',
    lastCommitMessage: 'feat: [CI-5092]: sort ascending parallel steps (#11374)',
    isFile: false,
    updated: '1 day ago'
  },
  {
    name: '.dockerignore',
    lastCommitMessage: 'feat: [PL-15242]: upload source maps for bugsnag during prod build',
    isFile: true,
    updated: '13 months ago'
  },
  {
    name: '.gitignore',
    lastCommitMessage: 'feat: [FFM-3171]: UI: Filter Feature Flags (#8971)',
    isFile: true,
    updated: '4 months ago'
  },
  {
    name: 'LICENSE.md',
    lastCommitMessage: 'fix: [ONP-257]: no license stamping output for deleted files (#6903)',
    isFile: true,
    updated: '14 days ago'
  },
  {
    name: '.github',
    lastCommitMessage: 'feat: [PIE-5298]: cypress retry 0 in open mode for easy debugging',
    isFile: true,
    updated: '14 days ago'
  }
]

export function ResourceContent(): JSX.Element {
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState('README.md')

  return (
    <Container padding="xlarge" className={cx(css.tabContent, css.resourceContent)} background={Color.WHITE}>
      <Container>
        <Layout.Horizontal>
          <TextInput
            placeholder="Search folder or file"
            leftIcon="slash"
            leftIconProps={{ name: 'slash', size: 16 }}
            style={{ width: 350 }}
            autoFocus
            onFocus={event => event.target.select()}
            value={searchTerm}
            onInput={event => {
              setSearchTerm(event.currentTarget.value)
            }}
          />
          <FlexExpander />
          <Button text="Clone" variation={ButtonVariation.PRIMARY} icon="main-clone" />
        </Layout.Horizontal>
      </Container>
      <Container className={css.tabs}>
        <Tabs
          id="repoContentTabs"
          defaultSelectedTabId={'content'}
          tabList={[
            {
              id: 'content',
              title: getString('content'),
              panel: <FolderListing />
            },
            {
              id: 'history',
              title: getString('history'),
              panel: <Container>tbd</Container>,
              disabled: false
            }
          ]}></Tabs>
      </Container>
    </Container>
  )
}

export function FolderListing(): JSX.Element {
  const { getString } = useStrings()
  const columns: Column<Folder>[] = useMemo(
    () => [
      {
        Header: getString('name'),
        accessor: (row: Folder) => row.name,
        width: '30%',
        Cell: ({ row }: CellProps<Folder>) => {
          return (
            <Text
              icon={row.original.isFile ? 'main-template-library' : 'main-folder'}
              lineClamp={1}
              iconProps={{ margin: { right: 'xsmall' } }}>
              {row.original.name}
            </Text>
          )
        }
      },
      {
        Header: getString('commits'),
        accessor: row => row.lastCommitMessage,
        width: '60%',
        Cell: ({ row }: CellProps<Folder>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1}>
              {row.original.lastCommitMessage}
            </Text>
          )
        }
      },
      {
        Header: getString('repos.lastChange'),
        accessor: row => row.updated,
        width: '10%',
        Cell: ({ row }: CellProps<Folder>) => {
          return (
            <Text color={Color.BLACK} lineClamp={1}>
              {row.original.updated}
            </Text>
          )
        },
        disableSortBy: true
      }
    ],
    [getString]
  )
  const [input, updateInput] = useState('{\n\t"foo":\t"bar"\n}')
  const inputContainerRef = useRef<HTMLDivElement>(null)
  const [inputEditor, setInputEditor] = useState<EDITOR.IStandaloneCodeEditor>()

  console.log({ sampleCSs })

  return (
    <Container>
      {/* <Table<Folder>
        className={css.table}
        columns={columns}
        data={repodata || []}
        onRowClick={_data => {
          // onPolicyClicked(data)
        }}
        getRowClassName={() => css.row}
      /> */}

      <Container className={css.fileContentContainer} background={Color.WHITE}>
        <Layout.Horizontal padding="small" className={css.fileContentHeading}>
          <Heading level={5}>README.md</Heading>
          <FlexExpander />
          <Button variation={ButtonVariation.ICON} icon="edit" />
        </Layout.Horizontal>
        <Container className={css.readmeContainer} style={{ display: 'none' }}>
          <MarkdownPreview
            source={markdown}
            // rehypeRewrite={(node, _index, parent) => {
            //   if (
            //     (node as unknown as Element).tagName === 'a' &&
            //     parent &&
            //     /^h(1|2|3|4|5|6)/.test((parent as unknown as Element).tagName)
            //   ) {
            //     parent.children = parent.children.slice(1)
            //   }
            // }}
          />
          <Container flex ref={inputContainerRef} style={{ maxHeight: '95%', height: '900px' }}>
            <MonacoEditor
              language="json"
              theme="vs-light"
              value={input}
              options={{ ...MonacoEditorJsonOptions, automaticLayout: true, readOnly: true }}
              onChange={updateInput}
              editorDidMount={setInputEditor}
            />
          </Container>
        </Container>
      </Container>
    </Container>
  )
}

export const MonacoEditorOptions = {
  ignoreTrimWhitespace: true,
  minimap: { enabled: false },
  codeLens: false,
  scrollBeyondLastLine: false,
  smartSelect: false,
  tabSize: 4,
  insertSpaces: true,
  overviewRulerBorder: false
}

export const MonacoEditorJsonOptions = {
  ...MonacoEditorOptions,
  tabSize: 2
}
