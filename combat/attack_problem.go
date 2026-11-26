package combat

import "errors"

// joinProblems returns the errors.Join of ps, or nil for none, so a constructor
// returns an error if and only if at least one field is violated.
func joinProblems[P error](ps []P) error {
	if len(ps) == 0 {
		return nil
	}
	errs := make([]error, len(ps))
	for i, p := range ps {
		errs[i] = p
	}
	return errors.Join(errs...)
}

// problemText renders "combat: invalid <subject>: <field>" for a problem type;
// the text is not a contract.
func problemText(subject, field string) string { return "combat: invalid " + subject + ": " + field }
