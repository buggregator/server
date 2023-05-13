import { Meta, Story } from "@storybook/vue3";
import ExceptionPayload from '~/components/RayDumpPreview/RayTypesPreview/ExceptionPayload.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import eventMock from '~/mocks/ray-exception.json'

export default {
  title: "RayDump/Types/Exception",
  component: ExceptionPayload
} as Meta<typeof ExceptionPayload>;

const Template: Story = (args) => ({
  components: { ExceptionPayload },
  setup() {
    return {
      args,
    };
  },
  template: `<ExceptionPayload v-bind="args" />`,
});

export const Exception = Template.bind({});
Exception.args = {payload: normalizeRayDumpEvent(eventMock).payload.payload.payloads[0]};
