package errors

type PermissionError struct {
	msg string
	err error
}

func (r *PermissionError) Error() string {
	if r.err != nil {
		return r.msg + r.err.Error()
	} else {
		return r.msg
	}
}

func (r *PermissionError) Unwrap() error {
	return r.err
}

func NewPermissionError(msg string, errs ...error) *PermissionError {
	if len(errs) > 1 {
		panic("only 1 sub err")
	}
	e := &PermissionError{
		msg: msg,
	}
	if len(errs) == 1 {
		e.err = errs[0]
	}
	return e
}

type DBError = PermissionError

func NewDBError(msg string, errs ...error) *DBError {
	if len(errs) > 1 {
		panic("only 1 sub err")
	}
	e := &DBError{
		msg: msg,
	}
	if len(errs) == 1 {
		e.err = errs[0]
	}
	return e
}

type FormValidationError = PermissionError

func NewFormValidationError(msg string, errs ...error) *FormValidationError {
	if len(errs) > 1 {
		panic("only 1 sub err")
	}
	e := &FormValidationError{
		msg: msg,
	}
	if len(errs) == 1 {
		e.err = errs[0]
	}
	return e
}
