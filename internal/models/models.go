package models

// Variable represents a key-value variable with optional description.
type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// Header represents an HTTP header.
type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// QueryParam represents a URL query parameter.
type QueryParam struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// FormField represents a field in multipart/form-data body.
type FormField struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"` // "text" or "file"
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// Request represents an API request configuration.
type Request struct {
	ID                          string       `json:"id"`
	Type                        string       `json:"type"` // "http" or "websocket"
	Name                        string       `json:"name"`
	Method                      string       `json:"method"`
	URL                         string       `json:"url"`
	Headers                     []Header     `json:"headers"`
	Query                       []QueryParam `json:"query"`
	BodyType                    string       `json:"bodyType"` // "none", "json", "text", "form-data", "x-www-form-urlencoded"
	Body                        string       `json:"body"`
	FormData                    []FormField  `json:"formData"`
	PreRequestScript            string       `json:"preRequestScript"`
	PostResponseScript          string       `json:"postResponseScript"`
	Description                 string       `json:"description"`
	DisabledInheritedHeaderKeys []string     `json:"disabledInheritedHeaderKeys"`
}

// DirMeta represents directory-level metadata.
type DirMeta struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Headers   []Header   `json:"headers"`
	Variables []Variable `json:"variables"`
}

// Settings represents global application settings.
type Settings struct {
	Version          string     `json:"version"`
	Theme            string     `json:"theme"`
	RequestTimeoutMs int        `json:"requestTimeoutMs"`
	GlobalHeaders    []Header   `json:"globalHeaders"`
	GlobalVariables  []Variable `json:"globalVariables"`
}

// WorkspaceMigrationResult summarizes a workspace path change.
type WorkspaceMigrationResult struct {
	SourcePath     string   `json:"sourcePath"`
	TargetPath     string   `json:"targetPath"`
	CopiedFiles    int      `json:"copiedFiles"`
	SkippedFiles   int      `json:"skippedFiles"`
	CreatedDirs    int      `json:"createdDirs"`
	FailedFiles    int      `json:"failedFiles"`
	FailureSamples []string `json:"failureSamples"`
	Message        string   `json:"message"`
}

// HttpRequest is the runtime request sent from frontend to backend.
type HttpRequest struct {
	Request
	ResolvedURL     string            `json:"resolvedUrl"`
	ResolvedHeaders []Header          `json:"resolvedHeaders"`
	ResolvedBody    []byte            `json:"resolvedBody"`
	GlobalVariables map[string]string `json:"globalVariables"`
}

// HttpResponse is the runtime response returned from backend.
type HttpResponse struct {
	StatusCode     int                 `json:"statusCode"`
	Status         string              `json:"status"`
	Headers        map[string][]string `json:"headers"`
	Cookies        []string            `json:"cookies"`
	Body           string              `json:"body"`
	TimeMs         int64               `json:"timeMs"`
	ServerTimeMs   int64               `json:"serverTimeMs"`
	TransferTimeMs int64               `json:"transferTimeMs"`
	SizeBytes      int64               `json:"sizeBytes"`
	ContentType    string              `json:"contentType"`
	PreScript      *ScriptResult       `json:"preScript"`
	PostScript     *ScriptResult       `json:"postScript"`
}

// ScriptResult holds the result of a pre/post script execution.
type ScriptResult struct {
	Success    bool     `json:"success"`
	Error      string   `json:"error,omitempty"`
	DurationMs int64    `json:"durationMs"`
	Logs       []string `json:"logs"`
}

// ApiNode is a tree node for workspace UI.
type ApiNode struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "dir" or "request"
	Name        string    `json:"name"`
	Method      string    `json:"method,omitempty"`
	RequestType string    `json:"requestType,omitempty"`
	Path        string    `json:"path"`
	URL         string    `json:"url,omitempty"`
	Children    []ApiNode `json:"children,omitempty"`
}

// FetchParseResult holds parsed fetch call information.
type FetchParseResult struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Query   []QueryParam      `json:"query"`
	Body    string            `json:"body"`
}

// RequestPreview shows the resolved request for UI preview.
type RequestPreview struct {
	URL         string       `json:"url"`
	Headers     []Header     `json:"headers"`
	Query       []QueryParam `json:"query"`
	BodyType    string       `json:"bodyType"`
	Body        string       `json:"body"`
	FormData    []FormField  `json:"formData"`
	ContentType string       `json:"contentType"`
}

// LoadTestMode defines how a load test stops.
type LoadTestMode string

const (
	LoadTestModeRounds   LoadTestMode = "rounds"
	LoadTestModeDuration LoadTestMode = "duration"
)

// LoadTestConfig holds load test settings from the UI.
type LoadTestConfig struct {
	Concurrency     int    `json:"concurrency"`
	Mode            string `json:"mode"`
	Rounds          int    `json:"rounds"`
	DurationSeconds int    `json:"durationSeconds"`
	LogEnabled      bool   `json:"logEnabled"`
}

// LoadTestProgress is emitted during a running load test.
type LoadTestProgress struct {
	ID            string  `json:"id"`
	Completed     int64   `json:"completed"`
	Total         int64   `json:"total"`
	Percent       float64 `json:"percent"`
	RPS           float64 `json:"rps"`
	DownloadSpeed float64 `json:"downloadSpeed"` // bytes per second
	Running       bool    `json:"running"`
}

// LoadTestResult is emitted when a load test finishes.
type LoadTestResult struct {
	TotalRequests int64   `json:"totalRequests"`
	DurationSec   float64 `json:"durationSec"`
	SuccessCount  int64   `json:"successCount"`
	FailCount     int64   `json:"failCount"`
	ErrorRate     float64 `json:"errorRate"`
	TotalBytesMB  float64 `json:"totalBytesMB"`
	RPS           float64 `json:"rps"`
	SuccessRPS    float64 `json:"successRps"`
	BytesPerSecMB float64 `json:"bytesPerSecMB"`
	MaxTimeMs     int64   `json:"maxTimeMs"`
	MinTimeMs     int64   `json:"minTimeMs"`
	AvgTimeMs     float64 `json:"avgTimeMs"`
}
