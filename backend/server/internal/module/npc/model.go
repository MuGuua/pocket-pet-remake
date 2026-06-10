package npc

type MenuEntry struct {
	EntityID         uint64
	EntryID          string
	EntryType        string
	Title            string
	Subtitle         string
	State            string
	Priority         uint32
	ActionResultType string
	ActionNotice     string
}

type ActionResult struct {
	EntityID   uint64
	EntryID    string
	ResultType string
	Notice     string
}
