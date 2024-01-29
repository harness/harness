import { atom } from 'jotai'
import type { TypesRepository } from 'services/code'

export const repoMetadataAtom = atom<TypesRepository | undefined>(undefined)
