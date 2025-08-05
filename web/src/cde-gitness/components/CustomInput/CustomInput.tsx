import React from 'react'
import { Container, Text, TextInput } from '@harnessio/uicore'
import FormError from '../FormError/FormError'
import css from './CustomInput.module.scss'

interface CustomInputProps {
  label: string
  name: string
  value: string
  onChange: (value: any) => void
  placeholder?: string
  error: any
  type: string
  autoComplete?: string
}

function CustomInput({ label, name, value, onChange, placeholder, error, type, autoComplete }: CustomInputProps) {
  return (
    <Container className={css.mb15}>
      <Text className={css.inputLabel}>{label}</Text>
      <TextInput
        name={name}
        placeholder={placeholder}
        value={value}
        type={type}
        className={error ? css.errorClass : ''}
        autoComplete={autoComplete}
        onChange={(e: React.FormEvent) => onChange(e.target)}
      />
      <FormError message={error} />
    </Container>
  )
}

export default CustomInput
