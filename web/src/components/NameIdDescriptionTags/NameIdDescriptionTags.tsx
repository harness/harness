import React, { useState } from 'react'
import { Container, FormInput, Icon, Label, DataTooltipInterface, HarnessDocTooltip } from '@harness/uicore'
import type { InputWithIdentifierProps } from '@harness/uicore/dist/components/InputWithIdentifier/InputWithIdentifier'
import { isEmpty } from 'lodash-es'
import { Classes, IInputGroupProps, ITagInputProps } from '@blueprintjs/core'
import cx from 'classnames'
import type { FormikProps } from 'formik'
import { useStrings } from 'framework/strings'
import type {
  DescriptionComponentProps,
  DescriptionProps,
  NameIdDescriptionProps,
  NameIdDescriptionTagsDeprecatedProps,
  TagsComponentProps,
  TagsDeprecatedComponentProps
} from './NameIdDescriptionTagsConstants'
import css from './NameIdDescriptionTags.module.scss'

export interface NameIdDescriptionTagsProps {
  identifierProps?: Omit<InputWithIdentifierProps, 'formik'>
  inputGroupProps?: IInputGroupProps
  descriptionProps?: DescriptionProps
  tagsProps?: Partial<ITagInputProps> & {
    isOption?: boolean
  }
  formikProps: FormikProps<any>
  className?: string
  tooltipProps?: DataTooltipInterface
}

interface NameIdProps {
  nameLabel?: string // Strong default preference for "Name" vs. Contextual Name (e.g. "Service Name") unless approved otherwise
  namePlaceholder?: string
  identifierProps?: Omit<InputWithIdentifierProps, 'formik'>
  inputGroupProps?: IInputGroupProps
  dataTooltipId?: string
}

export const NameId = (props: NameIdProps): JSX.Element => {
  const { getString } = useStrings()
  const { identifierProps, nameLabel = getString('name'), inputGroupProps = {} } = props
  const newInputGroupProps = { placeholder: getString('namePlaceholder'), ...inputGroupProps }
  return (
    <FormInput.InputWithIdentifier inputLabel={nameLabel} inputGroupProps={newInputGroupProps} {...identifierProps} />
  )
}

export const Description = (props: DescriptionComponentProps): JSX.Element => {
  const { descriptionProps = {}, hasValue, disabled = false } = props
  const { isOptional = true, ...restDescriptionProps } = descriptionProps
  const { getString } = useStrings()
  const [isDescriptionOpen, setDescriptionOpen] = useState<boolean>(hasValue || false)
  const [isDescriptionFocus, setDescriptionFocus] = useState<boolean>(false)

  return (
    <Container style={{ marginBottom: isDescriptionOpen ? '0' : 'var(--spacing-medium)' }}>
      <Label className={cx(Classes.LABEL, css.descriptionLabel)} data-tooltip-id={props.dataTooltipId}>
        {isOptional ? getString('optionalField', { name: getString('description') }) : getString('description')}
        {props.dataTooltipId ? <HarnessDocTooltip useStandAlone={true} tooltipId={props.dataTooltipId} /> : null}
        {!isDescriptionOpen && (
          <Icon
            className={css.editOpen}
            data-name="edit"
            data-testid="description-edit"
            size={12}
            name="Edit"
            onClick={() => {
              setDescriptionOpen(true)
              setDescriptionFocus(true)
            }}
          />
        )}
      </Label>
      {isDescriptionOpen && (
        <FormInput.TextArea
          data-name="description"
          disabled={disabled}
          autoFocus={isDescriptionFocus}
          name="description"
          placeholder={getString('descriptionPlaceholder')}
          {...restDescriptionProps}
        />
      )}
    </Container>
  )
}

export const Tags = (props: TagsComponentProps): JSX.Element => {
  const { tagsProps, hasValue, isOptional = true } = props
  const { getString } = useStrings()
  const [isTagsOpen, setTagsOpen] = useState<boolean>(hasValue || false)

  return (
    <Container>
      <Label className={cx(Classes.LABEL, css.descriptionLabel)} data-tooltip-id={props.dataTooltipId}>
        {isOptional ? getString('optionalField', { name: getString('tagsLabel') }) : getString('tagsLabel')}
        {props.dataTooltipId ? <HarnessDocTooltip useStandAlone={true} tooltipId={props.dataTooltipId} /> : null}
        {!isTagsOpen && (
          <Icon
            className={css.editOpen}
            data-name="edit"
            data-testid="tags-edit"
            size={12}
            name="Edit"
            onClick={() => {
              setTagsOpen(true)
            }}
          />
        )}
      </Label>
      {isTagsOpen && <FormInput.KVTagInput name="tags" tagsProps={tagsProps} />}
    </Container>
  )
}

function TagsDeprecated(props: TagsDeprecatedComponentProps): JSX.Element {
  const { hasValue } = props
  const { getString } = useStrings()
  const [isTagsOpen, setTagsOpen] = useState<boolean>(hasValue || false)

  return (
    <Container>
      <Label className={cx(Classes.LABEL, css.descriptionLabel)}>
        {getString('tagsLabel')}
        {!isTagsOpen && (
          <Icon
            className={css.editOpen}
            data-name="Edit"
            size={12}
            name="edit"
            onClick={() => {
              setTagsOpen(true)
            }}
          />
        )}
      </Label>
      {isTagsOpen && (
        <FormInput.TagInput
          name="tags"
          labelFor={name => (typeof name === 'string' ? name : '')}
          itemFromNewTag={newTag => newTag}
          items={[]}
          tagInputProps={{
            noInputBorder: true,
            openOnKeyDown: false,
            showAddTagButton: true,
            showClearAllButton: true,
            allowNewTag: true
          }}
        />
      )}
    </Container>
  )
}

export function NameIdDescriptionTags(props: NameIdDescriptionTagsProps): JSX.Element {
  const { getString } = useStrings()
  const { className, identifierProps, descriptionProps, formikProps, inputGroupProps = {}, tooltipProps } = props
  const newInputGroupProps = { placeholder: getString('namePlaceholder'), ...inputGroupProps }
  return (
    <Container className={cx(css.main, className)}>
      <NameId identifierProps={identifierProps} inputGroupProps={newInputGroupProps} />
      <Description
        descriptionProps={descriptionProps}
        hasValue={!!formikProps?.values.description}
        dataTooltipId={tooltipProps?.dataTooltipId ? `${tooltipProps.dataTooltipId}_description` : undefined}
      />
    </Container>
  )
}

// Requires verification with existing tags
export function NameIdDescriptionTagsDeprecated<T>(props: NameIdDescriptionTagsDeprecatedProps<T>): JSX.Element {
  const { className, identifierProps, descriptionProps, formikProps } = props
  return (
    <Container className={cx(css.main, className)}>
      <NameId identifierProps={identifierProps} />
      <Description descriptionProps={descriptionProps} hasValue={!!formikProps?.values.description} />
      <TagsDeprecated hasValue={!isEmpty(formikProps?.values.tags)} />
    </Container>
  )
}

export function NameIdDescription(props: NameIdDescriptionProps): JSX.Element {
  const { getString } = useStrings()
  const { className, identifierProps, descriptionProps, formikProps, inputGroupProps = {} } = props
  const newInputGroupProps = { placeholder: getString('namePlaceholder'), ...inputGroupProps }

  return (
    <Container className={cx(css.main, className)}>
      <NameId identifierProps={identifierProps} inputGroupProps={newInputGroupProps} />
      <Description descriptionProps={descriptionProps} hasValue={!!formikProps?.values.description} />
    </Container>
  )
}

export function DescriptionTags(props: Omit<NameIdDescriptionTagsProps, 'identifierProps'>): JSX.Element {
  const { className, descriptionProps, tagsProps, formikProps } = props
  return (
    <Container className={cx(css.main, className)}>
      <Description descriptionProps={descriptionProps} hasValue={!!formikProps?.values.description} />
      <Tags tagsProps={tagsProps} isOptional={tagsProps?.isOption} hasValue={!isEmpty(formikProps?.values.tags)} />
    </Container>
  )
}
