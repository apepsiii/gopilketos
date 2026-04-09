package models

type Admin struct {
	ID           uint
	Username     string
	PasswordHash string
	CreatedAt    string
}

type Setting struct {
	ID               uint
	AnnouncementText string
	UpdatedAt        string
}

type Candidate struct {
	ID        uint
	Name      string
	ClassName string
	PhotoURL  string
	Vision    string
	Mission   string
	Program   string
	Position  string
	CreatedAt string
}

type Voter struct {
	ID          uint
	UUID        string
	Name        string
	ClassName   string
	PhoneNumber string
	HasVoted    bool
	CreatedAt   string
}

type Vote struct {
	ID             uint
	MaskedUUID     string
	ChairmanID     uint
	ViceChairmanID uint
	VotedAt        string
}
