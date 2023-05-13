import { Meta, Story } from "@storybook/vue3";
import LogPayload from '~/components/RayDumpPreview/RayTypesPreview/LogPayload.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import eventMock from '~/mocks/ray-color.json'

export default {
  title: "RayDump/Types/Log",
  component: LogPayload
} as Meta<typeof LogPayload>;

const Template: Story = (args) => ({
  components: { LogPayload },
  setup() {
    return {
      args,
    };
  },
  template: `<LogPayload v-bind="args" />`,
});

export const Log = Template.bind({});
Log.args = {payload: normalizeRayDumpEvent(eventMock).payload.payload.payloads[0]};
