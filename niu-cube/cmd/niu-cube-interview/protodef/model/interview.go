package model

type Interview struct {
	ID          string
	RoomId      string
	CandidateId string
	CreatorId   string
	Status      InterviewStatus
}

type InterviewStatus string
