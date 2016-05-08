package yaml

// Constraints define constraints for container execution.
type Constraints struct {
	Platform    []string
	Environment []string
	Event       []string
	Branch      []string
	Status      []string
	Matrix      map[string]string
}

//
// // Constraint defines an individual contraint.
// type Constraint struct {
// 	Include []string
// 	Exclude []string
// }
//
// // Match returns true if the branch matches the include patterns and does not
// // match any of the exclude patterns.
// func (c *Constraint) Match(v string) bool {
// 	// when no includes or excludes automatically match
// 	if len(c.Include) == 0 && len(c.Exclude) == 0 {
// 		return true
// 	}
//
// 	// exclusions are processed first. So we can include everything and then
// 	// selectively exclude certain sub-patterns.
// 	for _, pattern := range c.Exclude {
// 		if pattern == v {
// 			return false
// 		}
// 		if ok, _ := filepath.Match(pattern, v); ok {
// 			return false
// 		}
// 	}
//
// 	for _, pattern := range c.Include {
// 		if pattern == v {
// 			return true
// 		}
// 		if ok, _ := filepath.Match(pattern, v); ok {
// 			return true
// 		}
// 	}
//
// 	return false
// }
