import { Meta, Story } from "@storybook/vue3";
import CustomPayload from '~/components/RayDumpPreview/RayTypesPreview/CustomPayload.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import eventMock from '~/mocks/ray-text.json'

export default {
  title: "RayDump/Types/Custom",
  component: CustomPayload
} as Meta<typeof CustomPayload>;

const Template: Story = (args) => ({
  components: { CustomPayload },
  setup() {
    return {
      args,
    };
  },
  template: `<CustomPayload v-bind="args" />`,
});

export const Custom = Template.bind({});
Custom.args = {payload: normalizeRayDumpEvent(eventMock).payload.payload.payloads[0]};
