export const EVENT_TYPES = {
  VAR_DUMP: "var-dump",
  SMTP: "smtp",
  SENTRY: "sentry",
  PROFILER: "profiler",
  MONOLOG: "monolog",
  INSPECTOR: "inspector",
  HTTP_DUMP: "http-dump",
  RAY_DUMP: "ray",
};

export const RAY_EVENT_TYPES = {
  LOG: "log",
  //SIZE: "size",
  CUSTOM: "custom",
  //LABEL: "label",
  CALLER: "caller",
  CARBON: "carbon",
  //COLOR: "color",
  EXCEPTION: "exception",
  //HIDE: "hide",
  MEASURE: "measure",
  //NOTIFY: "notify",
  TABLE: "table",
  TRACE: "trace",
  QUERY: "executed_query",
  ELOQUENT: "eloquent_model",
  VIEW: "view",
  EVENT: "event",
  JOB: "job_event",
  LOCK: "create_lock",
}

// TODO: colors should depends on level for some tools
export const EVENT_STATUS_COLOR_MAP = {
  CRITICAL: "red",
  ERROR: "red",
  ALERT: "red",
  EMERGENCY: "red",
  WARNING: "orange",
  INFO: "blue",
  NOTICE: "blue",
  DEBUG: "gray",
  SUCCESS: "green",
};
