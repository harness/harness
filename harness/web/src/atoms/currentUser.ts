import { atom } from 'jotai'
import type { TypesUser } from 'services/code'

export const currentUserAtom = atom<TypesUser | undefined>(undefined)
