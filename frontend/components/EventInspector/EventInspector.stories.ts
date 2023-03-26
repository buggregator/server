import { Meta, Story } from "@storybook/vue3";
import { normalizeInspectorEvent } from "~/utils/normalize-event";
import inspectorEventMock from '~/mocks/inspector.json'
import EventInspector from './EventInspector.vue';

export default {
  title: "Inspector",
  component: EventInspector
} as Meta<typeof EventInspector>;

const Template: Story = (args) => ({
  components: { EventInspector },
  setup() {
    return {
      args,
    };
  },
  template: `<event-inspector v-bind="args" />`,
});

export const Event = Template.bind({});

Event.args = {
  event: normalizeInspectorEvent(inspectorEventMock),
};
