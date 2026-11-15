package conf

import "strings"

// ReloadReport records the outcome of applying a configuration reload.
//
// A running process cannot restart its listeners, swap database drivers or
// rebuild a log core, so components split the reloaded settings into two
// groups: Applied (took effect immediately) and Restart (changed, but ignored
// until the process restarts). Callers usually log both groups so an operator
// knows why a value "did not take effect".
type ReloadReport struct {
	Applied []string
	Restart []string
}

// MarkApplied records settings that are now live.
func (r *ReloadReport) MarkApplied(fields ...string) {
	r.Applied = append(r.Applied, fields...)
}

// MarkRestartNeeded records settings that changed but require a restart.
func (r *ReloadReport) MarkRestartNeeded(fields ...string) {
	r.Restart = append(r.Restart, fields...)
}

// Changed reports whether the reload observed any difference at all.
func (r *ReloadReport) Changed() bool {
	return len(r.Applied) > 0 || len(r.Restart) > 0
}

// String renders the report as "applied=a,b restart=c".
func (r *ReloadReport) String() string {
	var b strings.Builder
	if len(r.Applied) > 0 {
		b.WriteString("applied=")
		b.WriteString(strings.Join(r.Applied, ","))
	}
	if len(r.Restart) > 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("restart=")
		b.WriteString(strings.Join(r.Restart, ","))
	}
	return b.String()
}
