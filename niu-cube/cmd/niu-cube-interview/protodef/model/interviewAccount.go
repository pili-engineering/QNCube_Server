package model

type InterviewAccount struct {
	InterviewId string
	AccountId   string
	// Status User's Interview Status
	Status InterviewAccountStatus
}

type InterviewAccountStatus string

const (
	InterviewAccountStatusIn  = "in"
	InterviewAccountStatusNil = "nil"
	InterviewAccountStatusOut = "out"
)
