// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"
	"fmt"
	"strings"
)

/* https://git-scm.com/docs/git-check-ref-format
 * How to handle various characters in refnames:
 * 0: An acceptable character for refs
 * 1: End-of-component
 * 2: ., look for a preceding . to reject .. in refs
 * 3: {, look for a preceding @ to reject @{ in refs
 * 4: A bad character: ASCII control characters, and
 *    ":", "?", "[", "\", "^", "~", SP, or TAB
 * 5: *, reject unless REFNAME_REFSPEC_PATTERN is set
 */
var refnameDisposition = [256]byte{
	1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 2, 1,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 4,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 4, 0, 4, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 4, 4,
}

func BranchName(branch string) error {
	lock := ".lock"
	last := '\x00'

	for i, c := range branch {
		ch := c & 255
		disp := refnameDisposition[ch]

		switch disp {
		case 1:
			if i == 0 {
				goto out
			}
			if last == '/' { // Refname contains "//"
				return fmt.Errorf("branch '%s' cannot have two consecutive slashes // ", branch)
			}
		case 2:
			if last == '.' { // Refname contains ".."
				return fmt.Errorf("branch '%s' cannot have two consecutive dots .. ", branch)
			}
		case 3:
			if last == '@' { // Refname contains "@{".
				return fmt.Errorf("branch '%s' cannot contain a sequence @{", branch)
			}
		case 4:
			return fmt.Errorf("branch '%s' cannot have ASCII control characters "+
				"(i.e. bytes whose values are lower than \040, or \177 DEL), space, tilde ~, caret ^, or colon : anywhere", branch)
		case 5:
			return fmt.Errorf("branch '%s' can't be a pattern", branch)
		}
		last = ch
	}
out:
	if last == '\x00' {
		return errors.New("branch name is empty")
	}
	if last == '.' {
		return fmt.Errorf("branch '%s' cannot have . at the end", branch)
	}
	if last == '@' {
		return fmt.Errorf("branch '%s' cannot be the single character @", branch)
	}
	if last == '/' {
		return fmt.Errorf("branch '%s' cannot have / at the end", branch)
	}
	if branch[0] == '.' {
		return fmt.Errorf("branch '%s' cannot start with '.'", branch)
	}
	if strings.HasSuffix(branch, lock) {
		return fmt.Errorf("branch '%s' cannot ends with '%s'", branch, lock)
	}
	return nil
}
