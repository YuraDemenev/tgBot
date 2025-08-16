package handlers

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

// func (s Status) String() string {
// 	switch s {

// 	}
// }
