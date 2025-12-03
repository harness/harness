import moment from 'moment'

export const DATE_FORMAT = 'MM/DD/YYYY'

export const getReadableDateTime = (timestamp?: number, formatString = 'MMM DD, YYYY'): string => {
  if (!timestamp) {
    return ''
  }
  return moment(timestamp).format(formatString)
}

export const getSelectedExpirationDate = (value: string): string => {
  switch (value) {
    case '30':
    case '90':
    case '180':
      return getReadableDateTime(new Date().setDate(new Date().getDate() + parseInt(value)), DATE_FORMAT)
    case '-1':
      return '12/31/2099'
    default:
      return ''
  }
}
