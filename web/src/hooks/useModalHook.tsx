/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useState, useCallback, useMemo, memo, useContext } from 'react'
import ReactDOM from 'react-dom'
import { noop } from 'lodash-es'

type ModalType = React.FunctionComponent<Unknown>

interface ModalContextType {
  showModal(key: string, component: ModalType): void
  hideModal(key: string): void
}

const ModalContext = React.createContext<ModalContextType>({
  showModal: noop,
  hideModal: noop
})

interface ModalRootProps {
  modals: Record<string, ModalType>
  component?: React.ComponentType<Unknown>
  container?: Element
}

interface ModalRendererProps {
  component: ModalType
}

const ModalRenderer = memo(({ component, ...rest }: ModalRendererProps) => component(rest))
ModalRenderer.displayName = 'ModalRenderer'

const ModalRoot: Unknown = memo(({ modals, container, component: RootComponent = React.Fragment }: ModalRootProps) => {
  const [mountNode, setMountNode] = useState<Element | undefined>(undefined)

  useEffect(() => {
    setMountNode(container || document.body)
  }, [container])

  return mountNode
    ? ReactDOM.createPortal(
        <RootComponent>
          {Object.keys(modals).map(key => (
            <ModalRenderer key={key} component={modals[key]} />
          ))}
        </RootComponent>,
        mountNode
      )
    : null
})

ModalRoot.displayName = 'ModalRoot'

interface ModalProviderProps {
  container?: Element
  rootComponent?: React.ComponentType<Unknown>
  children: React.ReactNode
}

export const ModalProvider = ({ container, rootComponent, children }: ModalProviderProps) => {
  if (container && !(container instanceof HTMLElement)) {
    throw new Error('Container must specify DOM element to mount modal root into.')
  }
  const [modals, setModals] = useState<Record<string, ModalType>>({})
  const showModal = useCallback(
    (key: string, modal: ModalType) =>
      setModals(_modals => ({
        ..._modals,
        [key]: modal
      })),
    []
  )
  const hideModal = useCallback(
    (key: string) =>
      setModals(_modals => {
        const newModals = { ..._modals }
        delete newModals[key]
        return newModals
      }),
    []
  )
  const contextValue = useMemo(() => ({ showModal, hideModal }), []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <ModalContext.Provider value={contextValue}>
      <React.Fragment>
        {children}
        <ModalRoot modals={modals} component={rootComponent} container={container} />
      </React.Fragment>
    </ModalContext.Provider>
  )
}

type ShowModal = () => void
type HideModal = () => void

const generateModalKey = (() => {
  let count = 0
  return () => `${++count}`
})()

const isFunctionalComponent = (Component: React.FunctionComponent) => {
  const prototype = Component.prototype
  return !prototype || !prototype.isReactComponent
}

/**
 * @deprecated, use UICore ModalDialog instead.
 */
export const useModalHook = (component: ModalType, inputs: Unknown[] = []): [ShowModal, HideModal] => {
  if (!isFunctionalComponent(component)) {
    throw new Error(
      'Only stateless components can be used as an argument to useModal. You have probably passed a class component where a function was expected.'
    )
  }

  const key = useMemo(generateModalKey, [])
  const modal = useMemo(() => component, inputs) // eslint-disable-line react-hooks/exhaustive-deps
  const context = useContext(ModalContext)
  const [isShown, setShown] = useState<boolean>(false)
  const showModal = useCallback(() => setShown(true), [])
  const hideModal = useCallback(() => setShown(false), [])

  useEffect(() => {
    if (isShown) {
      context.showModal(key, modal)
    } else {
      context.hideModal(key)
    }

    return () => context.hideModal(key)
  }, [modal, isShown]) // eslint-disable-line react-hooks/exhaustive-deps

  return [showModal, hideModal]
}
