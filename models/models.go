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
	ID              uint
	CandidateNumber int
	Name            string
	ClassName       string
	PhotoURL        string
	VideoURL        string
	Vision          string
	Mission         string
	Program         string
	Position        string
	CreatedAt       string
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

type Credit struct {
	ID        uint
	Name      string
	ClassName string
	Division  string
	PhotoURL  string
	Period    string
	OrderNum  int
	CreatedAt string
}

type Testimonial struct {
	ID           uint
	Name         string
	ClassName    string
	Message      string
	VideoPath    string
	VideoType    string
	GDriveFileID string
	GDriveURL    string
	SyncStatus   string
	SyncError    string
	IsApproved   bool
	CreatedAt    string
}

