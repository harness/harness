import React, { useMemo } from 'react'
import { useGet } from 'restful-react'
import {
  ButtonSize,
  ButtonVariation,
  Color,
  Container,
  FlexExpander,
  Heading,
  Layout,
  Tabs,
  Utils
} from '@harness/uicore'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { SourceCodeViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { OpenapiContentInfo, RepoFileContent, TypesCommit } from 'services/code'
import {
  decodeGitContent,
  findMarkdownInfo,
  GitCommitAction,
  GitInfoProps,
  isRefATag,
  makeDiffRefs
} from 'utils/GitUtils'
import { filenameToLanguage, permissionProps, LIST_FETCHING_LIMIT, RenameDetails, FileSection } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { LatestCommitForFile } from 'components/LatestCommit/LatestCommit'
import { useCommitModal } from 'components/CommitModalButton/CommitModalButton'
import { useStrings } from 'framework/strings'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { PlainButton } from 'components/PlainButton/PlainButton'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { usePageIndex } from 'hooks/usePageIndex'
import { Readme } from '../FolderContent/Readme'
import { GitBlame } from './GitBlame'
import RenameContentHistory from './RenameContentHistory'
import css from './FileContent.module.scss'

export function FileContent({
  repoMetadata,
  gitRef,
  resourcePath,
  resourceContent,
  commitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent' | 'commitRef'>) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const history = useHistory()
  const [activeTab, setActiveTab] = React.useState<string>(FileSection.CONTENT)

  const content = useMemo(
    () => decodeGitContent((resourceContent?.content as RepoFileContent)?.data),
    [resourceContent?.content]
  )
  const markdownInfo = useMemo(() => findMarkdownInfo(resourceContent), [resourceContent])
  const [openDeleteFileModal] = useCommitModal({
    repoMetadata,
    gitRef,
    resourcePath,
    commitAction: GitCommitAction.DELETE,
    commitTitlePlaceHolder: getString('deleteFile').replace('__path__', resourcePath),
    onSuccess: (_commitInfo, newBranch) => {
      if (newBranch) {
        history.replace(
          routes.toCODECompare({
            repoPath: repoMetadata.path as string,
            diffRefs: makeDiffRefs(repoMetadata?.default_branch as string, newBranch)
          })
        )
      } else {
        history.push(
          routes.toCODERepository({
            repoPath: repoMetadata.path as string,
            gitRef
          })
        )
      }
    }
  })

  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const space = useGetSpaceParam()

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const permsFinal = useMemo(() => {
    const perms = permissionProps(permPushResult, standalone)
    if (isRefATag(gitRef) && perms) {
      return { tooltip: perms.tooltip, disabled: true }
    }

    if (isRefATag(gitRef)) {
      return { tooltip: getString('editNotAllowed'), disabled: true }
    } else if (perms?.disabled) {
      return { disabled: perms.disabled, tooltip: perms.tooltip }
    }
    return { disabled: isRefATag(gitRef) || false, tooltip: undefined }
  }, [permPushResult, gitRef]) // eslint-disable-line react-hooks/exhaustive-deps
  
  const [page, setPage] = usePageIndex()
  const { data: commits, response } = useGet<{ commits: TypesCommit[]; rename_details: RenameDetails[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commitsV2`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      git_ref: commitRef || repoMetadata?.default_branch,
      path: resourcePath
    },
    lazy: !repoMetadata
  })

  return (
    <Container className={css.tabsContainer}>
      <Tabs
        id="fileTabs"
        selectedTabId={activeTab}
        defaultSelectedTabId={FileSection.CONTENT}
        large={false}
        onChange={(id: string) => setActiveTab(id)}
        tabList={[
          {
            id: FileSection.CONTENT,
            title: getString('content'),
            panel: (
              <Container className={css.fileContent}>
                <Layout.Vertical spacing="small" style={{ maxWidth: '100%' }}>
                  <LatestCommitForFile
                    repoMetadata={repoMetadata}
                    latestCommit={resourceContent.latest_commit}
                    standaloneStyle
                  />
                  <Container className={css.container} background={Color.WHITE}>
                    <Layout.Horizontal padding="small" className={css.heading}>
                      <Heading level={5} color={Color.BLACK}>
                        {resourceContent.name}
                      </Heading>
                      <FlexExpander />
                      <Layout.Horizontal spacing="xsmall" style={{ alignItems: 'center' }}>
                        <PlainButton
                          withoutCurrentColor
                          size={ButtonSize.SMALL}
                          variation={ButtonVariation.TERTIARY}
                          iconProps={{ size: 16 }}
                          text={getString('edit')}
                          icon="code-edit"
                          tooltipProps={{ isDark: true }}
                          tooltip={permsFinal.tooltip}
                          disabled={permsFinal.disabled}
                          onClick={() => {
                            history.push(
                              routes.toCODEFileEdit({
                                repoPath: repoMetadata.path as string,
                                gitRef,
                                resourcePath
                              })
                            )
                          }}
                        />
                        <OptionsMenuButton
                          isDark={true}
                          icon="Options"
                          iconProps={{ size: 14 }}
                          style={{ padding: '5px' }}
                          width="145px"
                          items={[
                            {
                              hasIcon: true,
                              iconName: 'arrow-right',
                              text: getString('viewRaw'),
                              onClick: () => {
                                window.open(
                                  `/code/api/v1/repos/${
                                    repoMetadata?.path
                                  }/+/raw/${resourcePath}?${`git_ref=${gitRef}`}`,
                                  '_blank'
                                )
                              }
                            },
                            '-',
                            {
                              hasIcon: true,
                              iconName: 'cloud-download',
                              text: getString('download'),
                              download: resourceContent?.name || 'download',
                              href: `/code/api/v1/repos/${
                                repoMetadata?.path
                              }/+/raw/${resourcePath}?${`git_ref=${gitRef}`}`
                            },
                            {
                              hasIcon: true,
                              iconName: 'code-copy',
                              iconSize: 16,
                              text: getString('copy'),
                              onClick: () => Utils.copy(content)
                            },
                            {
                              hasIcon: true,
                              iconName: 'code-delete',
                              iconSize: 16,
                              title: getString(isRefATag(gitRef) ? 'deleteNotAllowed' : 'delete'),
                              disabled: isRefATag(gitRef),
                              text: getString('delete'),
                              onClick: openDeleteFileModal
                            }
                          ]}
                        />
                      </Layout.Horizontal>
                    </Layout.Horizontal>

                    <Render when={(resourceContent?.content as RepoFileContent)?.data}>
                      <Container className={css.content}>
                        <Render when={!markdownInfo}>
                          <SourceCodeViewer language={filenameToLanguage(resourceContent?.name)} source={content} />
                        </Render>
                        <Render when={markdownInfo}>
                          <Readme
                            metadata={repoMetadata}
                            readmeInfo={markdownInfo as OpenapiContentInfo}
                            contentOnly
                            maxWidth="calc(100vw - 346px)"
                            gitRef={gitRef}
                          />
                        </Render>
                      </Container>
                    </Render>
                  </Container>
                </Layout.Vertical>
              </Container>
            )
          },
          {
            id: FileSection.BLAME,
            title: getString('blame'),
            panel: (
              <Container className={css.gitBlame}>
                {[resourcePath + gitRef].map(key => (
                  <GitBlame repoMetadata={repoMetadata} resourcePath={resourcePath} gitRef={gitRef} key={key} />
                ))}
              </Container>
            )
          },
          {
            id: FileSection.HISTORY,
            title: getString('history'),
            panel: (
              <>
                {repoMetadata && !!commits?.commits?.length && (
                  <>
                    <Container className={css.gitBlame}>
                      <CommitsView
                        commits={commits.commits}
                        repoMetadata={repoMetadata}
                        emptyTitle={getString('noCommits')}
                        emptyMessage={getString('noCommitsMessage')}
                        showFileHistoryIcons={true}
                        gitRef={gitRef}
                        resourcePath={resourcePath}
                        setActiveTab={setActiveTab}
                      />
                      {/* <ThreadSection></ThreadSection> */}

                      <ResourceListingPagination response={response} page={page} setPage={setPage} />
                    </Container>
                    <Container className={css.gitHistory}>
                      {commits?.rename_details && repoMetadata ? (
                        <RenameContentHistory rename_details={commits.rename_details} repoMetadata={repoMetadata} />
                      ) : null}
                    </Container>
                  </>
                )}
              </>
            )
          }
        ]}
      />
    </Container>
  )
}
