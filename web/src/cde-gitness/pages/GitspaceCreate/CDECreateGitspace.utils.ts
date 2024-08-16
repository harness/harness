import { StandaloneIDEType } from 'cde-gitness/constants'
import type { UseStringsReturn } from 'framework/strings'

export const getIDESelectItems = (getString: UseStringsReturn['getString']) => {
  return [
    { label: getString('cde.ide.desktop'), value: StandaloneIDEType.VSCODE },
    { label: getString('cde.ide.browser'), value: StandaloneIDEType.VSCODEWEB }
  ]
}
