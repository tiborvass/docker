package cli

import "bytes"

type Errors []error

func (errs Errors) Error() string {
	if len(errs) < 1 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(errs[0].Error())
	for _, err := range errs[1:] {
		buf.WriteString(", ")
		buf.WriteString(err.Error())
	}
	return buf.String()
}
