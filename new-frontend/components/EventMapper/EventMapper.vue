<script lang="ts">
import { defineComponent, PropType, h } from "vue";
import { Monolog, Sentry, ServerEvent, VarDump, SMTP } from "~/config/types";
import EventMonolog from "~/components/EventMonolog/EventMonolog.vue";
import EventSentry from "~/components/EventSentry/EventSentry.vue";
import EventVarDump from "~/components/EventVarDump/EventVarDump.vue";
import EventSmtp from "~/components/EventSmtp/EventSmtp.vue";
import EventProfiler from "~/components/EventProfiler/EventProfiler.vue";
import EventFallback from "~/components/EventFallback/EventFallback.vue";
import { EVENT_TYPES } from "~/config/constants";
import {
  normalizeMonologEvent,
  normalizeSMTPEvent,
  normalizeVarDumpEvent,
  normalizeSentryEvent,
  normalizeInspectorEvent,
  normalizeProfilerEvent,
  normalizeFallbackEvent,
} from "~/utils/normalizeEvent";

export default defineComponent({
  props: {
    event: {
      type: Object as PropType<ServerEvent<unknown>>,
      required: true,
    },
  },
  render() {
    const EVENT_TYPE_RENDER_MAP = {
      [EVENT_TYPES.SENTRY]: (event: ServerEvent<Sentry>) =>
        h(EventSentry, { event: normalizeSentryEvent(event) }),
      [EVENT_TYPES.MONOLOG]: (event: ServerEvent<Monolog>) =>
        h(EventMonolog, { event: normalizeMonologEvent(event) }),
      [EVENT_TYPES.VAR_DUMP]: (event: ServerEvent<VarDump>) =>
        h(EventVarDump, { event: normalizeVarDumpEvent(event) }),
      [EVENT_TYPES.SMTP]: (event: ServerEvent<SMTP>) =>
        h(EventSmtp, { event: normalizeSMTPEvent(event) }),
      [EVENT_TYPES.PROFILER]: (event: ServerEvent<unknown>) =>
        h(EventProfiler, { event: normalizeProfilerEvent(event) }),
      [EVENT_TYPES.INSPECTOR]: (event: ServerEvent<unknown>) =>
        h(EventFallback, { event: normalizeInspectorEvent(event) }),
    };

    if (Object.values(EVENT_TYPES).includes(this.event.type)) {
      const renderFunction = EVENT_TYPE_RENDER_MAP[this.event.type];

      if (renderFunction) {
        return renderFunction(this.event);
      }
    }

    return h(EventFallback, { event: normalizeFallbackEvent(this.event) });
  },
});
</script>
