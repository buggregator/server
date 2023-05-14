<script lang="ts">
import {defineComponent, h, PropType} from "vue";
import {RayPayload} from "~/config/types";
import {RAY_EVENT_TYPES} from "~/config/constants";

import LogPayload from "~/components/RayDumpPreview/RayTypesPreview/LogPayload.vue";
import CustomPayload from "~/components/RayDumpPreview/RayTypesPreview/CustomPayload.vue";
import CallerPayload from "~/components/RayDumpPreview/RayTypesPreview/CallerPayload.vue";
import CarbonPayload from "~/components/RayDumpPreview/RayTypesPreview/CarbonPayload.vue";
import TracePayload from "~/components/RayDumpPreview/RayTypesPreview/TracePayload.vue";
import ExceptionPayload from "~/components/RayDumpPreview/RayTypesPreview/ExceptionPayload.vue";
import TablePayload from "~/components/RayDumpPreview/RayTypesPreview/TablePayload.vue";
import MeasurePayload from "~/components/RayDumpPreview/RayTypesPreview/MeasurePayload.vue";
import QueryPayload from "~/components/RayDumpPreview/RayTypesPreview/QueryPayload.vue";
import EloquentPayload from "~/components/RayDumpPreview/RayTypesPreview/EloquentPayload.vue";
import ViewsPayload from "~/components/RayDumpPreview/RayTypesPreview/ViewsPayload.vue";
import EventPayload from "~/components/RayDumpPreview/RayTypesPreview/EventPayload.vue";
import JobPayload from "~/components/RayDumpPreview/RayTypesPreview/JobPayload.vue";
import LockPayload from "~/components/RayDumpPreview/RayTypesPreview/LockPayload.vue";

export default defineComponent({
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  render() {
    const TYPE_RENDER_MAP = {
      [RAY_EVENT_TYPES.LOG]: (payload: RayPayload) =>
        h(LogPayload, {payload}),
      [RAY_EVENT_TYPES.CUSTOM]: (payload: RayPayload) =>
        h(CustomPayload, {payload}),
      [RAY_EVENT_TYPES.CALLER]: (payload: RayPayload) =>
        h(CallerPayload, {payload}),
      [RAY_EVENT_TYPES.CARBON]: (payload: RayPayload) =>
        h(CarbonPayload, {payload}),
      [RAY_EVENT_TYPES.TRACE]: (payload: RayPayload) =>
        h(TracePayload, {payload}),
      [RAY_EVENT_TYPES.EXCEPTION]: (payload: RayPayload) =>
        h(ExceptionPayload, {payload}),
      [RAY_EVENT_TYPES.TABLE]: (payload: RayPayload) =>
        h(TablePayload, {payload}),
      [RAY_EVENT_TYPES.MEASURE]: (payload: RayPayload) =>
        h(MeasurePayload, {payload}),
      [RAY_EVENT_TYPES.QUERY]: (payload: RayPayload) =>
        h(QueryPayload, {payload}),
      [RAY_EVENT_TYPES.ELOQUENT]: (payload: RayPayload) =>
        h(EloquentPayload, {payload}),
      [RAY_EVENT_TYPES.VIEW]: (payload: RayPayload) =>
        h(ViewsPayload, {payload}),
      [RAY_EVENT_TYPES.EVENT]: (payload: RayPayload) =>
        h(EventPayload, {payload}),
      [RAY_EVENT_TYPES.JOB]: (payload: RayPayload) =>
        h(JobPayload, {payload}),
      [RAY_EVENT_TYPES.LOCK]: (payload: RayPayload) =>
        h(LockPayload, {payload}),
    };

    if (this.payload.type === 'hide') {
      this.$emit("toggleView", true);
    }

    if (Object.values(RAY_EVENT_TYPES).includes(this.payload.type)) {
      const renderFunction = TYPE_RENDER_MAP[this.payload.type];

      if (renderFunction) {
        return renderFunction(this.payload);
      }
    }
    return null;
  },
});
</script>
