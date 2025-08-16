package states

//Enums for session

type Status int

const (
	Unknown Status = iota
	Start
	AddTask
	DeleteTask
	ChangeTask
	MyTasks
)

func GetZeroValue() Status {
	return Unknown
}
