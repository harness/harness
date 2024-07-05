import { atom } from 'jotai'
import type { RepoRepositoryOutput } from 'services/code'

export const repoMetadataAtom = atom<RepoRepositoryOutput | undefined>(undefined)
