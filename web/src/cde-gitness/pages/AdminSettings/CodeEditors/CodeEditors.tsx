import React, { useEffect, useState, useMemo } from 'react'
import { Card, Text, Layout, Checkbox, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext, getIn } from 'formik'
import { useStrings } from 'framework/strings'
import type { StringsMap } from 'framework/strings/stringTypes'
import type { EnumIDEType } from 'services/cde'
import { getIDETypeOptions, groupEnums } from 'cde-gitness/constants'
import type { AdminSettingsFormValues } from '../utils/adminSettingsUtils'
import css from './CodeEditors.module.scss'

type Editor = { value: EnumIDEType; label: string; icon: string; group: string }
type EditorSection = { titleKey: keyof StringsMap; editors: Editor[] }

const editorSectionsTemplate: EditorSection[] = [
  { titleKey: 'cde.settings.editors.vsCode', editors: [] },
  { titleKey: 'cde.settings.editors.jetbrains', editors: [] }
]

const groupMapping: Record<string, keyof StringsMap> = {
  [groupEnums.VSCODE]: 'cde.settings.editors.vsCode',
  [groupEnums.JETBRAIN]: 'cde.settings.editors.jetbrains'
}

const CodeEditors: React.FC = () => {
  const { getString } = useStrings()
  const { values, setFieldValue } = useFormikContext<AdminSettingsFormValues>()
  const [selectAllChecked, setSelectAllChecked] = useState(true)

  const availableEditors = useMemo(() => getIDETypeOptions(getString), [getString])

  const editorSections: EditorSection[] = useMemo(() => {
    const sections: EditorSection[] = structuredClone(editorSectionsTemplate)

    availableEditors.forEach(editor => {
      const sectionKey = groupMapping[editor.group]
      const section = sections.find((s: EditorSection) => s.titleKey === sectionKey)

      if (section) {
        section.editors.push({
          value: editor.value,
          label: editor.label,
          icon: editor.icon,
          group: editor.group
        })
      }
    })

    return sections
  }, [availableEditors])

  useEffect(() => {
    if (values.codeEditors) {
      const allEditors = availableEditors.map(e => e.value)
      const allSelected = allEditors.every(editor => values.codeEditors[editor])
      setSelectAllChecked(allSelected)
    }
  }, [values.codeEditors, availableEditors])

  const handleSelectAll = (checked: boolean) => {
    setSelectAllChecked(checked)
    const newCodeEditorValues = { ...values.codeEditors }
    availableEditors.forEach(editor => {
      newCodeEditorValues[editor.value] = checked
    })
    setFieldValue('codeEditors', newCodeEditorValues)
  }

  return (
    <div className={css.codeEditorsContainer}>
      <Layout.Vertical spacing="small">
        <Text font={{ variation: FontVariation.H5 }}>{getString('cde.settings.availableCodeEditors')}</Text>
        <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
          {getString('cde.settings.codeEditorsDescription')}
        </Text>
      </Layout.Vertical>

      <Card className={css.mainContentCard}>
        <Layout.Vertical spacing="medium">
          <Checkbox
            checked={selectAllChecked}
            className={css.checkbox}
            labelElement={
              <Layout.Horizontal spacing="small">
                <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_700}>
                  {getString('cde.settings.selectAllCodeEditors')}
                </Text>
              </Layout.Horizontal>
            }
            onChange={event => handleSelectAll(event.currentTarget.checked)}
          />

          <Layout.Vertical spacing="medium">
            {editorSections.map((section: EditorSection) => {
              return (
                <Layout.Vertical spacing="medium" key={section.titleKey}>
                  <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                    {getString(section.titleKey)}
                  </Text>
                  <div className={css.editorsGrid}>
                    {section.editors.map((editor: Editor) => (
                      <Container className={css.editorCheckbox} key={editor.value}>
                        <Checkbox
                          className={css.checkbox}
                          checked={getIn(values, `codeEditors.${editor.value}`) || false}
                          labelElement={
                            <Layout.Horizontal spacing="small">
                              <img src={editor.icon} alt={editor.label} width={24} height={24} />
                              <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_700}>
                                {editor.label}
                              </Text>
                            </Layout.Horizontal>
                          }
                          onChange={event => setFieldValue(`codeEditors.${editor.value}`, event.currentTarget.checked)}
                        />
                      </Container>
                    ))}
                  </div>
                </Layout.Vertical>
              )
            })}
          </Layout.Vertical>
        </Layout.Vertical>
      </Card>
    </div>
  )
}

export default CodeEditors
