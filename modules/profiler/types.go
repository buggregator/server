package profiler

// Metrics holds XHProf metric values.
type Metrics struct {
	WallTime int64 `json:"wt"`
	CPU      int64 `json:"cpu"`
	Memory   int64 `json:"mu"`
	PeakMem  int64 `json:"pmu"`
	Calls    int64 `json:"ct"`
}

// Percentages of peak values.
type Percentages struct {
	WallTime float64 `json:"p_wt"`
	CPU      float64 `json:"p_cpu"`
	Memory   float64 `json:"p_mu"`
	PeakMem  float64 `json:"p_pmu"`
	Calls    float64 `json:"p_ct"`
}

// Diffs (exclusive metrics).
type Diffs struct {
	WallTime int64 `json:"d_wt"`
	CPU      int64 `json:"d_cpu"`
	Memory   int64 `json:"d_mu"`
	PeakMem  int64 `json:"d_pmu"`
	Calls    int64 `json:"d_ct"`
}

// Edge represents a processed caller→callee relationship.
type Edge struct {
	ID       string      `json:"id"`
	Caller   *string     `json:"caller"`
	Callee   string      `json:"callee"`
	Cost     Metrics     `json:"cost"`
	Diff     Diffs       `json:"diff"`
	Percents Percentages `json:"percents"`
	Parent   *string     `json:"parent"`
}

// IncomingProfile is the raw XHProf payload.
type IncomingProfile struct {
	Profile  map[string]Metrics `json:"profile"`
	AppName  string             `json:"app_name"`
	Hostname string             `json:"hostname"`
	Date     int64              `json:"date"`
	Tags     map[string]any     `json:"tags,omitempty"`
}
