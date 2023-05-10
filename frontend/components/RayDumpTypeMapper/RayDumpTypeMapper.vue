<script lang="ts">
import {defineComponent, h, PropType} from "vue";
import {RayPayload,} from "~/config/types";
import {RAY_EVENT_TYPES} from "~/config/constants";

import LogPayload from "~/components/RayDumpPreview/LogPayload.vue";

export default defineComponent({
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  render() {
    const TYPE_RENDER_MAP = {
      [RAY_EVENT_TYPES.LOG]: (payload: RayPayload) => h(LogPayload, {payload}),
    };

    if (Object.values(RAY_EVENT_TYPES).includes(this.payload.type)) {
      const renderFunction = TYPE_RENDER_MAP[this.payload.type];

      if (renderFunction) {
        return renderFunction(this.payload);
      }
    }
  },
});
</script>
