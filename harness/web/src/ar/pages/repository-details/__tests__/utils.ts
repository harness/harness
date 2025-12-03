/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import userEvent from '@testing-library/user-event'
import type { Registry } from '@harnessio/react-har-service-client'
import { fireEvent, getByTestId, screen } from '@testing-library/react'
import { queryByNameAttribute } from 'utils/test/testUtils'

async function openModal(container: HTMLElement): Promise<Element> {
  const pageSubHeader = getByTestId(container, 'page-subheader')
  const createRegistryButton = pageSubHeader.querySelector('button span[icon="chevron-down"]')
  expect(createRegistryButton).toBeInTheDocument()
  await userEvent.click(createRegistryButton!)

  const createUpstreamRegistryBtn = screen.getByText('repositoryList.upstreamProxy.label')
  await userEvent.click(createUpstreamRegistryBtn!)

  const modal = document.getElementsByClassName('bp3-dialog')[0]
  expect(modal).toBeInTheDocument()
  return modal
}

async function verifyPackageTypeSelector(container: HTMLElement, packageType: string): Promise<void> {
  expect(container).toHaveTextContent('repositoryDetails.repositoryForm.selectRepoType')
  const packageTypeOption = container.querySelector(`input[type="checkbox"][name=packageType][value=${packageType}]`)
  expect(packageTypeOption).not.toBeDisabled()
  await userEvent.click(packageTypeOption!)
  expect(packageTypeOption).toBeChecked()
}

async function verifySourceAndAuthSection(
  container: HTMLElement,
  source: string,
  authType: string,
  defaultSource = 'Dockerhub',
  defaultAuthType = 'Anonymous'
) {
  // add source
  const sourceOption = container.querySelector(`input[type="radio"][name="config.source"][value="${source}"]`)
  expect(sourceOption).not.toBeDisabled()
  if (defaultSource !== source) {
    await userEvent.click(sourceOption!)
  }
  expect(sourceOption).toBeChecked()

  // add registry url
  if (source === 'Custom') {
    const urlField = queryByNameAttribute('config.url', container)
    expect(urlField).toBeInTheDocument()
    expect(urlField).not.toBeDisabled()
    fireEvent.change(urlField!, { target: { value: 'https://custom.docker.com' } })
  } else if (source === 'AwsEcr') {
    const urlField = queryByNameAttribute('config.url', container)
    expect(urlField).toBeInTheDocument()
    expect(urlField).not.toBeDisabled()
    fireEvent.change(urlField!, { target: { value: 'https://aws.ecr.com' } })
  } else {
    const urlField = queryByNameAttribute('config.url', container)
    expect(urlField).not.toBeInTheDocument
  }

  // add auth
  const authTypeOption = container.querySelector(`input[type="radio"][name="config.authType"][value="${authType}"]`)
  expect(authTypeOption).not.toBeDisabled()
  if (defaultAuthType !== authType) {
    await userEvent.click(authTypeOption!)
  }
  expect(authTypeOption).toBeChecked()

  if (authType === 'UserPassword') {
    // add username password
    const usernameField = queryByNameAttribute('config.auth.userName', container)
    expect(usernameField).toBeInTheDocument()
    fireEvent.change(usernameField!, { target: { value: 'username' } })

    const passwordField = queryByNameAttribute('config.auth.secretIdentifier', container)
    expect(passwordField).toBeInTheDocument()
    fireEvent.change(passwordField!, { target: { value: 'password' } })
  } else if (authType === 'AccessKeySecretKey') {
    // add access key and secret key
    const accessKeyField = queryByNameAttribute('config.auth.accessKey', container)
    expect(accessKeyField).toBeInTheDocument()
    fireEvent.change(accessKeyField!, { target: { value: 'accessKey' } })

    const secretKeyField = queryByNameAttribute('config.auth.secretKeyIdentifier', container)
    expect(secretKeyField).toBeInTheDocument()
    fireEvent.change(secretKeyField!, { target: { value: 'secretKey' } })
  } else {
    const usernameField = queryByNameAttribute('config.auth.userName', container)
    expect(usernameField).not.toBeInTheDocument()

    const passwordField = queryByNameAttribute('config.auth.secretIdentifier', container)
    expect(passwordField).not.toBeInTheDocument()

    const accessKeyField = queryByNameAttribute('config.auth.accessKey', container)
    expect(accessKeyField).not.toBeInTheDocument()

    const secretKeyField = queryByNameAttribute('config.auth.secretKeyIdentifier', container)
    expect(secretKeyField).not.toBeInTheDocument()
  }
}

async function verifyUpstreamProxyCreateForm(
  container: HTMLElement,
  formData: Registry,
  source: string,
  authType: string,
  defaultSource = 'Dockerhub',
  defaultAuthType = 'Anonymous'
): Promise<void> {
  // Add name
  const nameField = queryByNameAttribute('identifier', container)
  expect(nameField).not.toBeDisabled()
  fireEvent.change(nameField!, { target: { value: formData.identifier } })

  // add description
  const descriptionEditButton = getByTestId(container, 'description-edit')
  expect(descriptionEditButton).toBeInTheDocument()
  await userEvent.click(descriptionEditButton)
  const descriptionField = queryByNameAttribute('description', container)
  expect(descriptionField).not.toBeDisabled()
  fireEvent.change(descriptionField!, { target: { value: formData.description } })

  // verify source and auth section
  await verifySourceAndAuthSection(container, source, authType, defaultSource, defaultAuthType)
}

async function getSubmitButton(): Promise<Element> {
  const dialogFooter = screen.getByTestId('modaldialog-footer')
  expect(dialogFooter).toBeInTheDocument()
  const createButton = dialogFooter.querySelector(
    'button[type="submit"][aria-label="upstreamProxyDetails.createForm.create"]'
  )
  expect(createButton).toBeInTheDocument()
  return createButton!
}

const upstreamProxyUtils = {
  openModal,
  verifyPackageTypeSelector,
  verifyUpstreamProxyCreateForm,
  verifySourceAndAuthSection,
  getSubmitButton
}

export default upstreamProxyUtils
