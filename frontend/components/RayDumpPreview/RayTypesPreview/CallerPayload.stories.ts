import { Meta, Story } from "@storybook/vue3";
import CallerPayload from '~/components/RayDumpPreview/RayTypesPreview/CallerPayload.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import rayCallerEventMock from '~/mocks/ray-caller.json'

export default {
  title: "RayDump/Types/Caller",
  component: CallerPayload
} as Meta<typeof CallerPayload>;

const Template: Story = (args) => ({
  components: { CallerPayload },
  setup() {
    return {
      args,
    };
  },
  template: `<CallerPayload v-bind="args" />`,
});

export const Caller = Template.bind({});
Caller.args = {payload: normalizeRayDumpEvent(rayCallerEventMock).payload.payload.payloads[0]};
