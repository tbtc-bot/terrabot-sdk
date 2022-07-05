package terrabot

type Metadata struct {
	LastGridReached   int64            `json:"lastGridReached"`
	MapIDtoGridNumber map[string]int64 `json:"idToGridNumberMap"`
}
