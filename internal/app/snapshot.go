package app

import "context"

type Status struct {
	Version                                                          string
	Installed, NodeAvailable, NpmAvailable                           bool
	RemoteProcesses, RemoteRoots, OtherDcrProcesses, CacheCandidates int
	CacheKnown                                                       bool
}
type Runtime struct{ Node, Npm, DeclaredNodeRange, Assessment string }
type Installation struct {
	Kind, Version, NodeRange string
	EntryExists              bool
}
type Process struct {
	PID, Parent    int
	Created        uint64
	Role, Location string
}
type Health struct {
	PID, Parent int
	Assessment  string
}
type Client struct {
	Client, Status, Modes string
	Entries, Floating     int
}
type Log struct {
	Name  string
	Codes []string
}
type Backup struct {
	Name    string
	AgeDays int
	Review  bool
}
type Snapshot struct {
	ProcessControlReady bool
	ProcessControlIssue string
	Status              Status
	Runtime             Runtime
	Installations       []Installation
	Processes           []Process
	Health              []Health
	Clients             []Client
	Logs                []Log
	Backups             []Backup
	Caches              []CacheCandidate
}
type Reader func(context.Context) (Snapshot, error)
