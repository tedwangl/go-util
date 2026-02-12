package zapx

type (
	LogConf struct {
		ServiceName         string       `json:",optional"`
		Mode                string       `json:",default=console,options=[console,file,volume]"`
		Encoding            string       `json:",default=json,options=[json,console]"`
		TimeFormat          string       `json:",optional"`
		Path                string       `json:",default=logs"`
		Level               string       `json:",default=info,options=[debug,info,error,severe]"`
		MaxContentLength    uint32       `json:",optional"`
		Compress            bool         `json:",optional"`
		KeepDays            int          `json:",optional"`
		StackCooldownMillis int          `json:",default=100"`
		MaxBackups          int          `json:",default=0"`
		MaxSize             int          `json:",default=0"`
		Rotation            string       `json:",default=daily,options=[daily,size]"`
		FileTimeFormat      string       `json:",optional"`
		FieldKeys           fieldKeyConf `json:",optional"`
		Development         bool         `json:",optional"`
		CallerSkip          int          `json:",default=2"`
		CollectSysLog       bool         `json:",optional"`
		SysLogLevel         string       `json:",default=info,options=[debug,info,error,severe]"`
	}

	fieldKeyConf struct {
		CallerKey    string `json:",default=caller"`
		ContentKey   string `json:",default=content"`
		DurationKey  string `json:",default=duration"`
		LevelKey     string `json:",default=level"`
		SpanKey      string `json:",default=span"`
		TimestampKey string `json:",default=@timestamp"`
		TraceKey     string `json:",default=trace"`
		TruncatedKey string `json:",default=truncated"`
	}
)
