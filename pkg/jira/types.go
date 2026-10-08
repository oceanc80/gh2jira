package jira

type Issue struct {
	ID     string
	Key    string
	Fields *IssueFields
}

type IssueFields struct {
	Summary     string
	Description string
	Type        IssueType
	Project     Project
	Priority    *Priority
	Status      *Status
	Assignee    *User
	Reporter    *User
}

type IssueType struct {
	Name string
}

type Project struct {
	Key string
}

type Priority struct {
	Name string
}

type Status struct {
	Name string
}

type User struct {
	DisplayName string
}

type RemoteLink struct {
	GlobalID     string
	Relationship string
	Object       *RemoteLinkObject
}

type RemoteLinkObject struct {
	URL   string
	Title string
}
