<script lang="ts">
import { defineComponent, h, PropType } from "vue";
import { RayPayload } from "~/config/types";
import { RAY_EVENT_TYPES } from "~/config/constants";

import LogPayload from "~/components/RayDumpPreview/RayTypesPreview/LogPayload.vue";
import CustomPayload from "~/components/RayDumpPreview/RayTypesPreview/CustomPayload.vue";
import CallerPayload from "~/components/RayDumpPreview/RayTypesPreview/CallerPayload.vue";
import CarbonPayload from "~/components/RayDumpPreview/RayTypesPreview/CarbonPayload.vue";
import TracePayload from "~/components/RayDumpPreview/RayTypesPreview/TracePayload.vue";
import ExceptionPayload from "~/components/RayDumpPreview/RayTypesPreview/ExceptionPayload.vue";
import TablePayload from "~/components/RayDumpPreview/RayTypesPreview/TablePayload.vue";
import MeasurePayload from "~/components/RayDumpPreview/RayTypesPreview/MeasurePayload.vue";
import NotifyPayload from "~/components/RayDumpPreview/RayTypesPreview/NotifyPayload.vue";
import PreviewFallback from "~/components/PreviewFallback/PreviewFallback.vue";

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
        h(LogPayload, { payload }),
      [RAY_EVENT_TYPES.CUSTOM]: (payload: RayPayload) =>
        h(CustomPayload, { payload }),
      [RAY_EVENT_TYPES.CALLER]: (payload: RayPayload) =>
        h(CallerPayload, { payload }),
      [RAY_EVENT_TYPES.CARBON]: (payload: RayPayload) =>
        h(CarbonPayload, { payload }),
      [RAY_EVENT_TYPES.TRACE]: (payload: RayPayload) =>
        h(TracePayload, { payload }),
      [RAY_EVENT_TYPES.EXCEPTION]: (payload: RayPayload) =>
        h(ExceptionPayload, { payload }),
      [RAY_EVENT_TYPES.TABLE]: (payload: RayPayload) =>
        h(TablePayload, { payload }),
      [RAY_EVENT_TYPES.MEASURE]: (payload: RayPayload) =>
        h(MeasurePayload, { payload }),
      [RAY_EVENT_TYPES.NOTIFY]: (payload: RayPayload) =>
        h(NotifyPayload, { payload }),
    };

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
