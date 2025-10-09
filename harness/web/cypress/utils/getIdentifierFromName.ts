/*
 * Copyright 2024 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

export const getIdentifierFromName = (str: string): string => {
  return str
    .trim()
    .replace(/[^0-9a-zA-Z_$\- ]/g, '') // remove special chars except _ and $
    .replace(/ +/g, '_') // replace spaces with _
}
