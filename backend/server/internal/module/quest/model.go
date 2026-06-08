package quest

type Template struct {
	QuestID     uint64
	QuestType   string
	Title       string
	Description string
	AcceptMode  string
	SubmitMode  string
	StartNPCID  uint64
	SubmitNPCID uint64
	AutoTrack   bool
	PreQuestIDs []uint64
	Objectives  []ObjectiveTemplate
}

type ObjectiveTemplate struct {
	ObjectiveID         uint64
	EventType           string
	Description         string
	TargetValue         uint32
	TargetSelector      map[string]any
	AutoCompleteOnMatch bool
}

type PlayerQuest struct {
	PlayerID uint64
	QuestID  uint64
	State    string
	Tracked  bool
}

type PlayerObjective struct {
	PlayerID     uint64
	QuestID      uint64
	ObjectiveID  uint64
	Description  string
	CurrentValue uint32
	TargetValue  uint32
	Completed    bool
}

type Summary struct {
	QuestID     uint64
	QuestType   string
	State       string
	Tracked     bool
	StartNPCID  uint64
	SubmitNPCID uint64
	Title       string
	Description string
	Objectives  []ObjectiveSummary
}

type ObjectiveSummary struct {
	ObjectiveID uint64
	Description string
	Current     uint32
	Target      uint32
	Completed   bool
}

type Event struct {
	PlayerID  uint64
	EventType string
	SceneID   uint32
	NPCID     uint64
	Count     uint32
	Meta      map[string]any
}
