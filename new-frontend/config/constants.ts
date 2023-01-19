export const EVENT_TYPES = {
  VAR_DUMP: "var-dump",
  SMTP: "Smtp",
  SENTRY: "Sentry",
  PROFILER: "profiler",
  MONOLOG: "monolog",
  INSPECTOR: "inspector",
  RAY: "ray",
};


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
