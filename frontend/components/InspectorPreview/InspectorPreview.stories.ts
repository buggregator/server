import { Meta, Story } from "@storybook/vue3";
import { normalizeInspectorEvent } from "~/utils/normalize-event";
import inspectorEventMock from '~/mocks/inspector.json'
import InspectorPreview from '~/components/InspectorPreview/InspectorPreview.vue';

export default {
  title: "Inspector/Components/InspectorPreview",
  component: InspectorPreview
} as Meta<typeof InspectorPreview>;

const Template: Story = (args) => ({
  components: { InspectorPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<InspectorPreview v-bind="args" />`,
});

export const Event = Template.bind({});

Event.args = {
  event: normalizeInspectorEvent(inspectorEventMock),
};
