export const EVENT_TYPES = {
  VAR_DUMP: "var-dump",
  SMTP: "smtp",
  SENTRY: "sentry",
  PROFILER: "profiler",
  MONOLOG: "monolog",
  INSPECTOR: "inspector",
  HTTP_DUMP: "http-dump",
  RAY: "ray",
};

export const RAY_EVENT_TYPES = {
  LOG: "log",
  SIZE: "size",
  CUSTOM: "custom",
  LABEL: "label",
  CALLER: "caller",
  CARBON: "carbon",
  COLOR: "color",
  EXCEPTION: "exception",
  HIDE: "hide",
  MEASURE: "measure",
  NOTIFY: "notify",
  TABLE: "table",
  TRACE: "trace",
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
