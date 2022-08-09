export type LangLocale = 'es' | 'en' | 'en-IN' | 'en-US' | 'en-UK'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type LanguageRecord = Record<string, Record<string, any>>

export function languageLoader(langId: LangLocale = 'en'): Promise<LanguageRecord> {
  switch (langId) {
    case 'es':
      return import('../../i18n/strings.es.yaml')
    case 'en':
    case 'en-US':
    case 'en-IN':
    case 'en-UK':
    default:
      return import('../../i18n/strings.en.yaml')
  }
}
